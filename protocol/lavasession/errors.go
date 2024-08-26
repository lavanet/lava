package lavasession

import (
	sdkerrors "cosmossdk.io/errors"
)

var ( // Consumer Side Errors
	PairingListEmptyError                                = sdkerrors.New("pairingListEmpty Error", 665, "No pairings available.") // client could not connect to any provider.
	UnreachableCodeError                                 = sdkerrors.New("UnreachableCode Error", 666, "Should not get here.")
	AllProviderEndpointsDisabledError                    = sdkerrors.New("AllProviderEndpointsDisabled Error", 667, "All endpoints are not available.") // a provider is completely unresponsive all endpoints are not available
	MaximumNumberOfSessionsExceededError                 = sdkerrors.New("MaximumNumberOfSessionsExceeded Error", 668, "Provider reached maximum number of active sessions.")
	MaxComputeUnitsExceededError                         = sdkerrors.New("MaxComputeUnitsExceeded Error", 669, "Consumer is trying to exceed the maximum number of compute units available.")
	EpochMismatchError                                   = sdkerrors.New("ReportingAnOldEpoch Error", 670, "Tried to Report to an older epoch")
	AddressIndexWasNotFoundError                         = sdkerrors.New("AddressIndexWasNotFound Error", 671, "address index was not found in list")
	LockMisUseDetectedError                              = sdkerrors.New("LockMisUseDetected Error", 672, "Faulty use of locks detected")
	SessionIsAlreadyBlockListedError                     = sdkerrors.New("SessionIsAlreadyBlockListed Error", 673, "Session is already in block list")
	NegativeComputeUnitsAmountError                      = sdkerrors.New("NegativeComputeUnitsAmount", 674, "Tried to subtract to negative compute units amount")
	ReportAndBlockProviderError                          = sdkerrors.New("ReportAndBlockProvider Error", 675, "Report and block the provider")
	BlockProviderError                                   = sdkerrors.New("BlockProvider Error", 676, "Block the provider")
	SessionOutOfSyncError                                = sdkerrors.New("SessionOutOfSync Error", 677, "Session went out of sync with the provider") // do no change the name, before also fixing the consumerSessionManager.ts file as it relies on the error message
	MaximumNumberOfBlockListedSessionsError              = sdkerrors.New("MaximumNumberOfBlockListedSessions Error", 678, "Provider reached maximum number of block listed sessions.")
	SendRelayError                                       = sdkerrors.New("SendRelay Error", 679, "Failed To Send Relay")
	DataReliabilityIndexRequestedIsOriginalProviderError = sdkerrors.New("DataReliabilityIndexRequestedIsOriginalProvider Error", 680, "Data reliability session index belongs to the original provider")
	DataReliabilityIndexOutOfRangeError                  = sdkerrors.New("DataReliabilityIndexOutOfRange Error", 681, "Trying to get provider index out of range")
	DataReliabilityAlreadySentThisEpochError             = sdkerrors.New("DataReliabilityAlreadySentThisEpoch Error", 682, "Trying to send data reliability more than once per provider per epoch")
	FailedToConnectToEndPointForDataReliabilityError     = sdkerrors.New("FailedToConnectToEndPointForDataReliability Error", 683, "Failed to connect to a providers endpoints")
	DataReliabilityEpochMismatchError                    = sdkerrors.New("DataReliabilityEpochMismatch Error", 684, "Data reliability epoch mismatch original session epoch.")
	NoDataReliabilitySessionWasCreatedError              = sdkerrors.New("NoDataReliabilitySessionWasCreated Error", 685, "No Data reliability session was created")
	ContextDoneNoNeedToLockSelectionError                = sdkerrors.New("ContextDoneNoNeedToLockSelection Error", 687, "Context deadline exceeded while trying to lock selection")
	BlockEndpointError                                   = sdkerrors.New("BlockEndpoint Error", 688, "Block the endpoint")
)

var ( // Provider Side Errors
	InvalidEpochError                                = sdkerrors.New("InvalidEpoch Error", 881, "Requested Epoch Is Too Old")
	NewSessionWithRelayNumError                      = sdkerrors.New("NewSessionWithRelayNum Error", 882, "Requested Session With Relay Number Is Invalid")
	ConsumerIsBlockListed                            = sdkerrors.New("ConsumerIsBlockListed Error", 883, "This Consumer Is Blocked.")
	ConsumerNotRegisteredYet                         = sdkerrors.New("ConsumerNotActive Error", 884, "This Consumer Is Not Currently In The Pool.")
	SessionDoesNotExist                              = sdkerrors.New("SessionDoesNotExist Error", 885, "This Session Id Does Not Exist.")
	MaximumCULimitReachedByConsumer                  = sdkerrors.New("MaximumCULimitReachedByConsumer Error", 886, "Consumer reached maximum cu limit")
	ProviderConsumerCuMisMatch                       = sdkerrors.New("ProviderConsumerCuMisMatch Error", 887, "Provider and Consumer disagree on total cu for session")
	RelayNumberMismatch                              = sdkerrors.New("RelayNumberMismatch Error", 888, "Provider and Consumer disagree on relay number for session")
	SubscriptionInitiationError                      = sdkerrors.New("SubscriptionInitiationError Error", 889, "Provider failed initiating subscription")
	EpochIsNotRegisteredError                        = sdkerrors.New("EpochIsNotRegisteredError Error", 890, "Epoch is not registered in provider session manager")
	ConsumerIsNotRegisteredError                     = sdkerrors.New("ConsumerIsNotRegisteredError Error", 891, "Consumer is not registered in provider session manager")
	SubscriptionAlreadyExistsError                   = sdkerrors.New("SubscriptionAlreadyExists Error", 892, "Subscription already exists in single provider session")
	DataReliabilitySessionAlreadyUsedError           = sdkerrors.New("DataReliabilitySessionAlreadyUsed Error", 893, "Data Reliability Session already used by this consumer in this epoch")
	DataReliabilityCuSumMisMatchError                = sdkerrors.New("DataReliabilityCuSumMisMatch Error", 894, "Data Reliability Cu sum mismatch error")
	DataReliabilityRelayNumberMisMatchError          = sdkerrors.New("DataReliabilityRelayNumberMisMatch Error", 895, "Data Reliability RelayNumber mismatch error")
	SubscriptionPointerIsNilError                    = sdkerrors.New("SubscriptionPointerIsNil Error", 896, "Trying to unsubscribe a nil pointer.")
	CouldNotFindIndexAsConsumerNotYetRegisteredError = sdkerrors.New("CouldNotFindIndexAsConsumerNotYetRegistered Error", 897, "fetching provider index from psm failed")
	ProviderIndexMisMatchError                       = sdkerrors.New("ProviderIndexMisMatch Error", 898, "provider index mismatch")
	SessionIdNotFoundError                           = sdkerrors.New("SessionIdNotFound Error", 899, "Session Id not found")
)
