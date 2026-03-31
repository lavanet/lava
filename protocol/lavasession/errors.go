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

var ( // Provider Side Errors
	InvalidEpochError                                = errors.New("Requested Epoch Is Too Old")
	NewSessionWithRelayNumError                      = errors.New("Requested Session With Relay Number Is Invalid")
	ConsumerIsBlockListed                            = errors.New("This Consumer Is Blocked.")
	ConsumerNotRegisteredYet                         = errors.New("This Consumer Is Not Currently In The Pool.")
	SessionDoesNotExist                              = errors.New("This Session Id Does Not Exist.")
	MaximumCULimitReachedByConsumer                  = errors.New("Consumer reached maximum cu limit")
	ProviderConsumerCuMisMatch                       = errors.New("Provider and Consumer disagree on total cu for session")
	RelayNumberMismatch                              = errors.New("Provider and Consumer disagree on relay number for session")
	SubscriptionInitiationError                      = errors.New("Provider failed initiating subscription")
	EpochIsNotRegisteredError                        = errors.New("Epoch is not registered in provider session manager")
	ConsumerIsNotRegisteredError                     = errors.New("Consumer is not registered in provider session manager")
	SubscriptionAlreadyExistsError                   = errors.New("Subscription already exists in single provider session")
	SubscriptionPointerIsNilError                    = errors.New("Trying to unsubscribe a nil pointer.")
	CouldNotFindIndexAsConsumerNotYetRegisteredError = errors.New("fetching provider index from psm failed")
	ProviderIndexMisMatchError                       = errors.New("provider index mismatch")
	SessionIdNotFoundError                           = errors.New("Session Id not found")
)

// SessionOutOfSyncGRPCCode is the gRPC status code used when wrapping SessionOutOfSyncError
// as a gRPC status error. This value was historically derived from the cosmos SDK error code (677).
const SessionOutOfSyncGRPCCode = 677
