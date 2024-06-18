package types

// DONTCOVER

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/rewards module sentinel errors
var (
	ErrFundIprpc                   = sdkerrors.Register(ModuleName, 1, "fund iprpc TX failed")
	ErrMemoNotIprpcOverIbc         = sdkerrors.Register(ModuleName, 2, "ibc-transfer packet's memo is not in the right format of IPRPC over IBC")
	ErrIprpcMemoInvalid            = sdkerrors.Register(ModuleName, 3, "ibc-transfer packet's memo of IPRPC over IBC is invalid")
	ErrCoverIbcIprpcFundCostFailed = sdkerrors.Register(ModuleName, 4, "cover ibc iprpc fund cost failed")
)
