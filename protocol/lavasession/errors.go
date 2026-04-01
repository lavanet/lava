package lavasession

import "errors"

var ( // Consumer Side Errors
	PairingListEmptyError                   = errors.New("No pairings available.") // client could not connect to any provider.
	UnreachableCodeError                    = errors.New("Should not get here.")
	AllProviderEndpointsDisabledError       = errors.New("All endpoints are not available.") // a provider is completely unresponsive all endpoints are not available
	MaximumNumberOfSessionsExceededError    = errors.New("Provider reached maximum number of active sessions.")
	MaxComputeUnitsExceededError            = errors.New("Consumer is trying to exceed the maximum number of compute units available.")
	EpochMismatchError                      = errors.New("Tried to Report to an older epoch")
	AddressIndexWasNotFoundError            = errors.New("address index was not found in list")
	LockMisUseDetectedError                 = errors.New("Faulty use of locks detected")
	SessionIsAlreadyBlockListedError        = errors.New("Session is already in block list")
	NegativeComputeUnitsAmountError         = errors.New("Tried to subtract to negative compute units amount")
	ReportAndBlockProviderError             = errors.New("Report and block the provider")
	BlockProviderError                      = errors.New("Block the provider")
	SessionOutOfSyncError                   = errors.New("Session went out of sync with the provider") // do no change the name, before also fixing the consumerSessionManager.ts file as it relies on the error message
	MaximumNumberOfBlockListedSessionsError = errors.New("Provider reached maximum number of block listed sessions.")
	SendRelayError                          = errors.New("Failed To Send Relay")
	ContextDoneNoNeedToLockSelectionError   = errors.New("Context deadline exceeded while trying to lock selection")
	BlockEndpointError                      = errors.New("Block the endpoint")
	ConsistencyPreValidationError           = errors.New("endpoint failed pre-request consistency validation")
)


// SessionOutOfSyncGRPCCode is the gRPC status code used when wrapping SessionOutOfSyncError
// as a gRPC status error. This value was historically derived from the cosmos SDK error code (677).
const SessionOutOfSyncGRPCCode = 677
