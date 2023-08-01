package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/pairing module sentinel errors
var (
	ErrInvalidIndex = sdkerrors.Register(MODULE_NAME, 1, "entry index is invalid")
)
