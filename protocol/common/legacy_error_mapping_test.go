package common

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLegacyErrorMapping_NilError(t *testing.T) {
	assert.Nil(t, ClassifyLegacyError(nil, TransportJsonRPC))
}

func TestLegacyErrorMapping_NonLegacyError(t *testing.T) {
	// A plain error without sdkerrors code falls back to message classification
	err := errors.New("nonce too low")
	result := ClassifyLegacyError(err, TransportJsonRPC)
	assert.Equal(t, LavaErrorChainNonceTooLow, result)
}

func TestLegacyErrorMapping_UnknownFallback(t *testing.T) {
	err := errors.New("completely unknown error")
	result := ClassifyLegacyError(err, TransportJsonRPC)
	assert.Equal(t, LavaErrorUnknown, result)
}

func TestLegacyErrorMapping_FallbackUsesTransport(t *testing.T) {
	// A gRPC "not implemented" message must be classified via gRPC matchers,
	// not JSON-RPC matchers — this is the bug the transport parameter fixes.
	err := errors.New("not implemented")
	grpcResult := ClassifyLegacyError(err, TransportGRPC)
	assert.Equal(t, LavaErrorNodeUnimplemented, grpcResult, "gRPC transport should match 'not implemented'")

	jsonRPCResult := ClassifyLegacyError(err, TransportJsonRPC)
	assert.Equal(t, LavaErrorUnknown, jsonRPCResult, "JSON-RPC transport should not match gRPC-only message")
}

// mockSDKError mimics the (codespace, code) interface from cosmossdk.io/errors.
// The legacy mapping keys on the tuple so every mock must supply both fields.
type mockSDKError struct {
	codespace string
	code      uint32
	msg       string
}

func (e *mockSDKError) Error() string     { return e.msg }
func (e *mockSDKError) Codespace() string { return e.codespace }
func (e *mockSDKError) ABCICode() uint32  { return e.code }

type legacyCase struct {
	name      string
	codespace string
	code      uint32
	expected  *LavaError
}

func runLegacyCases(t *testing.T, cases []legacyCase) {
	t.Helper()
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := &mockSDKError{codespace: tt.codespace, code: tt.code, msg: "test"}
			result := ClassifyLegacyError(err, TransportJsonRPC)
			require.NotNil(t, result)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLegacyErrorMapping_SessionErrors(t *testing.T) {
	runLegacyCases(t, []legacyCase{
		{"PairingListEmpty", "pairingListEmpty Error", 665, LavaErrorNoProviders},
		{"AllEndpointsDisabled", "AllProviderEndpointsDisabled Error", 667, LavaErrorAllEndpointsDisabled},
		{"MaxSessions", "MaximumNumberOfSessionsExceeded Error", 668, LavaErrorSessionNotFound},
		{"MaxCU", "MaxComputeUnitsExceeded Error", 669, LavaErrorMaxCUExceeded},
		{"EpochMismatch", "ReportingAnOldEpoch Error", 670, LavaErrorEpochMismatch},
		{"ConsumerNotRegistered", "AddressIndexWasNotFound Error", 671, LavaErrorConsumerNotRegistered},
		{"ConsumerBlocked", "SessionIsAlreadyBlockListed Error", 673, LavaErrorConsumerBlocked},
		{"SessionOutOfSync", "SessionOutOfSync Error", 677, LavaErrorSessionOutOfSync},
		{"ContextDeadline", "ContextDoneNoNeedToLockSelection Error", 687, LavaErrorContextDeadline},
		{"ConsistencyPreValidation", "ConsistencyPreValidation Error", 699, LavaErrorConsistencyError},
	})
}

func TestLegacyErrorMapping_ProviderErrors(t *testing.T) {
	runLegacyCases(t, []legacyCase{
		{"InvalidEpoch", "InvalidEpoch Error", 881, LavaErrorEpochMismatch},
		{"RelayNumMismatch", "NewSessionWithRelayNum Error", 882, LavaErrorRelayNumberMismatch},
		{"ConsumerBlockListed", "ConsumerIsBlockListed Error", 883, LavaErrorConsumerBlocked},
		{"ConsumerNotRegistered", "ConsumerNotActive Error", 884, LavaErrorConsumerNotRegistered},
		{"SessionNotExist", "SessionDoesNotExist Error", 885, LavaErrorSessionNotFound},
		{"MaxCU", "MaximumCULimitReachedByConsumer Error", 886, LavaErrorMaxCUExceeded},
		{"CuMismatch", "ProviderConsumerCuMisMatch Error", 887, LavaErrorSessionOutOfSync},
		{"RelayNumberMismatch", "RelayNumberMismatch Error", 888, LavaErrorRelayNumberMismatch},
		{"SubscriptionInit", "SubscriptionInitiationError Error", 889, LavaErrorSubscriptionInitFailed},
		{"EpochNotRegistered", "EpochIsNotRegisteredError Error", 890, LavaErrorEpochMismatch},
		{"ConsumerNotRegistered2", "ConsumerIsNotRegisteredError Error", 891, LavaErrorConsumerNotRegistered},
		{"SubscriptionAlreadyExists", "SubscriptionAlreadyExists Error", 892, LavaErrorSubscriptionAlreadyExists},
		// SubscriptionPointerIsNil is a provider-side invariant violation, so it
		// surfaces as RelayProcessingFailed rather than the misleading NotFound
		// classification that preceded the fix.
		{"SubscriptionPointerIsNil", "SubscriptionPointerIsNil Error", 896, LavaErrorRelayProcessingFailed},
		{"SessionIdNotFound", "SessionIdNotFound Error", 899, LavaErrorSessionNotFound},
	})
}

func TestLegacyErrorMapping_ProtocolErrors(t *testing.T) {
	runLegacyCases(t, []legacyCase{
		{"FinalizationData", "ProviderFinalizationData Error", 3365, LavaErrorFinalizationError},
		{"FinalizationAccountability", "ProviderFinalizationDataAccountability Error", 3366, LavaErrorFinalizationError},
		{"HashesConsensus", "HashesConsensus Error", 3367, LavaErrorHashConsensusError},
		{"Consistency", "Consistency Error", 3368, LavaErrorConsistencyError},
		{"UnhandledRelay", "UnhandledRelayReceiver Error", 3369, LavaErrorNodeMethodNotFound},
		{"DisabledRelay", "DisabledRelayReceiverError Error", 3370, LavaErrorNodeMethodNotSupported},
		{"NoResponseTimeout", "NoResponseTimeout Error", 685, LavaErrorNoResponseTimeout},
	})
}

func TestLegacyErrorMapping_CommonErrors(t *testing.T) {
	runLegacyCases(t, []legacyCase{
		{"ContextDeadline", "ContextDeadlineExceeded Error", 300, LavaErrorContextDeadline},
		{"StatusCode504", "Disallowed StatusCode Error", 504, LavaErrorNodeGatewayTimeout},
		{"StatusCode429", "Disallowed StatusCode Error", 429, LavaErrorNodeRateLimited},
		{"StatusCodeStrict", "Disallowed StatusCode Error", 800, LavaErrorNodeServerError},
		{"APINotSupported", "APINotSupported Error", 900, LavaErrorNodeMethodNotFound},
		{"SubscriptionNotFound", "SubscriptionNotFoundError Error", 901, LavaErrorSubscriptionNotFound},
	})
}

func TestLegacyErrorMapping_ChainTrackerErrors(t *testing.T) {
	runLegacyCases(t, []legacyCase{
		{"InvalidLatestBlock", "Invalid value for latestBlockNum", 10703, LavaErrorChainBlockNotFound},
		{"FailedFetchLatest", "Error FailedToFetchLatestBlock", 10705, LavaErrorChainBlockNotFound},
		// 10706/10707/10709 are request-input validation failures — the consumer
		// asked for an impossible range. Must classify as USER_INVALID_PARAMS so
		// they don't inflate CHAIN_DATA_NOT_AVAILABLE and don't trigger retries
		// on requests that can never succeed.
		{"InvalidRequested", "Error InvalidRequestedBlocks", 10706, LavaErrorUserInvalidParams},
		{"OutOfRange", "RequestedBlocksOutOfRange", 10707, LavaErrorUserInvalidParams},
		{"TooEarlyBlock", "Error ErrorFailedToFetchTooEarlyBlock", 10708, LavaErrorChainBlockTooOld},
		{"InvalidSpecific", "Error InvalidRequestedSpecificBlock", 10709, LavaErrorUserInvalidParams},
	})
}

func TestLegacyErrorMapping_PerformanceErrors(t *testing.T) {
	err := &mockSDKError{codespace: "Not Connected Error", code: 700, msg: "No Connection To grpc server"}
	result := ClassifyLegacyError(err, TransportJsonRPC)
	assert.Equal(t, LavaErrorConnectionRefused, result)
}

func TestLegacyErrorMapping_UnmappedCode(t *testing.T) {
	// An sdkerrors code that isn't in our mapping should fall back to message classification
	err := &mockSDKError{codespace: "unknown", code: 99999, msg: "some unknown legacy error"}
	result := ClassifyLegacyError(err, TransportJsonRPC)
	assert.Equal(t, LavaErrorUnknown, result)
}

func TestLegacyErrorMapping_ForeignCodespaceDoesNotCollide(t *testing.T) {
	// A foreign Cosmos module error with the same numeric code as a legacy
	// Lava error MUST NOT match — this is the reason the map is keyed on
	// the (codespace, code) tuple. 885 is LavaErrorSessionNotFound only
	// under the "SessionDoesNotExist Error" codespace.
	foreign := &mockSDKError{codespace: "sdk", code: 885, msg: "unrelated foreign module error"}
	result := ClassifyLegacyError(foreign, TransportJsonRPC)
	assert.Equal(t, LavaErrorUnknown, result,
		"foreign codespace with matching code must not collide with Lava legacy mapping")
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