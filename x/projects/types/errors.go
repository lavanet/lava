package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/projects module sentinel errors
var (
	ErrInvalidKeyType = sdkerrors.Register(ModuleName, 1100, "invalid project key type")
)
