package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
)

// SetSessionStorageForSpec set a specific sessionStorageForSpec in the store from its index
func (k Keeper) SetSessionStorageForSpec(ctx sdk.Context, sessionStorageForSpec types.SessionStorageForSpec) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SessionStorageForSpecKeyPrefix))
	b := k.cdc.MustMarshal(&sessionStorageForSpec)
	store.Set(types.SessionStorageForSpecKey(
		sessionStorageForSpec.Index,
	), b)
}

// GetSessionStorageForSpec returns a sessionStorageForSpec from its index
func (k Keeper) GetSessionStorageForSpec(
	ctx sdk.Context,
	index string,

) (val types.SessionStorageForSpec, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SessionStorageForSpecKeyPrefix))

	b := store.Get(types.SessionStorageForSpecKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveSessionStorageForSpec removes a sessionStorageForSpec from the store
func (k Keeper) RemoveSessionStorageForSpec(
	ctx sdk.Context,
	index string,

) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SessionStorageForSpecKeyPrefix))
	store.Delete(types.SessionStorageForSpecKey(
		index,
	))
}

// GetAllSessionStorageForSpec returns all sessionStorageForSpec
func (k Keeper) GetAllSessionStorageForSpec(ctx sdk.Context) (list []types.SessionStorageForSpec) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SessionStorageForSpecKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.SessionStorageForSpec
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
