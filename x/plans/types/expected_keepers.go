package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type EpochStorageKeeper interface {
	GetEpochStart(ctx sdk.Context) uint64
	GetNextEpoch(ctx sdk.Context, block uint64) (uint64, error)
}
