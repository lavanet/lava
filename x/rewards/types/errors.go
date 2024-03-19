package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/rewards module sentinel errors
var (
	ErrFundIprpc = sdkerrors.Register(ModuleName, 1, "fund iprpc TX failed")
)
