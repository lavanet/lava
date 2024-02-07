package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/rewards/types"
)

// SetIprpcSubscription set a subscription in the IprpcSubscription store
func (k Keeper) SetIprpcSubscription(ctx sdk.Context, address string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.IprpcSubscriptionPrefix))
	store.Set([]byte(address), []byte{})
}

// IsIprpcSubscription checks whether a subscription is IPRPC eligible subscription
func (k Keeper) IsIprpcSubscription(ctx sdk.Context, address string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.IprpcSubscriptionPrefix))
	b := store.Get([]byte(address))
	return b != nil
}

// RemoveIprpcSubscription removes a subscription from the IprpcSubscription store
func (k Keeper) RemoveIprpcSubscription(ctx sdk.Context, address string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.IprpcSubscriptionPrefix))
	store.Delete([]byte(address))
}

// GetAllIprpcSubscription returns all subscription from the IprpcSubscription store
func (k Keeper) GetAllIprpcSubscription(ctx sdk.Context) []string {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.IprpcSubscriptionPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	list := []string{}
	for ; iterator.Valid(); iterator.Next() {
		list = append(list, string(iterator.Key()))
	}

	return list
}
