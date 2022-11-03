package lavasession

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var ( // Consumer Side Errors
	PairingListEmptyError                                = sdkerrors.New("pairingListEmpty Error", 665, "No pairings available.") // client could'nt connect to any provider.
	UnreachableCodeError                                 = sdkerrors.New("UnreachableCode Error", 666, "Should not get here.")
	AllProviderEndpointsDisabledError                    = sdkerrors.New("AllProviderEndpointsDisabled Error", 667, "All endpoints are not available.") // a provider is completely unresponsive all endpoints are not available
	MaximumNumberOfSessionsExceededError                 = sdkerrors.New("MaximumNumberOfSessionsExceeded Error", 668, "Provider reached maximum number of active sessions.")
	MaxComputeUnitsExceededError                         = sdkerrors.New("MaxComputeUnitsExceeded Error", 669, "Consumer is trying to exceed the maximum number of compute uints available.")
	EpochMismatchError                                   = sdkerrors.New("ReportingAnOldEpoch Error", 670, "Tried to Report to an older epoch")
	AddressIndexWasNotFoundError                         = sdkerrors.New("AddressIndexWasNotFound Error", 671, "address index was not found in list")
	LockMisUseDetectedError                              = sdkerrors.New("LockMisUseDetected Error", 672, "Faulty use of locks detected")
	SessionIsAlreadyBlockListedError                     = sdkerrors.New("SessionIsAlreadyBlockListed Error", 673, "Session is already in block list")
	NegativeComputeUnitsAmountError                      = sdkerrors.New("NegativeComputeUnitsAmount", 674, "Tried to subtract to negative compute units amount")
	ReportAndBlockProviderError                          = sdkerrors.New("ReportAndBlockProvider Error", 675, "Report and block the provider")
	BlockProviderError                                   = sdkerrors.New("BlockProvider Error", 676, "Block the provider")
	SessionOutOfSyncError                                = sdkerrors.New("SessionOutOfSync Error", 677, "Session went out of sync with the provider")
	MaximumNumberOfBlockListedSessionsError              = sdkerrors.New("MaximumNumberOfBlockListedSessions Error", 678, "Provider reached maximum number of block listed sessions.")
	SendRelayError                                       = sdkerrors.New("SendRelay Error", 679, "Failed To Send Relay")
	DataReliabilityIndexRequestedIsOriginalProviderError = sdkerrors.New("DataReliabilityIndexRequestedIsOriginalProvider Error", 680, "Data reliability session index belongs to the original provider")
)

var ( // Provider Side Errors
	InvalidEpochError           = sdkerrors.New("InvalidEpoch Error", 881, "Requested Epoch Is Too Old")
	NewSessionWithRelayNumError = sdkerrors.New("NewSessionWithRelayNum Error", 882, "Requested Session With Relay Number Is Invalid")
	ConsumerIsBlockListed       = sdkerrors.New("ConsumerIsBlockListed Error", 883, "This Consumer Is Blocked.")
)
