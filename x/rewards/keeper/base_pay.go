package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/rewards/types"
)

// SetBasePay set a specific BasePay in the store from its index
func (k Keeper) setBasePay(ctx sdk.Context, index types.BasePayIndex, basePay types.BasePay) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.BasePayPrefix))
	b := k.cdc.MustMarshal(&basePay)
	store.Set([]byte(index.String()), b)
}

// GetBasePay returns a BasePay from its index
func (k Keeper) getBasePay(
	ctx sdk.Context,
	index types.BasePayIndex,
) (val types.BasePay, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.BasePayPrefix))

	b := store.Get([]byte(index.String()))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// GetAllBasePay returns all BasePay
func (k Keeper) GetAllBasePay(ctx sdk.Context) (list []types.BasePayGenesis) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.BasePayPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.BasePay
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, types.BasePayGenesis{Index: string(iterator.Key()), BasePay: val})
	}

	return
}

// SetAllBasePay sets all BasePay
func (k Keeper) SetAllBasePay(ctx sdk.Context, list []types.BasePayGenesis) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.BasePayPrefix))
	for _, basePay := range list {
		b := k.cdc.MustMarshal(&basePay)
		store.Set([]byte(basePay.Index), b)
	}
}

func (k Keeper) getAllBasePayForChain(ctx sdk.Context, chainID string, provider string) (list []types.BasePayWithIndex) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.BasePayPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte(types.BasePayIndex{ChainID: chainID, Provider: provider}.String()))

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.BasePay
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, types.BasePayWithIndex{BasePayIndex: types.BasePayKeyRecover(string(iterator.Key())), BasePay: val})
	}

	return
}

func (k Keeper) popAllBasePayForChain(ctx sdk.Context, chainID string) (list []types.BasePayWithIndex) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.BasePayPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte(chainID))

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.BasePay
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, types.BasePayWithIndex{BasePayIndex: types.BasePayKeyRecover(string(iterator.Key())), BasePay: val})
		store.Delete(iterator.Key())
	}

	return
}

func (k Keeper) removeAllBasePay(ctx sdk.Context) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.BasePayPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		store.Delete(iterator.Key())
	}
}
