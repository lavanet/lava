package types

// DONTCOVER

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// x/spec module sentinel errors
var (
	ErrSample = sdkerrors.Register(ModuleName, 1100, "sample error")

	ErrUnknownSubspace  = sdkerrors.Register(ModuleName, 2, "unknown subspace")
	ErrSettingParameter = sdkerrors.Register(ModuleName, 3, "failed to set parameter")
	ErrEmptyChanges     = sdkerrors.Register(ModuleName, 4, "submitted parameter changes are empty")
	ErrEmptySubspace    = sdkerrors.Register(ModuleName, 5, "parameter subspace is empty")
	ErrEmptyKey         = sdkerrors.Register(ModuleName, 6, "parameter key is empty")
	ErrEmptyValue       = sdkerrors.Register(ModuleName, 7, "parameter value is empty")
)
