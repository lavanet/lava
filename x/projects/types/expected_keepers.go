package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type EpochStorageKeeper interface {
	GetNextEpoch(ctx sdk.Context, block uint64) (nextEpoch uint64, erro error)
}
