package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/user/types"
)

// SetSpecStakeStorage set a specific specStakeStorage in the store from its index
func (k Keeper) SetSpecStakeStorage(ctx sdk.Context, specStakeStorage types.SpecStakeStorage) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SpecStakeStorageKeyPrefix))
	b := k.cdc.MustMarshal(&specStakeStorage)
	store.Set(types.SpecStakeStorageKey(
		specStakeStorage.Index,
	), b)
}

// GetSpecStakeStorage returns a specStakeStorage from its index
func (k Keeper) GetSpecStakeStorage(
	ctx sdk.Context,
	index string,

) (val types.SpecStakeStorage, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SpecStakeStorageKeyPrefix))

	b := store.Get(types.SpecStakeStorageKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveSpecStakeStorage removes a specStakeStorage from the store
func (k Keeper) RemoveSpecStakeStorage(
	ctx sdk.Context,
	index string,

) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SpecStakeStorageKeyPrefix))
	store.Delete(types.SpecStakeStorageKey(
		index,
	))
}

// GetAllSpecStakeStorage returns all specStakeStorage
func (k Keeper) GetAllSpecStakeStorage(ctx sdk.Context) (list []types.SpecStakeStorage) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SpecStakeStorageKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.SpecStakeStorage
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
