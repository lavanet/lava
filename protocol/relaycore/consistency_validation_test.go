package relaycore

import (
	"testing"
	"time"

	"github.com/lavanet/lava/v5/protocol/lavaprotocol/protocolerrors"
	spectypes "github.com/lavanet/lava/v5/types/spec"
	"github.com/stretchr/testify/require"
)

// TestShouldSkipConsistencyValidation tests all cases for skipping validation
func TestShouldSkipConsistencyValidation(t *testing.T) {
	tests := []struct {
		name           string
		requestedBlock int64
		expectedSkip   bool
	}{
		{
			name:           "NOT_APPLICABLE should skip",
			requestedBlock: spectypes.NOT_APPLICABLE,
			expectedSkip:   true,
		},
		{
			name:           "LATEST_BLOCK should not skip",
			requestedBlock: spectypes.LATEST_BLOCK,
			expectedSkip:   false,
		},
		{
			name:           "EARLIEST_BLOCK should skip",
			requestedBlock: spectypes.EARLIEST_BLOCK,
			expectedSkip:   true,
		},
		{
			name:           "PENDING_BLOCK should skip",
			requestedBlock: spectypes.PENDING_BLOCK,
			expectedSkip:   true,
		},
		{
			name:           "SAFE_BLOCK should skip",
			requestedBlock: spectypes.SAFE_BLOCK,
			expectedSkip:   true,
		},
		{
			name:           "FINALIZED_BLOCK should skip",
			requestedBlock: spectypes.FINALIZED_BLOCK,
			expectedSkip:   true,
		},
		{
			name:           "specific historical block (positive) should skip",
			requestedBlock: 12345,
			expectedSkip:   true,
		},
		{
			name:           "block 0 should skip",
			requestedBlock: 0,
			expectedSkip:   true,
		},
		{
			name:           "block 1 should skip",
			requestedBlock: 1,
			expectedSkip:   true,
		},
		{
			name:           "unknown negative value should not skip",
			requestedBlock: -100,
			expectedSkip:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldSkipConsistencyValidation(tt.requestedBlock)
			require.Equal(t, tt.expectedSkip, result, "unexpected skip result for requestedBlock=%d", tt.requestedBlock)
		})
	}
}

// TestValidateEndpointCapability_Pass tests cases where endpoint is capable
func TestValidateEndpointCapability_Pass(t *testing.T) {
	config := &ConsistencyValidationConfig{
		EndpointLagThreshold: 10,
	}

	tests := []struct {
		name                string
		endpointLatestBlock int64
		seenBlock           int64
		requestedBlock      int64
	}{
		{
			name:                "endpoint ahead of seen block",
			endpointLatestBlock: 1100,
			seenBlock:           1000,
			requestedBlock:      spectypes.LATEST_BLOCK,
		},
		{
			name:                "endpoint equal to seen block",
			endpointLatestBlock: 1000,
			seenBlock:           1000,
			requestedBlock:      spectypes.LATEST_BLOCK,
		},
		{
			name:                "endpoint lag within threshold",
			endpointLatestBlock: 995,
			seenBlock:           1000,
			requestedBlock:      spectypes.LATEST_BLOCK,
		},
		{
			name:                "endpoint lag exactly at threshold",
			endpointLatestBlock: 990,
			seenBlock:           1000,
			requestedBlock:      spectypes.LATEST_BLOCK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEndpointCapability(tt.endpointLatestBlock, tt.seenBlock, tt.requestedBlock, config)
			require.NoError(t, err)
		})
	}
}

// TestValidateEndpointCapability_TooFarBehind tests endpoints that are too far behind
func TestValidateEndpointCapability_TooFarBehind(t *testing.T) {
	config := &ConsistencyValidationConfig{
		EndpointLagThreshold: 10,
	}

	tests := []struct {
		name                string
		endpointLatestBlock int64
		seenBlock           int64
		requestedBlock      int64
	}{
		{
			name:                "lag exceeds threshold by 1",
			endpointLatestBlock: 989,
			seenBlock:           1000,
			requestedBlock:      spectypes.LATEST_BLOCK,
		},
		{
			name:                "lag significantly exceeds threshold",
			endpointLatestBlock: 500,
			seenBlock:           1000,
			requestedBlock:      spectypes.LATEST_BLOCK,
		},
		{
			name:                "very large lag",
			endpointLatestBlock: 100,
			seenBlock:           10000,
			requestedBlock:      spectypes.LATEST_BLOCK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEndpointCapability(tt.endpointLatestBlock, tt.seenBlock, tt.requestedBlock, config)
			require.Error(t, err)
			require.True(t, protocolerrors.ConsistencyError.Is(err), "error should be ConsistencyError")
		})
	}
}

// TestValidateEndpointCapability_Skip tests cases where validation is skipped
func TestValidateEndpointCapability_Skip(t *testing.T) {
	config := &ConsistencyValidationConfig{
		EndpointLagThreshold: 10,
	}

	tests := []struct {
		name                string
		endpointLatestBlock int64
		seenBlock           int64
		requestedBlock      int64
		reason              string
	}{
		{
			name:                "no seen block requirement",
			endpointLatestBlock: 500,
			seenBlock:           0,
			requestedBlock:      spectypes.LATEST_BLOCK,
			reason:              "seenBlock is 0",
		},
		{
			name:                "endpoint block unknown",
			endpointLatestBlock: 0,
			seenBlock:           1000,
			requestedBlock:      spectypes.LATEST_BLOCK,
			reason:              "endpointLatestBlock is 0",
		},
		{
			name:                "historical block request",
			endpointLatestBlock: 500,
			seenBlock:           1000,
			requestedBlock:      12345,
			reason:              "historical block request",
		},
		{
			name:                "NOT_APPLICABLE request",
			endpointLatestBlock: 500,
			seenBlock:           1000,
			requestedBlock:      spectypes.NOT_APPLICABLE,
			reason:              "NOT_APPLICABLE request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEndpointCapability(tt.endpointLatestBlock, tt.seenBlock, tt.requestedBlock, config)
			require.NoError(t, err, "validation should be skipped for: %s", tt.reason)
		})
	}
}

// TestValidateEndpointCapability_NilConfig tests behavior with nil config
func TestValidateEndpointCapability_NilConfig(t *testing.T) {
	// Should not error even with endpoint far behind when config is nil
	err := ValidateEndpointCapability(500, 1000, spectypes.LATEST_BLOCK, nil)
	require.NoError(t, err, "nil config should skip validation")
}

// TestConsistencyValidationConfig_Creation tests config creation
func TestConsistencyValidationConfig_Creation(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		config := DefaultConsistencyValidationConfig()
		require.NotNil(t, config)
		require.Equal(t, int64(10), config.EndpointLagThreshold)
		require.False(t, config.EnableWaitForCatchup)
		require.Equal(t, 500*time.Millisecond, config.MaxWaitTime)
	})

	t.Run("config from chain spec - typical EVM", func(t *testing.T) {
		// Typical EVM: blockLagForQosSync=3, finalization=64, avgBlockTime=12s
		config := NewConsistencyValidationConfig(3, 64, 12*time.Second)
		require.NotNil(t, config)
		// EndpointLagThreshold = max(3*2=6, 64) = 64, but min 10, so 64
		require.Equal(t, int64(64), config.EndpointLagThreshold)
		// MaxWaitTime = min(12s*2=24s, 5s cap) = 5s
		require.Equal(t, 5*time.Second, config.MaxWaitTime)
	})

	t.Run("config from chain spec - fast chain", func(t *testing.T) {
		// Fast chain: blockLagForQosSync=10, finalization=32, avgBlockTime=2s
		config := NewConsistencyValidationConfig(10, 32, 2*time.Second)
		require.NotNil(t, config)
		// EndpointLagThreshold = max(10*2=20, 32) = 32
		require.Equal(t, int64(32), config.EndpointLagThreshold)
		// MaxWaitTime = 2s*2 = 4s
		require.Equal(t, 4*time.Second, config.MaxWaitTime)
	})

	t.Run("config from chain spec - slow chain", func(t *testing.T) {
		// Slow chain: blockLagForQosSync=1, finalization=6, avgBlockTime=60s
		config := NewConsistencyValidationConfig(1, 6, 60*time.Second)
		require.NotNil(t, config)
		// EndpointLagThreshold = max(1*2=2, 6) = 10 (minimum)
		require.Equal(t, int64(10), config.EndpointLagThreshold)
		// MaxWaitTime = min(60s*2=120s, 5s cap) = 5s
		require.Equal(t, 5*time.Second, config.MaxWaitTime)
	})

	t.Run("config from chain spec - very fast block time", func(t *testing.T) {
		// Very fast: blockLagForQosSync=20, finalization=100, avgBlockTime=100ms
		config := NewConsistencyValidationConfig(20, 100, 100*time.Millisecond)
		require.NotNil(t, config)
		// EndpointLagThreshold = max(20*2=40, 100) = 100
		require.Equal(t, int64(100), config.EndpointLagThreshold)
		// MaxWaitTime = max(100ms*2=200ms, 500ms floor) = 500ms
		require.Equal(t, 500*time.Millisecond, config.MaxWaitTime)
	})
}

// TestConsistencyValidationConfig_Methods tests config helper methods
func TestConsistencyValidationConfig_Methods(t *testing.T) {
	config := &ConsistencyValidationConfig{
		EndpointLagThreshold: 10,
	}

	t.Run("IsEndpointTooFarBehind", func(t *testing.T) {
		require.False(t, config.IsEndpointTooFarBehind(0))
		require.False(t, config.IsEndpointTooFarBehind(10))
		require.True(t, config.IsEndpointTooFarBehind(11))
		require.True(t, config.IsEndpointTooFarBehind(100))
	})

	t.Run("nil config safety", func(t *testing.T) {
		var nilConfig *ConsistencyValidationConfig
		require.False(t, nilConfig.IsEndpointTooFarBehind(100))
	})
}
