package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/pairing/types"
)

// SetFixatedStakeToMaxCu set a specific fixatedStakeToMaxCu in the store from its index
func (k Keeper) SetFixatedStakeToMaxCu(ctx sdk.Context, fixatedStakeToMaxCu types.FixatedStakeToMaxCu) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.FixatedStakeToMaxCuKeyPrefix))
	b := k.cdc.MustMarshal(&fixatedStakeToMaxCu)
	store.Set(types.FixatedStakeToMaxCuKey(
		fixatedStakeToMaxCu.Index,
	), b)
}

// GetFixatedStakeToMaxCu returns a fixatedStakeToMaxCu from its index
func (k Keeper) GetFixatedStakeToMaxCu(
	ctx sdk.Context,
	index string,

) (val types.FixatedStakeToMaxCu, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.FixatedStakeToMaxCuKeyPrefix))

	b := store.Get(types.FixatedStakeToMaxCuKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveFixatedStakeToMaxCu removes a fixatedStakeToMaxCu from the store
func (k Keeper) RemoveFixatedStakeToMaxCu(
	ctx sdk.Context,
	index string,

) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.FixatedStakeToMaxCuKeyPrefix))
	store.Delete(types.FixatedStakeToMaxCuKey(
		index,
	))
}

// GetAllFixatedStakeToMaxCu returns all fixatedStakeToMaxCu
func (k Keeper) GetAllFixatedStakeToMaxCu(ctx sdk.Context) (list []types.FixatedStakeToMaxCu) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.FixatedStakeToMaxCuKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.FixatedStakeToMaxCu
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}
