package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/rewards/types"
)

func (k Keeper) AggregateRewards(ctx sdk.Context, provider, chainid, subscription string, rewards math.Int) {
	index := types.BasePayIndex{Provider: provider, ChainID: chainid, Subscription: subscription}
	basepay, found := k.getBasePay(ctx, index)
	if !found {
		basepay = types.BasePay{Total: rewards}
	} else {
		basepay.Total = basepay.Total.Add(rewards)
	}

	k.setBasePay(ctx, index, basepay)
}

func (k Keeper) DistributeMonthlyBonusRewards(ctx sdk.Context) {
	_ = ""
}
