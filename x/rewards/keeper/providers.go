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
	adjustedPay = sdk.MinDec(adjustedPay, sdk.NewDecFromInt(rewards))
	if !found {
		basepay = types.BasePay{Total: rewards, TotalAdjusted: adjustedPay}
	} else {
		basepay.Total = basepay.Total.Add(rewards)
		basepay.TotalAdjusted = basepay.TotalAdjusted.Add(adjustedPay)
	}

	k.setBasePay(ctx, index, basepay)
}

// Distribute bonus rewards to providers across all chains based on performance
func (k Keeper) distributeMonthlyBonusRewards(ctx sdk.Context) {
	total := k.TotalPoolTokens(ctx, types.ProviderRewardsDistributionPool)
	totalRewarded := sdk.ZeroInt()
	// specs emissions from the total reward pool base on stake
	specs := k.specEmissionParts(ctx)
	for _, spec := range specs {
		// all providers basepays and the total basepay of the spec
		basepays, totalbasepay := k.specProvidersBasePay(ctx, spec.ChainID)
		if totalbasepay.IsZero() {
			continue
		}

		// calculate the maximum rewards for the spec
		specTotalPayout := k.specTotalPayout(ctx, total, sdk.NewDecFromInt(totalbasepay), spec)
		// distribute the rewards to all providers
		for _, basepay := range basepays {
			// calculate the providers bonus base on adjusted base pay
			reward := specTotalPayout.Mul(basepay.TotalAdjusted).QuoInt(totalbasepay).TruncateInt()
			totalRewarded = totalRewarded.Add(reward)
			if totalRewarded.GT(total) {
				utils.LavaFormatError("provider rewards are larger than the distribution pool balance", nil,
					utils.LogAttr("distribution_pool_balance", total.String()),
					utils.LogAttr("provider_reward", totalRewarded.String()))
				return
			}
			// now give the reward the provider contributor and delegators
			providerAddr, err := sdk.AccAddressFromBech32(basepay.Provider)
			if err != nil {
				continue
			}
			_, _, err = k.dualstakingKeeper.RewardProvidersAndDelegators(ctx, providerAddr, basepay.ChainID, reward, string(types.ProviderRewardsDistributionPool), false, false, false)
			if err != nil {
				utils.LavaFormatError("failed to send bonus rewards to provider", err, utils.LogAttr("provider", basepay.Provider))
			}
		}
	}
	k.removeAllBasePay(ctx)
}

// specTotalPayout calculates the total bonus for a specific spec
// specPayoutAllocation: maximum rewards that the spec can have
// rewardBoost: bonus based on the total rewards providers got factored by maxboost
// diminishingRewards: makes sure to diminish the bonuses in case there are enough consumers on the chain
func (k Keeper) specTotalPayout(ctx sdk.Context, totalMonthlyPayout math.Int, totalProvidersBaseRewards sdk.Dec, spec types.SpecEmissionPart) math.LegacyDec {
	specPayoutAllocation := spec.Emission.MulInt(totalMonthlyPayout)
	rewardBoost := totalProvidersBaseRewards.MulInt64(int64(k.MaxRewardBoost(ctx)))
	diminishingRewards := sdk.MaxDec(sdk.ZeroDec(), (sdk.NewDecWithPrec(15, 1).Mul(specPayoutAllocation)).Sub(sdk.NewDecWithPrec(5, 1).Mul(totalProvidersBaseRewards)))
	return sdk.MinDec(sdk.MinDec(specPayoutAllocation, rewardBoost), diminishingRewards)
}

func (k Keeper) specEmissionParts(ctx sdk.Context) (emissions []types.SpecEmissionPart) {
	chainIDs := k.specKeeper.GetAllChainIDs(ctx)
	totalStake := sdk.ZeroDec()
	chainStake := map[string]sdk.Dec{}
	for _, chainID := range chainIDs {
		spec, found := k.specKeeper.GetSpec(ctx, chainID)
		if !found {
			continue
		}

		if !spec.Enabled || spec.Shares == 0 {
			continue
		}

		stakeStorage, found := k.epochstorage.GetStakeStorageCurrent(ctx, chainID)
		if !found {
			continue
		}
		chainStake[chainID] = sdk.ZeroDec()
		for _, entry := range stakeStorage.StakeEntries {
			chainStake[chainID] = chainStake[chainID].Add(sdk.NewDecFromInt(entry.EffectiveStake()))
		}

		chainStake[chainID] = chainStake[chainID].MulInt64(int64(spec.Shares))
		totalStake = totalStake.Add(chainStake[chainID])
	}

	if totalStake.IsZero() {
		return emissions
	}

	for _, chainID := range chainIDs {
		if stake, ok := chainStake[chainID]; ok {
			if stake.IsZero() {
				continue
			}

			emissions = append(emissions, types.SpecEmissionPart{ChainID: chainID, Emission: stake.Quo(totalStake)})
		}
	}

	return emissions
}

func (k Keeper) specProvidersBasePay(ctx sdk.Context, chainID string) ([]types.BasePayWithIndex, math.Int) {
	basepays := k.popAllBasePayForChain(ctx, chainID)
	totalBasePay := math.ZeroInt()
	for _, basepay := range basepays {
		totalBasePay = totalBasePay.Add(basepay.Total)
	}
	return basepays, totalBasePay
}
