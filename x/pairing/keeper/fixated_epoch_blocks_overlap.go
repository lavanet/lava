package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/pairing/types"
)

// SetFixatedEpochBlocksOverlap set a specific fixatedEpochBlocksOverlap in the store from its index
func (k Keeper) SetFixatedEpochBlocksOverlap(ctx sdk.Context, fixatedEpochBlocksOverlap types.FixatedEpochBlocksOverlap) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.FixatedEpochBlocksOverlapKeyPrefix))
	b := k.cdc.MustMarshal(&fixatedEpochBlocksOverlap)
	store.Set(types.FixatedEpochBlocksOverlapKey(
		fixatedEpochBlocksOverlap.Index,
	), b)
}

// GetFixatedEpochBlocksOverlap returns a fixatedEpochBlocksOverlap from its index
func (k Keeper) GetFixatedEpochBlocksOverlap(
	ctx sdk.Context,
	index string,

) (val types.FixatedEpochBlocksOverlap, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.FixatedEpochBlocksOverlapKeyPrefix))

	b := store.Get(types.FixatedEpochBlocksOverlapKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveFixatedEpochBlocksOverlap removes a fixatedEpochBlocksOverlap from the store
func (k Keeper) RemoveFixatedEpochBlocksOverlap(
	ctx sdk.Context,
	index string,

) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.FixatedEpochBlocksOverlapKeyPrefix))
	store.Delete(types.FixatedEpochBlocksOverlapKey(
		index,
	))
}

// GetAllFixatedEpochBlocksOverlap returns all fixatedEpochBlocksOverlap
func (k Keeper) GetAllFixatedEpochBlocksOverlap(ctx sdk.Context) (list []types.FixatedEpochBlocksOverlap) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.FixatedEpochBlocksOverlapKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.FixatedEpochBlocksOverlap
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
