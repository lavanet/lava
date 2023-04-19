package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/pairing module sentinel errors
var (
	ErrInvalidIndex = sdkerrors.Register(MODULE_NAME, 1, "entry index is invalid")
)
