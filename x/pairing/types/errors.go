package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/pairing module sentinel errors
var (
	ErrSample                                          = sdkerrors.Register(ModuleName, 1100, "sample error")
	NoPreviousEpochForAverageBlockTimeCalculationError = sdkerrors.New("NoPreviousEpochForAverageBlockTimeCalculationError Error", 685, "Can't get previous epoch for average block time calculation.")
	PreviousEpochStartIsBlockZeroError                 = sdkerrors.New("PreviousEpochStartIsBlockZeroError Error", 686, "Previous epoch start is block 0, can't be used for average block time calculation (core.Block(0) panics).")
	AverageBlockTimeIsLessOrEqualToZeroError           = sdkerrors.New("AverageBlockTimeIsLessOrEqualToZeroError Error", 687, "The calculated average block time is less or equal to zero")
	NotEnoughBlocksToCalculateAverageBlockTimeError    = sdkerrors.New("NotEnoughBlocksToCalculateAverageBlockTimeError Error", 688, "There isn't enough blocks in the previous epoch to calculate average block time")
	FreezeReasonTooLongError                           = sdkerrors.New("FreezeReasonTooLongError Error", 689, "The freeze reason is too long. Keep the freeze reason less than 50 characters")
	FreezeStakeEntryNotFoundError                      = sdkerrors.New("FreezeStakeEntryNotFoundError Error", 690, "can't get stake entry to freeze")
	MonikerTooLongError                                = sdkerrors.New("MonikerTooLongError Error", 691, "The provider's moniker is too long. Keep it less than 50 characters")
	MonikerEmptyError                                  = sdkerrors.New("MonikerEmptyError Error", 692, "The provider's moniker cannot be empty")
	GeolocationNotMatchWithEndpointsError              = sdkerrors.New("GeolocationNotMatchWithEndpointsError Error", 693, "The combination of the endpoints' geolocation does not match to the provider's geolocation")
	DelegateCommissionOOBError                         = sdkerrors.New("DelegateCommissionOOBError Error", 694, "Delegation commission out of bound [0,100]")
	DelegateLimitError                                 = sdkerrors.New("DelegateLimitError Error", 695, "Delegation limit coin is invalid")
	ProviderRewardError                                = sdkerrors.New("ProviderRewardError Error", 696, "could not calculate provider reward with delegations")
)
