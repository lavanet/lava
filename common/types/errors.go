package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/pairing module sentinel errors
var (
	ErrEntryNotFound = sdkerrors.Register(MODULE_NAME, 1, "entry not found")
	ErrInvalidIndex  = sdkerrors.Register(MODULE_NAME, 2, "entry index is invalid")
)
