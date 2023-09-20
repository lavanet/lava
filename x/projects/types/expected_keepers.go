package types

import (
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common"
)

type EpochStorageKeeper interface {
	GetNextEpoch(ctx sdk.Context, block uint64) (nextEpoch uint64, erro error)
	GetEpochStartForBlock(ctx sdk.Context, block uint64) (epochStart, blockInEpoch uint64, err error)
}

type FixationStoreKeeper interface {
	NewFixationStore(storeKey storetypes.StoreKey, prefix string) *common.FixationStore
}
