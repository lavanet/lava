package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/rewards/types"
)

func (k Keeper) AggregateRewards(ctx sdk.Context, provider, chainid string, adjustmentDenom uint64, rewards math.Int) {
	index := types.BasePayIndex{Provider: provider, ChainID: chainid}
	basepay, found := k.getBasePay(ctx, index)
	if !found {
		basepay = types.BasePay{Total: rewards, TotalAdjusted: rewards.MulRaw(int64(adjustmentDenom))}
	} else {
		basepay.Total = basepay.Total.Add(rewards)
		basepay.TotalAdjusted = basepay.TotalAdjusted.Add(rewards.MulRaw(int64(adjustmentDenom)))
	}

	k.setBasePay(ctx, index, basepay)
}

func (k Keeper) DistributeMonthlyBonusRewards(ctx sdk.Context) {
	total := sdk.ZeroInt() // TODO yarom, get the pool balance
	specs := k.SpecEmissionParts(ctx)
	for _, spec := range specs {
		basepays, totalbasepay := k.SpecProvidersBasePay(ctx, spec.ChainID)
		specTotalPayout := k.SpecTotalPayout(ctx, spec.Emission.MulInt(total).TruncateInt(), sdk.NewDecFromInt(totalbasepay), spec)
		for _, basepay := range basepays {
			reward := specTotalPayout.MulInt(basepay.TotalAdjusted).QuoInt(basepay.Total)
			// now give the reward somehow
			_ = reward
		}
	}
}

func (k Keeper) SpecTotalPayout(ctx sdk.Context, totalMonthlyPayout math.Int, totalProvidersBaseRewards sdk.Dec, spec types.SpecEmmisionPart) math.LegacyDec {
	specPayoutAllocation := spec.Emission.MulInt(totalMonthlyPayout)
	rewardBoost := totalProvidersBaseRewards.MulInt64(int64(k.GetParams(ctx).Providers.MaxRewardBoost))
	anotherShit := sdk.MaxDec(sdk.ZeroDec(), specPayoutAllocation.Mul(sdk.NewDecWithPrec(15, 1).Sub(totalProvidersBaseRewards.Mul(sdk.NewDecWithPrec(5, 1)))))
	return sdk.MinDec(sdk.MinDec(specPayoutAllocation, rewardBoost), anotherShit)
}

func (k Keeper) SpecEmissionParts(ctx sdk.Context) (emisions []types.SpecEmmisionPart) {
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
			emisions = append(emisions, types.SpecEmmisionPart{ChainID: chainID, Emission: stake.Quo(totalStake)})
		}
	}

	return emisions
}

func (k Keeper) SpecProvidersBasePay(ctx sdk.Context, chainID string) ([]types.BasePayWithIndex, math.Int) {
	basepays := k.getAllBasePayForChain(ctx, chainID)
	totalBasePay := math.ZeroInt()
	for _, basepay := range basepays {
		totalBasePay = totalBasePay.Add(basepay.Total)
	}
	return basepays, totalBasePay
}
