package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type EpochstorageKeeper interface {
	BlocksToSaveRaw(ctx sdk.Context) (res uint64)
}
