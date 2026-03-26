package common

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyLegacyError_NilError(t *testing.T) {
	assert.Nil(t, ClassifyLegacyError(nil))
}

func TestClassifyLegacyError_NonLegacyError(t *testing.T) {
	// A plain error without sdkerrors code falls back to message classification
	err := errors.New("nonce too low")
	result := ClassifyLegacyError(err)
	assert.Equal(t, LavaErrorChainNonceTooLow, result)
}

func TestClassifyLegacyError_UnknownFallback(t *testing.T) {
	err := errors.New("completely unknown error")
	result := ClassifyLegacyError(err)
	assert.Equal(t, LavaErrorUnknown, result)
}

// mockSDKError mimics the ABCIError interface from cosmossdk.io/errors
type mockSDKError struct {
	code uint32
	msg  string
}

func (e *mockSDKError) Error() string    { return e.msg }
func (e *mockSDKError) ABCICode() uint32 { return e.code }

func TestClassifyLegacyError_SessionErrors(t *testing.T) {
	tests := []struct {
		name     string
		code     uint32
		expected *LavaError
	}{
		{"PairingListEmpty", 665, LavaErrorNoProviders},
		{"AllEndpointsDisabled", 667, LavaErrorAllEndpointsDisabled},
		{"MaxSessions", 668, LavaErrorSessionNotFound},
		{"MaxCU", 669, LavaErrorMaxCUExceeded},
		{"EpochMismatch", 670, LavaErrorEpochMismatch},
		{"ConsumerNotRegistered", 671, LavaErrorConsumerNotRegistered},
		{"ConsumerBlocked", 673, LavaErrorConsumerBlocked},
		{"SessionOutOfSync", 677, LavaErrorSessionOutOfSync},
		{"ContextDeadline", 687, LavaErrorContextDeadline},
		{"ConsistencyPreValidation", 699, LavaErrorConsistencyError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &mockSDKError{code: tt.code, msg: "test"}
			result := ClassifyLegacyError(err)
			require.NotNil(t, result)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClassifyLegacyError_ProviderErrors(t *testing.T) {
	tests := []struct {
		name     string
		code     uint32
		expected *LavaError
	}{
		{"InvalidEpoch", 881, LavaErrorEpochMismatch},
		{"RelayNumMismatch", 882, LavaErrorRelayNumberMismatch},
		{"ConsumerBlockListed", 883, LavaErrorConsumerBlocked},
		{"ConsumerNotRegistered", 884, LavaErrorConsumerNotRegistered},
		{"SessionNotExist", 885, LavaErrorSessionNotFound},
		{"MaxCU", 886, LavaErrorMaxCUExceeded},
		{"CuMismatch", 887, LavaErrorSessionOutOfSync},
		{"RelayNumberMismatch", 888, LavaErrorRelayNumberMismatch},
		{"SubscriptionInit", 889, LavaErrorSubscriptionInitFailed},
		{"EpochNotRegistered", 890, LavaErrorEpochMismatch},
		{"ConsumerNotRegistered2", 891, LavaErrorConsumerNotRegistered},
		{"SessionIdNotFound", 899, LavaErrorSessionNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &mockSDKError{code: tt.code, msg: "test"}
			result := ClassifyLegacyError(err)
			require.NotNil(t, result)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClassifyLegacyError_ProtocolErrors(t *testing.T) {
	tests := []struct {
		name     string
		code     uint32
		expected *LavaError
	}{
		{"FinalizationData", 3365, LavaErrorFinalizationError},
		{"FinalizationAccountability", 3366, LavaErrorFinalizationError},
		{"HashesConsensus", 3367, LavaErrorHashConsensusError},
		{"Consistency", 3368, LavaErrorConsistencyError},
		{"UnhandledRelay", 3369, LavaErrorNodeMethodNotFound},
		{"DisabledRelay", 3370, LavaErrorNodeMethodNotSupported},
		{"NoResponseTimeout", 685, LavaErrorNoResponseTimeout},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &mockSDKError{code: tt.code, msg: "test"}
			result := ClassifyLegacyError(err)
			require.NotNil(t, result)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClassifyLegacyError_CommonErrors(t *testing.T) {
	tests := []struct {
		name     string
		code     uint32
		expected *LavaError
	}{
		{"ContextDeadline", 300, LavaErrorContextDeadline},
		{"StatusCode504", 504, LavaErrorNodeGatewayTimeout},
		{"StatusCode429", 429, LavaErrorNodeRateLimited},
		{"StatusCodeStrict", 800, LavaErrorNodeServerError},
		{"APINotSupported", 900, LavaErrorNodeMethodNotFound},
		{"SubscriptionNotFound", 901, LavaErrorSubscriptionNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &mockSDKError{code: tt.code, msg: "test"}
			result := ClassifyLegacyError(err)
			require.NotNil(t, result)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClassifyLegacyError_ChainTrackerErrors(t *testing.T) {
	tests := []struct {
		name     string
		code     uint32
		expected *LavaError
	}{
		{"InvalidLatestBlock", 10703, LavaErrorChainBlockNotFound},
		{"FailedFetchLatest", 10705, LavaErrorChainBlockNotFound},
		{"InvalidRequested", 10706, LavaErrorChainDataNotAvailable},
		{"OutOfRange", 10707, LavaErrorChainDataNotAvailable},
		{"TooEarlyBlock", 10708, LavaErrorChainBlockTooOld},
		{"InvalidSpecific", 10709, LavaErrorChainDataNotAvailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &mockSDKError{code: tt.code, msg: "test"}
			result := ClassifyLegacyError(err)
			require.NotNil(t, result)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClassifyLegacyError_PerformanceErrors(t *testing.T) {
	err := &mockSDKError{code: 700, msg: "No Connection To grpc server"}
	result := ClassifyLegacyError(err)
	assert.Equal(t, LavaErrorConnectionRefused, result)
}

func TestClassifyLegacyError_UnmappedCode(t *testing.T) {
	// An sdkerrors code that isn't in our mapping should fall back to message classification
	err := &mockSDKError{code: 99999, msg: "some unknown legacy error"}
	result := ClassifyLegacyError(err)
	assert.Equal(t, LavaErrorUnknown, result)
}

func TestLegacyMappingCoversAllCategories(t *testing.T) {
	// Verify the mapping covers both internal and external error categories
	hasInternal := false
	hasExternal := false
	for _, le := range legacyCodeToLavaError {
		if le.Category == CategoryInternal {
			hasInternal = true
		}
		if le.Category == CategoryExternal {
			hasExternal = true
		}
	}
	assert.True(t, hasInternal, "mapping should include internal errors")
	assert.True(t, hasExternal, "mapping should include external errors")
}
