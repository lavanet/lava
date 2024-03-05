package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/rewards/types"
)

const DAY_SECONDS = 60 * 60 * 24

func (k Keeper) AggregateCU(ctx sdk.Context, provider string, chainID string, cu uint64) {
	index := types.BasePayIndex{Provider: provider, ChainID: chainID}
	basepay, found := k.getBasePay(ctx, index)
	if !found {
		basepay = types.BasePay{IprpcCu: cu}
	} else {
		basepay.IprpcCu += cu
	}
	k.setBasePay(ctx, index, basepay)
}

func (k Keeper) AggregateRewards(ctx sdk.Context, provider, chainid string, adjustment sdk.Dec, rewards math.Int) {
	index := types.BasePayIndex{Provider: provider, ChainID: chainid}
	basepay, found := k.getBasePay(ctx, index)
	adjustedPay := adjustment.MulInt(rewards)
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
	details := map[string]string{}
	coins := k.TotalPoolTokens(ctx, types.ProviderRewardsDistributionPool)
	total := coins.AmountOf(k.stakingKeeper.BondDenom(ctx))
	totalRewarded := sdk.ZeroInt()
	// specs emissions from the total reward pool base on stake
	specs := k.specEmissionParts(ctx)

	defer func() {
		k.removeAllBasePay(ctx)
		utils.LogLavaEvent(ctx, k.Logger(ctx), types.ProvidersBonusRewardsEventName, details, "provider bonus rewards distributed successfully")
	}()

	// Get serviced CU for each provider + spec
	specCuMap := map[string]types.SpecCuType{} // spec -> specCu
	for _, spec := range specs {
		// all providers basepays and the total basepay of the spec
		basepays, totalbasepay := k.specProvidersBasePay(ctx, spec.ChainID, true)
		if len(basepays) == 0 {
			continue
		}

		// calculate the maximum rewards for the spec
		specTotalPayout := math.LegacyZeroDec()
		if !totalbasepay.IsZero() {
			specTotalPayout = k.specTotalPayout(ctx, total, sdk.NewDecFromInt(totalbasepay), spec)
		}
		// distribute the rewards to all providers
		for _, basepay := range basepays {
			if !specTotalPayout.IsZero() {
				// calculate the providers bonus base on adjusted base pay
				reward := specTotalPayout.Mul(basepay.TotalAdjusted).QuoInt(totalbasepay).TruncateInt()
				totalRewarded = totalRewarded.Add(reward)
				if totalRewarded.GT(total) {
					utils.LavaFormatError("provider rewards are larger than the distribution pool balance", nil,
						utils.LogAttr("distribution_pool_balance", total.String()),
						utils.LogAttr("provider_reward", totalRewarded.String()))
					details["error"] = "provider rewards are larger than the distribution pool balance"
					return
				}
				// now give the reward the provider contributor and delegators
				providerAddr, err := sdk.AccAddressFromBech32(basepay.Provider)
				if err != nil {
					continue
				}
				_, _, err = k.dualstakingKeeper.RewardProvidersAndDelegators(ctx, providerAddr, basepay.ChainID, sdk.NewCoins(sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), reward)), string(types.ProviderRewardsDistributionPool), false, false, false)
				if err != nil {
					utils.LavaFormatError("failed to send bonus rewards to provider", err, utils.LogAttr("provider", basepay.Provider))
				}

				details[providerAddr.String()+" "+spec.ChainID] = reward.String()
			}

			// count iprpc cu
			k.countIprpcCu(specCuMap, basepay.IprpcCu, spec.ChainID, basepay.Provider)
		}
	}

	// Get current month IprpcReward and use it to distribute rewards
	iprpcReward, found := k.PopIprpcReward(ctx, true)
	if !found {
		utils.LavaFormatError("current month iprpc reward not found", fmt.Errorf("did not reward providers IPRPC bonus"))
		return
	}
	k.RemoveIprpcReward(ctx, iprpcReward.Id)

	// none of the providers will get the IPRPC reward this month, transfer the funds to the next month
	if len(specCuMap) == 0 {
		k.handleNoIprpcRewardToProviders(ctx, iprpcReward)
		return
	}

	// distribute IPRPC rewards
	k.distributeIprpcRewards(ctx, iprpcReward, specCuMap)
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

func (k Keeper) specProvidersBasePay(ctx sdk.Context, chainID string, pop bool) ([]types.BasePayWithIndex, math.Int) {
	var basepays []types.BasePayWithIndex
	if pop {
		basepays = k.popAllBasePayForChain(ctx, chainID)
	} else {
		basepays = k.getAllBasePayForChain(ctx, chainID)
	}

	totalBasePay := math.ZeroInt()
	for _, basepay := range basepays {
		totalBasePay = totalBasePay.Add(basepay.Total)
	}
	return basepays, totalBasePay
}

// ContributeToValidatorsAndCommunityPool transfers some of the providers' rewards to the validators and community pool
// the function return the updated reward after the participation deduction
func (k Keeper) ContributeToValidatorsAndCommunityPool(ctx sdk.Context, reward sdk.Coin, senderModule string) (updatedReward sdk.Coin, err error) {
	// calculate validators and community participation fractions
	validatorsParticipation, communityParticipation, err := k.CalculateContributionPercentages(ctx, reward.Amount)
	if err != nil {
		return reward, err
	}

	if communityParticipation.Equal(sdk.OneDec()) {
		err := k.FundCommunityPoolFromModule(ctx, sdk.NewCoins(reward), senderModule)
		if err != nil {
			return reward, utils.LavaFormatError("failed funding the community pool with whole reward", err,
				utils.Attribute{Key: "reward", Value: reward.String()},
				utils.Attribute{Key: "community_participation", Value: communityParticipation.String()},
			)
		}
		return sdk.NewCoin(reward.Denom, math.ZeroInt()), nil
	}

	// send validators participation
	validatorsParticipationReward := validatorsParticipation.MulInt(reward.Amount).TruncateInt()
	if !validatorsParticipationReward.IsZero() {
		coins := sdk.NewCoins(sdk.NewCoin(reward.Denom, validatorsParticipationReward))
		pool := types.ValidatorsRewardsDistributionPoolName
		if k.isEndOfMonth(ctx) {
			pool = types.ValidatorsRewardsAllocationPoolName
		}
		err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, senderModule, string(pool), coins)
		if err != nil {
			return reward, utils.LavaFormatError("sending validators participation failed", err,
				utils.Attribute{Key: "validators_participation_reward", Value: coins.String()},
				utils.Attribute{Key: "validators_participation", Value: validatorsParticipation.String()},
				utils.Attribute{Key: "reward", Value: reward.String()},
			)
		}
	}

	// send community participation
	communityParticipationReward := communityParticipation.MulInt(reward.Amount).TruncateInt()
	if !communityParticipationReward.IsZero() {
		err = k.FundCommunityPoolFromModule(ctx, sdk.NewCoins(sdk.NewCoin(reward.Denom, communityParticipationReward)), senderModule)
		if err != nil {
			return reward, utils.LavaFormatError("sending community participation failed", err,
				utils.Attribute{Key: "community_participation_reward", Value: communityParticipationReward.String() + k.stakingKeeper.BondDenom(ctx)},
				utils.Attribute{Key: "community_participation", Value: communityParticipation.String()},
				utils.Attribute{Key: "reward", Value: reward.String()},
			)
		}
	}

	// update reward amount
	reward = reward.SubAmount(communityParticipationReward).SubAmount(validatorsParticipationReward)

	return reward, nil
}

// CalculateContributionPercentages calculates the providers' rewards participation to the validators and community pool
func (k Keeper) CalculateContributionPercentages(ctx sdk.Context, reward math.Int) (validatorsParticipation math.LegacyDec, communityParticipation math.LegacyDec, err error) {
	communityTax := k.distributionKeeper.GetParams(ctx).CommunityTax
	if communityTax.Equal(sdk.OneDec()) {
		return sdk.ZeroDec(), sdk.OneDec(), nil
	}

	// validators_participation = validators_participation_param / (1-community_tax)
	validatorsParticipationParam := k.GetParams(ctx).ValidatorsSubscriptionParticipation
	validatorsParticipation = validatorsParticipationParam.Quo(sdk.OneDec().Sub(communityTax))
	if validatorsParticipation.GT(sdk.OneDec()) {
		return sdk.ZeroDec(), sdk.ZeroDec(), utils.LavaFormatError("validators participation bigger than 100%", fmt.Errorf("validators participation calc failed"),
			utils.Attribute{Key: "validators_participation", Value: validatorsParticipation.String()},
			utils.Attribute{Key: "validators_subscription_participation_param", Value: validatorsParticipationParam.String()},
			utils.Attribute{Key: "community_tax", Value: communityTax.String()},
		)
	}

	// community_participation = (community_tax + validators_participation_param) - validators_participation
	communityParticipation = communityTax.Add(validatorsParticipationParam).Sub(validatorsParticipation)
	if communityParticipation.IsNegative() || communityParticipation.GT(sdk.OneDec()) {
		return sdk.ZeroDec(), sdk.ZeroDec(), utils.LavaFormatError("community participation is negative or bigger than 100%", fmt.Errorf("community participation calc failed"),
			utils.Attribute{Key: "community_participation", Value: communityParticipation.String()},
			utils.Attribute{Key: "validators_participation", Value: validatorsParticipation.String()},
			utils.Attribute{Key: "validators_subscription_participation_param", Value: validatorsParticipationParam.String()},
			utils.Attribute{Key: "community_tax", Value: communityTax.String()},
		)
	}

	// check the participation rewards are not more than 100%
	if validatorsParticipation.Add(communityParticipation).GT(sdk.OneDec()) {
		return sdk.ZeroDec(), sdk.ZeroDec(), utils.LavaFormatError("validators and community participation parts are bigger than 100%", fmt.Errorf("validators and community participation aborted"),
			utils.Attribute{Key: "community_participation", Value: communityParticipation.String()},
			utils.Attribute{Key: "validators_participation", Value: validatorsParticipation.String()},
		)
	}

	return validatorsParticipation, communityParticipation, nil
}

func (k Keeper) FundCommunityPoolFromModule(ctx sdk.Context, amount sdk.Coins, senderModule string) error {
	if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, senderModule, distributiontypes.ModuleName, amount); err != nil {
		return err
	}

	feePool := k.distributionKeeper.GetFeePool(ctx)
	feePool.CommunityPool = feePool.CommunityPool.Add(sdk.NewDecCoinsFromCoins(amount...)...)
	k.distributionKeeper.SetFeePool(ctx, feePool)

	return nil
}

// isEndOfMonth checks that we're close to next timer expiry by at least 24 hours
func (k Keeper) isEndOfMonth(ctx sdk.Context) bool {
	return ctx.BlockTime().UTC().Unix()+DAY_SECONDS > k.TimeToNextTimerExpiry(ctx)
}
