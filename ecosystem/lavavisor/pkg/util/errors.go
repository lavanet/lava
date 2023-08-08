package lvutil

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var (
	MinVersionMismatchError    = sdkerrors.New("MinVersionMismatchError", 955, "Minimum version mismatch.")
	TargetVersionMismatchError = sdkerrors.New("TargetVersionMismatchError", 955, "Target version mismatch.")
)
