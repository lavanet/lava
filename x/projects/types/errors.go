package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/projects module sentinel errors
var (
	ErrInvalidKeyType = sdkerrors.Register(ModuleName, 1100, "invalid project key type")
)
