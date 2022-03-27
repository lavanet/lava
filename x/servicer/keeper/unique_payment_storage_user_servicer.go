package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
)

// SetUniquePaymentStorageUserServicer set a specific uniquePaymentStorageUserServicer in the store from its index
func (k Keeper) SetUniquePaymentStorageUserServicer(ctx sdk.Context, uniquePaymentStorageUserServicer types.UniquePaymentStorageUserServicer) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.UniquePaymentStorageUserServicerKeyPrefix))
	b := k.cdc.MustMarshal(&uniquePaymentStorageUserServicer)
	store.Set(types.UniquePaymentStorageUserServicerKey(
		uniquePaymentStorageUserServicer.Index,
	), b)
}

// GetUniquePaymentStorageUserServicer returns a uniquePaymentStorageUserServicer from its index
func (k Keeper) GetUniquePaymentStorageUserServicer(
	ctx sdk.Context,
	index string,

) (val types.UniquePaymentStorageUserServicer, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.UniquePaymentStorageUserServicerKeyPrefix))

	b := store.Get(types.UniquePaymentStorageUserServicerKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveUniquePaymentStorageUserServicer removes a uniquePaymentStorageUserServicer from the store
func (k Keeper) RemoveUniquePaymentStorageUserServicer(
	ctx sdk.Context,
	index string,

) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.UniquePaymentStorageUserServicerKeyPrefix))
	store.Delete(types.UniquePaymentStorageUserServicerKey(
		index,
	))
}

// GetAllUniquePaymentStorageUserServicer returns all uniquePaymentStorageUserServicer
func (k Keeper) GetAllUniquePaymentStorageUserServicer(ctx sdk.Context) (list []types.UniquePaymentStorageUserServicer) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.UniquePaymentStorageUserServicerKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.UniquePaymentStorageUserServicer
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) AddUniquePaymentStorageUserServicer(ctx sdk.Context,
	block types.BlockNum, userAddress sdk.AccAddress, servicerAddress sdk.AccAddress, uniqueIdentifier string) (bool, *types.UniquePaymentStorageUserServicer) {
	key := userAddress.String() + servicerAddress.String() + uniqueIdentifier
	entry, found := k.GetUniquePaymentStorageUserServicer(ctx, key)
	if found {
		return false, &entry
	}
	entry = types.UniquePaymentStorageUserServicer{Index: key, Block: block.Num}
	k.SetUniquePaymentStorageUserServicer(ctx, entry)
	return true, &entry
}
