package relaycore

import (
	"testing"
	"time"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/stretchr/testify/require"
)

// mockChainParserForEstimator wraps a real ChainParser but allows overriding ChainBlockStats
type mockChainParserForEstimator struct {
	chainlib.ChainParser          // Embed to satisfy interface
	allowedBlockLagForQosSync     int64
	averageBlockTime              time.Duration
	blockDistanceForFinalizedData uint32
	blocksInFinalizationProof     uint32
}

func (m *mockChainParserForEstimator) ChainBlockStats() (int64, time.Duration, uint32, uint32) {
	return m.allowedBlockLagForQosSync, m.averageBlockTime, m.blockDistanceForFinalizedData, m.blocksInFinalizationProof
}

func newMockChainParser() *mockChainParserForEstimator {
	return &mockChainParserForEstimator{
		ChainParser:                   nil, // Will be nil but methods are overridden
		allowedBlockLagForQosSync:     10,
		averageBlockTime:              6 * time.Second,
		blockDistanceForFinalizedData: 15,
		blocksInFinalizationProof:     5,
	}
}

func TestLatestBlockEstimator_NoObservations(t *testing.T) {
	estimator := NewLatestBlockEstimator()
	parser := newMockChainParser()

	expectedBlockHeight, numProviders := estimator.Estimate(parser)

	require.Equal(t, int64(0), expectedBlockHeight, "should return 0 with no observations")
	require.Equal(t, 0, numProviders, "should return 0 providers")
}

func TestLatestBlockEstimator_NilEstimator(t *testing.T) {
	var estimator *LatestBlockEstimator
	parser := newMockChainParser()

	expectedBlockHeight, numProviders := estimator.Estimate(parser)

	require.Equal(t, int64(0), expectedBlockHeight, "nil estimator should return 0")
	require.Equal(t, 0, numProviders, "nil estimator should return 0 providers")
}

func TestLatestBlockEstimator_NilParser(t *testing.T) {
	estimator := NewLatestBlockEstimator()

	expectedBlockHeight, numProviders := estimator.Estimate(nil)

	require.Equal(t, int64(0), expectedBlockHeight, "nil parser should return 0")
	require.Equal(t, 0, numProviders, "nil parser should return 0 providers")
}

func TestLatestBlockEstimator_SingleProvider(t *testing.T) {
	estimator := NewLatestBlockEstimator()
	parser := newMockChainParser()

	// Record observation from one provider
	estimator.Record("provider1", 1000)

	expectedBlockHeight, numProviders := estimator.Estimate(parser)

	// Expected calculation:
	// latestFinalizedBlock = 1000 - 15 = 985
	// interpolation = 0 (just recorded, time diff ~0)
	// medianOfExpectedBlocks = 985
	// providersMedianOfLatestBlock = 985 + 15 = 1000
	// expectedBlockHeight = 1000 - 10 = 990

	require.Equal(t, int64(990), expectedBlockHeight, "should calculate correctly with single provider")
	require.Equal(t, 1, numProviders, "should report 1 provider")
}

func TestLatestBlockEstimator_TwoProviders(t *testing.T) {
	estimator := NewLatestBlockEstimator()
	parser := newMockChainParser()

	// Record observations from two providers
	estimator.Record("provider1", 1000)
	estimator.Record("provider2", 1010)

	expectedBlockHeight, numProviders := estimator.Estimate(parser)

	// Median of [985, 995] = 990
	// providersMedianOfLatestBlock = 990 + 15 = 1005
	// expectedBlockHeight = 1005 - 10 = 995

	require.Equal(t, int64(995), expectedBlockHeight, "should use median of two providers")
	require.Equal(t, 2, numProviders, "should report 2 providers")
}

func TestLatestBlockEstimator_MultipleProviders_Median(t *testing.T) {
	estimator := NewLatestBlockEstimator()
	parser := newMockChainParser()

	// Record observations from multiple providers with varying blocks
	estimator.Record("provider1", 1000)
	estimator.Record("provider2", 1010)
	estimator.Record("provider3", 1005)
	estimator.Record("provider4", 990)
	estimator.Record("provider5", 1015)

	expectedBlockHeight, numProviders := estimator.Estimate(parser)

	// Finalized blocks: [985, 995, 990, 975, 1000]
	// Median = 990
	// providersMedianOfLatestBlock = 990 + 15 = 1005
	// expectedBlockHeight = 1005 - 10 = 995

	require.Equal(t, int64(995), expectedBlockHeight, "should calculate median correctly")
	require.Equal(t, 5, numProviders, "should report 5 providers")
}

func TestLatestBlockEstimator_ClockSkew_FutureTime(t *testing.T) {
	estimator := NewLatestBlockEstimator()
	parser := newMockChainParser()

	// Manually set observation with future timestamp
	estimator.mu.Lock()
	futureTime := time.Now().Add(10 * time.Second)
	estimator.observations["provider1"] = providerBlockObservation{
		LatestBlock: 1000,
		LastUpdated: futureTime,
	}
	estimator.mu.Unlock()

	expectedBlockHeight, numProviders := estimator.Estimate(parser)

	// interpolateBlocks should return 0 for future timestamps
	// latestFinalizedBlock = 1000 - 15 = 985
	// interpolation = 0 (future time)
	// expectedBlockHeight = (985 + 15) - 10 = 990

	require.Equal(t, int64(990), expectedBlockHeight, "should handle future timestamps gracefully")
	require.Equal(t, 1, numProviders, "should still count the provider")
}

func TestLatestBlockEstimator_ClockSkew_OldObservation(t *testing.T) {
	estimator := NewLatestBlockEstimator()
	parser := newMockChainParser()

	// Record observation, then wait a bit
	estimator.Record("provider1", 1000)
	time.Sleep(100 * time.Millisecond)

	expectedBlockHeight, numProviders := estimator.Estimate(parser)

	// With averageBlockTime of 6s, 100ms should add ~0 blocks interpolation
	// But the calculation should still work
	require.GreaterOrEqual(t, expectedBlockHeight, int64(990), "should interpolate slightly forward")
	require.LessOrEqual(t, expectedBlockHeight, int64(991), "interpolation should be minimal")
	require.Equal(t, 1, numProviders, "should report 1 provider")
}

func TestLatestBlockEstimator_InterpolateBlocks(t *testing.T) {
	tests := []struct {
		name             string
		timeDiff         time.Duration
		averageBlockTime time.Duration
		expectedBlocks   int64
	}{
		{
			name:             "no time passed",
			timeDiff:         0,
			averageBlockTime: 6 * time.Second,
			expectedBlocks:   0,
		},
		{
			name:             "one block time",
			timeDiff:         6 * time.Second,
			averageBlockTime: 6 * time.Second,
			expectedBlocks:   1,
		},
		{
			name:             "ten blocks",
			timeDiff:         60 * time.Second,
			averageBlockTime: 6 * time.Second,
			expectedBlocks:   10,
		},
		{
			name:             "fractional block",
			timeDiff:         3 * time.Second,
			averageBlockTime: 6 * time.Second,
			expectedBlocks:   0, // Integer division rounds down
		},
		{
			name:             "future time (negative)",
			timeDiff:         -10 * time.Second,
			averageBlockTime: 6 * time.Second,
			expectedBlocks:   0,
		},
		{
			name:             "zero average block time",
			timeDiff:         10 * time.Second,
			averageBlockTime: 0,
			expectedBlocks:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			latest := now.Add(-tt.timeDiff)
			result := interpolateBlocks(now, latest, tt.averageBlockTime)
			require.Equal(t, tt.expectedBlocks, result)
		})
	}
}

func TestLatestBlockEstimator_UpdateSameProvider(t *testing.T) {
	estimator := NewLatestBlockEstimator()
	parser := newMockChainParser()

	// Record initial observation
	estimator.Record("provider1", 1000)

	// Update same provider with newer block
	estimator.Record("provider1", 1020)

	expectedBlockHeight, numProviders := estimator.Estimate(parser)

	// Should use the latest observation (1020), not accumulate both
	require.Equal(t, 1, numProviders, "should have only 1 provider (updated)")

	// latestFinalizedBlock = 1020 - 15 = 1005
	// expectedBlockHeight = (1005 + 15) - 10 = 1010
	require.Equal(t, int64(1010), expectedBlockHeight, "should use updated block height")
}

func TestLatestBlockEstimator_IgnoreInvalidInputs(t *testing.T) {
	estimator := NewLatestBlockEstimator()
	parser := newMockChainParser()

	// Try to record invalid data
	estimator.Record("", 1000)          // empty address
	estimator.Record("provider1", 0)    // zero block
	estimator.Record("provider2", -100) // negative block

	expectedBlockHeight, numProviders := estimator.Estimate(parser)

	require.Equal(t, int64(0), expectedBlockHeight, "should ignore invalid inputs")
	require.Equal(t, 0, numProviders, "should have no valid providers")
}

func TestLatestBlockEstimator_NegativeBlockAfterFinalization(t *testing.T) {
	estimator := NewLatestBlockEstimator()
	parser := newMockChainParser()

	// Record block that would be negative after subtracting finalization distance
	estimator.Record("provider1", 10) // 10 - 15 = -5

	expectedBlockHeight, numProviders := estimator.Estimate(parser)

	// latestFinalizedBlock should be clamped to 0
	// expectedBlockHeight = (0 + 15) - 10 = 5
	require.Equal(t, int64(5), expectedBlockHeight, "should handle negative finalized blocks")
	require.Equal(t, 1, numProviders, "should still count the provider")
}

func TestLatestBlockEstimator_ZeroBlockAfterSubtraction(t *testing.T) {
	estimator := NewLatestBlockEstimator()
	parser := newMockChainParser()

	// Record multiple providers where some have very low blocks
	estimator.Record("provider1", 5)   // 5 - 15 = -10 -> clamped to 0
	estimator.Record("provider2", 100) // 100 - 15 = 85
	estimator.Record("provider3", 200) // 200 - 15 = 185

	expectedBlockHeight, numProviders := estimator.Estimate(parser)

	// Finalized: [0, 85, 185], median = 85
	// providersMedianOfLatestBlock = 85 + 15 = 100
	// expectedBlockHeight = 100 - 10 = 90

	require.Equal(t, int64(90), expectedBlockHeight)
	require.Equal(t, 3, numProviders)
}

func TestLatestBlockEstimator_MedianStoredCorrectly(t *testing.T) {
	estimator := NewLatestBlockEstimator()
	parser := newMockChainParser()

	// First observation
	estimator.Record("provider1", 1000)
	estimator.Estimate(parser)

	prevMedian := estimator.prevMedian.Load()
	require.Equal(t, uint64(1000), prevMedian, "should store median after first estimate")

	// Add higher observation
	estimator.Record("provider2", 1100)
	estimator.Estimate(parser)

	newMedian := estimator.prevMedian.Load()
	require.Greater(t, newMedian, prevMedian, "median should increase with higher blocks")
}

func TestLatestBlockEstimator_JumpDetection(t *testing.T) {
	estimator := NewLatestBlockEstimator()
	parser := newMockChainParser()

	// Set initial median
	estimator.Record("provider1", 1000)
	estimator.Estimate(parser)

	// Jump by more than 1000 blocks (should trigger warning in logs)
	estimator.Record("provider2", 5000)
	estimator.Estimate(parser)

	// Should still calculate correctly despite the jump
	_, numProviders := estimator.Estimate(parser)
	require.Equal(t, 2, numProviders, "should track both providers despite jump")
}

func TestLatestBlockEstimator_ConcurrentAccess(t *testing.T) {
	estimator := NewLatestBlockEstimator()
	parser := newMockChainParser()

	// Test concurrent Record calls
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(providerNum int) {
			for j := 0; j < 100; j++ {
				estimator.Record("provider"+string(rune(providerNum)), int64(1000+j))
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Concurrent Estimate calls
	for i := 0; i < 10; i++ {
		go func() {
			estimator.Estimate(parser)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic and should have valid data
	expectedBlockHeight, numProviders := estimator.Estimate(parser)
	require.Greater(t, expectedBlockHeight, int64(0), "should have valid estimate after concurrent access")
	require.Greater(t, numProviders, 0, "should have providers after concurrent access")
}

func TestLatestBlockEstimator_ZeroAverageBlockTime(t *testing.T) {
	estimator := NewLatestBlockEstimator()
	parser := &mockChainParserForEstimator{
		ChainParser:                   nil,
		allowedBlockLagForQosSync:     10,
		averageBlockTime:              0, // Invalid
		blockDistanceForFinalizedData: 15,
	}

	estimator.Record("provider1", 1000)

	// Should handle zero average block time gracefully
	expectedBlockHeight, numProviders := estimator.Estimate(parser)

	// interpolation will be 0, so should still work
	require.Equal(t, int64(990), expectedBlockHeight)
	require.Equal(t, 1, numProviders)
}

func TestLatestBlockEstimator_InterpolationWithTimePassed(t *testing.T) {
	estimator := NewLatestBlockEstimator()
	parser := newMockChainParser()

	// Manually set old observation
	estimator.mu.Lock()
	oldTime := time.Now().Add(-30 * time.Second) // 30 seconds ago
	estimator.observations["provider1"] = providerBlockObservation{
		LatestBlock: 1000,
		LastUpdated: oldTime,
	}
	estimator.mu.Unlock()

	expectedBlockHeight, numProviders := estimator.Estimate(parser)

	// With 6s average block time, 30s = 5 blocks
	// latestFinalizedBlock = 1000 - 15 = 985
	// interpolation = 30s / 6s = 5 blocks
	// expectedBlocks = 985 + 5 = 990
	// providersMedianOfLatestBlock = 990 + 15 = 1005
	// expectedBlockHeight = 1005 - 10 = 995

	require.Equal(t, int64(995), expectedBlockHeight, "should interpolate forward with time")
	require.Equal(t, 1, numProviders)
}

func TestLatestBlockEstimator_MixedBlockHeights(t *testing.T) {
	estimator := NewLatestBlockEstimator()
	parser := newMockChainParser()

	// Providers at different heights (simulating some behind)
	estimator.Record("provider1", 1100)
	estimator.Record("provider2", 1000)
	estimator.Record("provider3", 1050)
	estimator.Record("provider4", 900)
	estimator.Record("provider5", 1200)

	expectedBlockHeight, numProviders := estimator.Estimate(parser)

	// Should use median, not average or max
	require.Equal(t, 5, numProviders)
	require.Greater(t, expectedBlockHeight, int64(900), "should be above minimum")
	require.Less(t, expectedBlockHeight, int64(1200), "should be below maximum")
}

func TestLatestBlockEstimator_ProviderUpdatesOverTime(t *testing.T) {
	estimator := NewLatestBlockEstimator()
	parser := newMockChainParser()

	// Simulate provider updating multiple times
	for block := int64(1000); block <= 1050; block += 10 {
		estimator.Record("provider1", block)

		expectedBlockHeight, numProviders := estimator.Estimate(parser)

		require.Equal(t, 1, numProviders, "should always have 1 provider")
		require.Greater(t, expectedBlockHeight, int64(0), "should have valid estimate")
	}

	// Final check
	expectedBlockHeight, _ := estimator.Estimate(parser)

	// Latest was 1050
	// latestFinalizedBlock = 1050 - 15 = 1035
	// expectedBlockHeight = (1035 + 15) - 10 = 1040
	require.Equal(t, int64(1040), expectedBlockHeight, "should use most recent observation")
}

func TestLatestBlockEstimator_AllProvidersWithZeroOrNegativeBlocks(t *testing.T) {
	estimator := NewLatestBlockEstimator()
	parser := newMockChainParser()

	// All providers report invalid blocks
	estimator.mu.Lock()
	estimator.observations["provider1"] = providerBlockObservation{
		LatestBlock: 0,
		LastUpdated: time.Now(),
	}
	estimator.observations["provider2"] = providerBlockObservation{
		LatestBlock: -10,
		LastUpdated: time.Now(),
	}
	estimator.mu.Unlock()

	expectedBlockHeight, numProviders := estimator.Estimate(parser)

	// Should filter out invalid observations
	require.Equal(t, int64(0), expectedBlockHeight, "should return 0 with no valid observations")
	require.Equal(t, 0, numProviders, "should report 0 providers after filtering")
}

func TestLatestBlockEstimator_ExpectedBlockHeightNeverNegative(t *testing.T) {
	estimator := NewLatestBlockEstimator()
	parser := newMockChainParser()

	// Provider with very low block that would cause negative result
	estimator.Record("provider1", 5)

	expectedBlockHeight, numProviders := estimator.Estimate(parser)

	// latestFinalizedBlock = max(5 - 15, 0) = 0
	// providersMedianOfLatestBlock = 0 + 15 = 15
	// expectedBlockHeight = max(15 - 10, 0) = 5

	require.GreaterOrEqual(t, expectedBlockHeight, int64(0), "expected block height should never be negative")
	require.Equal(t, 1, numProviders)
}

func TestLatestBlockEstimator_RecordFiltersZeroBlock(t *testing.T) {
	estimator := NewLatestBlockEstimator()

	// Record with zero block should be ignored
	estimator.Record("provider1", 0)

	require.Empty(t, estimator.observations, "should not record zero block")
}

func TestLatestBlockEstimator_RecordFiltersEmptyAddress(t *testing.T) {
	estimator := NewLatestBlockEstimator()

	// Record with empty address should be ignored
	estimator.Record("", 1000)

	require.Empty(t, estimator.observations, "should not record empty address")
}

func TestLatestBlockEstimator_RecordFiltersNegativeBlock(t *testing.T) {
	estimator := NewLatestBlockEstimator()

	// Record with negative block should be ignored
	estimator.Record("provider1", -100)

	require.Empty(t, estimator.observations, "should not record negative block")
}
