package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/rewards/types"
)

// SetBasePay set a specific BasePay in the store from its index
func (k Keeper) setBasePay(ctx sdk.Context, index types.BasePayWithIndex) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.BasePayPrefix))
	b := k.cdc.MustMarshal(&index.BasePay)
	store.Set([]byte(index.Index()), b)
}

// GetBasePay returns a BasePay from its index
func (k Keeper) getBasePay(
	ctx sdk.Context,
	index types.BasePayWithIndex,
) (val types.BasePay, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.BasePayPrefix))

	b := store.Get([]byte(index.Index()))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// GetAllBasePay returns all BasePay
func (k Keeper) GetAllBasePay(ctx sdk.Context) (list []types.BasePayWithIndex) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.BasePayPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.BasePay
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		bs := types.BasePayKeyRecover(string(iterator.Key()))
		bs.BasePay = val
		list = append(list, bs)
	}

	return
}

// SetAllBasePay sets all BasePay
func (k Keeper) SetAllBasePay(ctx sdk.Context, list []types.BasePayWithIndex) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.BasePayPrefix))
	for _, basePay := range list {
		b := k.cdc.MustMarshal(&basePay)
		store.Set([]byte(basePay.Index()), b)
	}
}

func (k Keeper) getAllBasePayForChain(ctx sdk.Context, chainID string, provider string) (list []types.BasePayWithIndex) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.BasePayPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte(types.BasePayWithIndex{ChainId: chainID, Provider: provider}.Index()))

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.BasePay
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		bs := types.BasePayKeyRecover(string(iterator.Key()))
		bs.BasePay = val
		list = append(list, bs)
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
		bs := types.BasePayKeyRecover(string(iterator.Key()))
		bs.BasePay = val
		list = append(list, bs)
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
