package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/subscription module sentinel errors
var (
	ErrSample           = sdkerrors.Register(ModuleName, 1100, "sample error")
	ErrBlankParameter   = sdkerrors.New(ModuleName, 101, "required parameter is empty")
	ErrInvalidParameter = sdkerrors.New(ModuleName, 102, "required parameter is invalid")
)
