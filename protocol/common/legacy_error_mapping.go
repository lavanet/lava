package common

// legacyCodeToLavaError maps existing sdkerrors codes (from lavasession, protocolerrors,
// chaintracker, common, performance) to their LavaError equivalents.
//
// This bridge allows legacy errors to be classified for logging/metrics without
// changing any control flow. Phase 7 will replace the sdkerrors definitions entirely.
//
// Usage:
//
//	lavaErr := classifyLegacyError(legacyErr)
var legacyCodeToLavaError = map[uint32]*LavaError{
	// --- protocol/common/errors.go ---
	300: LavaErrorContextDeadline,      // ContextDeadlineExceededError
	504: LavaErrorNodeGatewayTimeout,   // StatusCodeError504
	429: LavaErrorNodeRateLimited,      // StatusCodeError429
	800: LavaErrorNodeServerError,      // StatusCodeErrorStrict
	900: LavaErrorNodeMethodNotFound,   // APINotSupportedError
	901: LavaErrorSubscriptionNotFound, // SubscriptionNotFoundError

	// --- protocol/lavasession/errors.go (consumer) ---
	665: LavaErrorNoProviders,           // PairingListEmptyError
	667: LavaErrorAllEndpointsDisabled,  // AllProviderEndpointsDisabledError
	668: LavaErrorSessionNotFound,       // MaximumNumberOfSessionsExceededError
	669: LavaErrorMaxCUExceeded,         // MaxComputeUnitsExceededError
	670: LavaErrorEpochMismatch,         // EpochMismatchError
	671: LavaErrorConsumerNotRegistered, // AddressIndexWasNotFoundError
	673: LavaErrorConsumerBlocked,       // SessionIsAlreadyBlockListedError
	677: LavaErrorSessionOutOfSync,      // SessionOutOfSyncError
	678: LavaErrorSessionNotFound,       // MaximumNumberOfBlockListedSessionsError
	687: LavaErrorContextDeadline,       // ContextDoneNoNeedToLockSelectionError
	699: LavaErrorConsistencyError,      // ConsistencyPreValidationError

	// --- protocol/lavasession/errors.go (provider) ---
	881: LavaErrorEpochMismatch,          // InvalidEpochError
	882: LavaErrorRelayNumberMismatch,    // NewSessionWithRelayNumError
	883: LavaErrorConsumerBlocked,        // ConsumerIsBlockListed
	884: LavaErrorConsumerNotRegistered,  // ConsumerNotRegisteredYet
	885: LavaErrorSessionNotFound,        // SessionDoesNotExist
	886: LavaErrorMaxCUExceeded,          // MaximumCULimitReachedByConsumer
	887: LavaErrorSessionOutOfSync,       // ProviderConsumerCuMisMatch
	888: LavaErrorRelayNumberMismatch,    // RelayNumberMismatch
	889: LavaErrorSubscriptionInitFailed, // SubscriptionInitiationError
	890: LavaErrorEpochMismatch,          // EpochIsNotRegisteredError
	891: LavaErrorConsumerNotRegistered,  // ConsumerIsNotRegisteredError
	892: LavaErrorSubscriptionNotFound,   // SubscriptionAlreadyExistsError
	896: LavaErrorSubscriptionNotFound,   // SubscriptionPointerIsNilError
	897: LavaErrorConsumerNotRegistered,  // CouldNotFindIndexAsConsumerNotYetRegisteredError
	898: LavaErrorSessionOutOfSync,       // ProviderIndexMisMatchError
	899: LavaErrorSessionNotFound,        // SessionIdNotFoundError

	// --- protocol/lavaprotocol/protocolerrors/errors.go ---
	3365: LavaErrorFinalizationError,      // ProviderFinalizationDataError
	3366: LavaErrorFinalizationError,      // ProviderFinalizationDataAccountabilityError
	3367: LavaErrorHashConsensusError,     // HashesConsensusError
	3368: LavaErrorConsistencyError,       // ConsistencyError
	3369: LavaErrorNodeMethodNotFound,     // UnhandledRelayReceiverError
	3370: LavaErrorNodeMethodNotSupported, // DisabledRelayReceiverError
	685:  LavaErrorNoResponseTimeout,      // NoResponseTimeout

	// --- protocol/chaintracker/errors.go ---
	10703: LavaErrorChainBlockNotFound,    // InvalidLatestBlockNumValue
	10705: LavaErrorChainBlockNotFound,    // ErrorFailedToFetchLatestBlock
	10706: LavaErrorChainDataNotAvailable, // InvalidRequestedBlocks
	10707: LavaErrorChainDataNotAvailable, // RequestedBlocksOutOfRange
	10708: LavaErrorChainBlockTooOld,      // ErrorFailedToFetchTooEarlyBlock
	10709: LavaErrorChainDataNotAvailable, // InvalidRequestedSpecificBlock

	// --- protocol/performance/errors.go ---
	700: LavaErrorConnectionRefused, // NotConnectedError

	// --- protocol/chainlib/common.go ---
	// ErrBatchRequestSizeExceeded uses sdkerrors codespace, code 1 — mapped by message instead
}

// classifyLegacyError extracts the sdkerrors code from a legacy error and returns
// the corresponding LavaError. Returns LavaErrorUnknown if no mapping exists.
func classifyLegacyError(err error) *LavaError {
	if err == nil {
		return nil
	}

	// Try to extract sdkerrors code
	code := extractLegacyCode(err)
	if code != 0 {
		if le, ok := legacyCodeToLavaError[code]; ok {
			return le
		}
	}

	// Fall back to message-based classification
	return ClassifyError(nil, -1, TransportJsonRPC, 0, err.Error())
}

// extractLegacyCode attempts to extract the numeric code from an sdkerrors error.
// sdkerrors implements the ABCIError interface with a Code() method.
func extractLegacyCode(err error) uint32 {
	type coder interface {
		ABCICode() uint32
	}
	if c, ok := err.(coder); ok {
		return c.ABCICode()
	}
	return 0
}
