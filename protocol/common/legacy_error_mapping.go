package common

// legacyKey identifies a legacy sdkerrors.Error by the (codespace, code) tuple
// that cosmossdk.io/errors uses for uniqueness. Keying on the tuple — rather
// than the raw code — prevents numeric collisions with foreign Cosmos modules
// that may flow through ClassifyLegacyError.
type legacyKey struct {
	codespace string
	code      uint32
}

// legacyCodeToLavaError maps existing sdkerrors errors (from lavasession,
// protocolerrors, chaintracker, common, performance) to their LavaError
// equivalents. The key is the (codespace, code) tuple taken directly from
// each sdkerrors.New(...) definition.
//
// Lava uses sdkerrors.New(<name>, <code>, <desc>) where the first argument is
// the codespace — so each error's codespace is the descriptive name string
// from its definition. Keep this map in sync with the source files when
// adding or renaming legacy errors.
//
// Usage:
//
//	lavaErr := ClassifyLegacyError(legacyErr, transport)
var legacyCodeToLavaError = map[legacyKey]*LavaError{
	// --- protocol/common/errors.go ---
	{"ContextDeadlineExceeded Error", 300}:                LavaErrorContextDeadline,
	{"Disallowed StatusCode Error", 504}:                  LavaErrorNodeGatewayTimeout,
	{"Disallowed StatusCode Error", 429}:                  LavaErrorNodeRateLimited,
	{"Disallowed StatusCode Error", 800}:                  LavaErrorNodeServerError,
	{"APINotSupported Error", 900}:                        LavaErrorNodeMethodNotFound,
	{"SubscriptionNotFoundError Error", 901}:              LavaErrorSubscriptionNotFound,
	{"ProviderFinalizationDataAccountability Error", 3365}: LavaErrorFinalizationError,

	// --- protocol/lavasession/errors.go (consumer) ---
	{"pairingListEmpty Error", 665}:                     LavaErrorNoProviders,
	{"AllProviderEndpointsDisabled Error", 667}:         LavaErrorAllEndpointsDisabled,
	{"MaximumNumberOfSessionsExceeded Error", 668}:      LavaErrorSessionNotFound,
	{"MaxComputeUnitsExceeded Error", 669}:              LavaErrorMaxCUExceeded,
	{"ReportingAnOldEpoch Error", 670}:                  LavaErrorEpochMismatch,
	{"AddressIndexWasNotFound Error", 671}:              LavaErrorConsumerNotRegistered,
	{"SessionIsAlreadyBlockListed Error", 673}:          LavaErrorConsumerBlocked,
	{"SessionOutOfSync Error", 677}:                     LavaErrorSessionOutOfSync,
	{"MaximumNumberOfBlockListedSessions Error", 678}:   LavaErrorSessionNotFound,
	{"ContextDoneNoNeedToLockSelection Error", 687}:     LavaErrorContextDeadline,
	{"ConsistencyPreValidation Error", 699}:             LavaErrorConsistencyError,

	// --- protocol/lavasession/errors.go (provider) ---
	{"InvalidEpoch Error", 881}:                                    LavaErrorEpochMismatch,
	{"NewSessionWithRelayNum Error", 882}:                          LavaErrorRelayNumberMismatch,
	{"ConsumerIsBlockListed Error", 883}:                           LavaErrorConsumerBlocked,
	{"ConsumerNotActive Error", 884}:                               LavaErrorConsumerNotRegistered,
	{"SessionDoesNotExist Error", 885}:                             LavaErrorSessionNotFound,
	{"MaximumCULimitReachedByConsumer Error", 886}:                 LavaErrorMaxCUExceeded,
	{"ProviderConsumerCuMisMatch Error", 887}:                      LavaErrorSessionOutOfSync,
	{"RelayNumberMismatch Error", 888}:                             LavaErrorRelayNumberMismatch,
	{"SubscriptionInitiationError Error", 889}:                     LavaErrorSubscriptionInitFailed,
	{"EpochIsNotRegisteredError Error", 890}:                       LavaErrorEpochMismatch,
	{"ConsumerIsNotRegisteredError Error", 891}:                    LavaErrorConsumerNotRegistered,
	{"SubscriptionAlreadyExists Error", 892}:                       LavaErrorSubscriptionAlreadyExists,
	// SubscriptionPointerIsNil is a provider-side invariant violation (a bug),
	// not a user-facing "subscription missing" signal. Classify as an internal
	// relay-processing failure so it surfaces in internal-error dashboards.
	{"SubscriptionPointerIsNil Error", 896}:                        LavaErrorRelayProcessingFailed,
	{"CouldNotFindIndexAsConsumerNotYetRegistered Error", 897}:     LavaErrorConsumerNotRegistered,
	{"ProviderIndexMisMatch Error", 898}:                           LavaErrorSessionOutOfSync,
	{"SessionIdNotFound Error", 899}:                               LavaErrorSessionNotFound,

	// --- protocol/lavaprotocol/protocolerrors/errors.go ---
	{"ProviderFinalizationData Error", 3365}:               LavaErrorFinalizationError,
	{"ProviderFinalizationDataAccountability Error", 3366}: LavaErrorFinalizationError,
	{"HashesConsensus Error", 3367}:                        LavaErrorHashConsensusError,
	{"Consistency Error", 3368}:                            LavaErrorConsistencyError,
	{"UnhandledRelayReceiver Error", 3369}:                 LavaErrorNodeMethodNotFound,
	{"DisabledRelayReceiverError Error", 3370}:             LavaErrorNodeMethodNotSupported,
	{"NoResponseTimeout Error", 685}:                       LavaErrorNoResponseTimeout,

	// --- protocol/chaintracker/errors.go ---
	{"Invalid value for latestBlockNum", 10703}: LavaErrorChainBlockNotFound,
	{"Error FailedToFetchLatestBlock", 10705}:   LavaErrorChainBlockNotFound,
	// 10706/10707/10709 are client-side input-validation failures (malformed
	// range, invalid specific block) — not chain-state-availability issues.
	// Classifying them as USER_INVALID_PARAMS keeps the chain-data-unavailable
	// metric clean for real node/chain issues and prevents retries that
	// burn CU on requests that can never succeed.
	{"Error InvalidRequestedBlocks", 10706}:          LavaErrorUserInvalidParams,
	{"RequestedBlocksOutOfRange", 10707}:             LavaErrorUserInvalidParams,
	{"Error ErrorFailedToFetchTooEarlyBlock", 10708}: LavaErrorChainBlockTooOld,
	{"Error InvalidRequestedSpecificBlock", 10709}:   LavaErrorUserInvalidParams,

	// --- protocol/performance/errors.go ---
	{"Not Connected Error", 700}: LavaErrorConnectionRefused,

	// --- protocol/chainlib/common.go ---
	// ErrBatchRequestSizeExceeded uses sdkerrors codespace, code 1 — mapped by message instead
}

// ClassifyLegacyError extracts the sdkerrors (codespace, code) tuple from a
// legacy error and returns the corresponding LavaError. Returns the
// transport-scoped ClassifyError result if no mapping exists.
//
// transport is used for the message-based fallback when no sdkerrors
// codespace+code is found or the pair is not in our mapping. Use this for
// protocol-layer errors from the lavasession/protocolerrors packages that
// carry an ABCI code via the sdkerrors interface.
func ClassifyLegacyError(err error, transport TransportType) *LavaError {
	if err == nil {
		return nil
	}

	// Try to extract sdkerrors (codespace, code) tuple
	if key, ok := extractLegacyKey(err); ok {
		if le, mapped := legacyCodeToLavaError[key]; mapped {
			return le
		}
	}

	// Fall back to message-based classification using the caller's transport
	return ClassifyError(DetectConnectionError(err), -1, transport, 0, err.Error())
}

// extractLegacyKey attempts to extract the (codespace, code) tuple from an
// sdkerrors error. cosmossdk.io/errors implements both Codespace() string
// and ABCICode() uint32 on *Error. Returns ok=false if either method is
// missing (e.g. plain errors.New, or an older ABCIError-only interface).
func extractLegacyKey(err error) (legacyKey, bool) {
	type codespaceCoder interface {
		Codespace() string
		ABCICode() uint32
	}
	if c, ok := err.(codespaceCoder); ok {
		return legacyKey{codespace: c.Codespace(), code: c.ABCICode()}, true
	}
	return legacyKey{}, false
}