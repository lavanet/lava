package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/pairing module sentinel errors
var (
	ErrSample                                          = sdkerrors.Register(ModuleName, 1100, "sample error")
	NoPreviousEpochForAverageBlockTimeCalculationError = sdkerrors.New("NoPreviousEpochForAverageBlockTimeCalculationError Error", 685, "Can't get previous epoch for average block time calculation.")
	PreviousEpochStartIsBlockZeroError                 = sdkerrors.New("PreviousEpochStartIsBlockZeroError Error", 686, "Previous epoch start is block 0, can't be used for average block time calculation (core.Block(0) panics).")
	AverageBlockTimeIsLessOrEqualToZeroError           = sdkerrors.New("AverageBlockTimeIsLessOrEqualToZeroError Error", 687, "The calculated average block time is less or equal to zero")
	NotEnoughBlocksToCalculateAverageBlockTimeError    = sdkerrors.New("NotEnoughBlocksToCalculateAverageBlockTimeError Error", 688, "There isn't enough blocks in the previous epoch to calculate average block time")
	FreezeReasonTooLongError                           = sdkerrors.New("FreezeReasonTooLongError Error", 689, "The freeze reason is too long. Keep the freeze reason less than 50 characters")
)
