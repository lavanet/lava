package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/rewards/types"
)

func (k Keeper) SetIprpcData(ctx sdk.Context, cost sdk.Coin, subs []string) error {
	if cost.Denom != k.stakingKeeper.BondDenom(ctx) {
		return utils.LavaFormatWarning("min iprpc cost must be in acceptable denom", fmt.Errorf("invalid cost"),
			utils.LogAttr("cost", cost.String()),
			utils.LogAttr("acceptable_denom", k.stakingKeeper.BondDenom(ctx)),
		)
	}
	k.SetMinIprpcCost(ctx, cost)

	for _, sub := range subs {
		k.SetIprpcSubscription(ctx, sub)
	}

	return nil
}

/********************** Min IPRPC Cost **********************/

// SetMinIprpcCost sets the min iprpc cost
func (k Keeper) SetMinIprpcCost(ctx sdk.Context, cost sdk.Coin) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.MinIprpcCostPrefix))
	b := k.cdc.MustMarshal(&cost)
	store.Set([]byte{0}, b)
}

// GetMinIprpcCost gets the min iprpc cost
func (k Keeper) GetMinIprpcCost(ctx sdk.Context) sdk.Coin {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.MinIprpcCostPrefix))
	b := store.Get([]byte{0})
	var cost sdk.Coin
	k.cdc.MustUnmarshal(b, &cost)
	return cost
}

/********************** IPRPC Subscription **********************/

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
