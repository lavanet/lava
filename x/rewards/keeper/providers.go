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

func (k Keeper) SpecEmissions(ctx sdk.Context) (emisions []types.SpecEmmision) {
	chainIDs := k.specKeeper.GetAllChainIDs(ctx)
	totalStake := sdk.ZeroDec()
	chainStake := map[string]sdk.Dec{}
	for _, chainID := range chainIDs {
		stakeStorage, found := k.epochstorage.GetStakeStorageCurrent(ctx, chainID)
		if !found {
			continue
		}
		chainStake[chainID] = sdk.ZeroDec()
		for _, entry := range stakeStorage.StakeEntries {
			chainStake[chainID] = chainStake[chainID].Add(sdk.NewDecFromInt(entry.EffectiveStake()))
		}

		spec, _ := k.specKeeper.GetSpec(ctx, chainID)
		chainStake[chainID] = chainStake[chainID].MulInt64(int64(spec.Shares))
		totalStake = totalStake.Add(chainStake[chainID])
	}

	for _, chainID := range chainIDs {
		if stake, ok := chainStake[chainID]; ok {
			emisions = append(emisions, types.SpecEmmision{ChainID: chainID, Emission: stake.Quo(totalStake)})
		}
	}

	return emisions
}
