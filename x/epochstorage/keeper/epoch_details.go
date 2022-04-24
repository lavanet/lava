package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/epochstorage/types"
)

// SetEpochDetails set epochDetails in the store
func (k Keeper) SetEpochDetails(ctx sdk.Context, epochDetails types.EpochDetails) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.EpochDetailsKey))
	b := k.cdc.MustMarshal(&epochDetails)
	store.Set([]byte{0}, b)
}

// GetEpochDetails returns epochDetails
func (k Keeper) GetEpochDetails(ctx sdk.Context) (val types.EpochDetails, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.EpochDetailsKey))

	b := store.Get([]byte{0})
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveEpochDetails removes epochDetails from the store
func (k Keeper) RemoveEpochDetails(ctx sdk.Context) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.EpochDetailsKey))
	store.Delete([]byte{0})
}
func (k Keeper) SetEpochDetailsStart(ctx sdk.Context, block uint64) {
	details, found := k.GetEpochDetails(ctx)
	if !found {
		panic("did not find EpochDetails")
	}
	details.StartBlock = block
	k.SetEpochDetails(ctx, details)
}

func (k Keeper) GetEpochStart(ctx sdk.Context) uint64 {
	details, found := k.GetEpochDetails(ctx)
	if !found {
		panic("did not find EpochDetails")
	}
	return details.StartBlock
}

func (k Keeper) GetEarliestEpochStart(ctx sdk.Context) uint64 {
	details, found := k.GetEpochDetails(ctx)
	if !found {
		panic("did not find EpochDetails")
	}
	return details.EarliestStart
}

func (k Keeper) SetEarliestEpochStart(ctx sdk.Context, block uint64) {
	details, found := k.GetEpochDetails(ctx)
	if !found {
		panic("did not find EpochDetails")
	}
	details.EarliestStart = block
	k.SetEpochDetails(ctx, details)
}
