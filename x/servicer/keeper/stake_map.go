package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
)

// SetStakeMap set a specific stakeMap in the store from its index
func (k Keeper) SetStakeMap(ctx sdk.Context, stakeMap types.StakeMap) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.StakeMapKeyPrefix))
	b := k.cdc.MustMarshal(&stakeMap)
	store.Set(types.StakeMapKey(
		stakeMap.Index,
	), b)
}

// GetStakeMap returns a stakeMap from its index
func (k Keeper) GetStakeMap(
	ctx sdk.Context,
	index string,

) (val types.StakeMap, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.StakeMapKeyPrefix))

	b := store.Get(types.StakeMapKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveStakeMap removes a stakeMap from the store
func (k Keeper) RemoveStakeMap(
	ctx sdk.Context,
	index string,

) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.StakeMapKeyPrefix))
	store.Delete(types.StakeMapKey(
		index,
	))
}

// GetAllStakeMap returns all stakeMap
func (k Keeper) GetAllStakeMap(ctx sdk.Context) (list []types.StakeMap) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.StakeMapKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.StakeMap
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
