package types

import (
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	fixationstoretypes "github.com/lavanet/lava/v2/x/fixationstore/types"
)

type EpochStorageKeeper interface {
	GetNextEpoch(ctx sdk.Context, block uint64) (nextEpoch uint64, erro error)
	GetEpochStartForBlock(ctx sdk.Context, block uint64) (epochStart, blockInEpoch uint64, err error)
	BlocksToSaveRaw(ctx sdk.Context) (res uint64)
}

type FixationStoreKeeper interface {
	NewFixationStore(storeKey storetypes.StoreKey, prefix string) *fixationstoretypes.FixationStore
}
