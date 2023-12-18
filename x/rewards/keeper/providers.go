package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/rewards/types"
)

func (k Keeper) AggregateRewards(ctx sdk.Context, provider, chainid string, adjustmentDenom uint64, rewards math.Int) {
	index := types.BasePayIndex{Provider: provider, ChainID: chainid}
	basepay, found := k.getBasePay(ctx, index)
	adjustedPay := sdk.NewDecFromInt(rewards).QuoInt64(int64(adjustmentDenom))
	if !found {
		basepay = types.BasePay{Total: rewards, TotalAdjusted: adjustedPay}
	} else {
		basepay.Total = basepay.Total.Add(rewards)
		basepay.TotalAdjusted = basepay.TotalAdjusted.Add(adjustedPay)
	}

	k.setBasePay(ctx, index, basepay)
}

func (k Keeper) DistributeMonthlyBonusRewards(ctx sdk.Context) {
	total := k.TotalPoolTokens(ctx, types.ProviderDistributionPool)
	totalRewarded := sdk.ZeroInt()
	specs := k.SpecEmissionParts(ctx)
	for _, spec := range specs {
		basepays, totalbasepay := k.SpecProvidersBasePay(ctx, spec.ChainID)
		specTotalPayout := k.SpecTotalPayout(ctx, total, sdk.NewDecFromInt(totalbasepay), spec)
		for _, basepay := range basepays {
			reward := specTotalPayout.Mul(basepay.TotalAdjusted).QuoInt(totalbasepay).TruncateInt()
			totalRewarded = totalRewarded.Add(reward)
			if totalRewarded.GT(total) {
				utils.LavaFormatError("trying to send more than we can", nil, utils.LogAttr("total", total.String()), utils.LogAttr("totalRewarded", totalRewarded.String()))
				return
			}
			// now give the reward the provider contributor and delegators
			providerAddr, err := sdk.AccAddressFromBech32(basepay.Provider)
			if err != nil {
				continue
			}
			_, err = k.dualstakingKeeper.RewardProvidersAndDelegators(ctx, providerAddr, basepay.ChainID, reward, "reward pool", false, false, false)
			if err != nil {
				utils.LavaFormatError("Failed to send bonus rewards to provider", err, utils.LogAttr("provider", basepay.Provider))
			}
		}
	}
	k.removeAllBasePay(ctx)
	tokensToBurn := total.Sub(totalRewarded)
	k.BurnPoolTokens(ctx, types.ProviderDistributionPool, tokensToBurn)
}

func (k Keeper) SpecTotalPayout(ctx sdk.Context, totalMonthlyPayout math.Int, totalProvidersBaseRewards sdk.Dec, spec types.SpecEmmisionPart) math.LegacyDec {
	specPayoutAllocation := spec.Emission.MulInt(totalMonthlyPayout)
	// TODO yarom move the param read to a method
	rewardBoost := totalProvidersBaseRewards.MulInt64(int64(k.GetParams(ctx).Providers.MaxRewardBoost))
	diminishingRewards := sdk.MaxDec(sdk.ZeroDec(), (sdk.NewDecWithPrec(15, 1).Mul(specPayoutAllocation)).Sub(sdk.NewDecWithPrec(5, 1).Mul(totalProvidersBaseRewards)))
	return sdk.MinDec(sdk.MinDec(specPayoutAllocation, rewardBoost), diminishingRewards)
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
			if stake.IsZero() {
				continue
			}

			emisions = append(emisions, types.SpecEmmisionPart{ChainID: chainID, Emission: stake.Quo(totalStake)})
		}
	}

	return emisions
}

func (k Keeper) SpecProvidersBasePay(ctx sdk.Context, chainID string) ([]types.BasePayWithIndex, math.Int) {
	basepays := k.popAllBasePayForChain(ctx, chainID)
	totalBasePay := math.ZeroInt()
	for _, basepay := range basepays {
		totalBasePay = totalBasePay.Add(basepay.Total)
	}
	return basepays, totalBasePay
}
