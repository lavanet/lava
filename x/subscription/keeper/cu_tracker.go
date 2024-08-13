package keeper

import (
	"errors"
	"fmt"
	"strconv"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/v2/utils"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	rewardstypes "github.com/lavanet/lava/v2/x/rewards/types"
	"github.com/lavanet/lava/v2/x/subscription/types"
)

const LIMIT_TOKEN_PER_CU = 100

// GetTrackedCu gets the tracked CU counter (with QoS influence) and the trackedCu entry's block
func (k Keeper) GetTrackedCu(ctx sdk.Context, sub string, provider string, chainID string, subBlock uint64) (cu uint64, found bool, key string) {
	cuTrackerKey := types.CuTrackerKey(sub, provider, chainID)
	var trackedCu types.TrackedCu
	entryBlock, _, _, found := k.cuTrackerFS.FindEntryDetailed(ctx, cuTrackerKey, subBlock, &trackedCu)
	if !found || entryBlock != subBlock {
		// entry not found/deleted -> this is the first, so not an error. return CU=0
		return 0, false, cuTrackerKey
	}
	return trackedCu.Cu, found, cuTrackerKey
}

// AddTrackedCu adds CU to the CU counters in relevant trackedCu entry
// Also, it counts the IPRPC CU if the subscription is IPRPC eligible
func (k Keeper) AddTrackedCu(ctx sdk.Context, sub string, provider string, chainID string, cuToAdd uint64, block uint64) error {
	k.rewardsKeeper.AggregateCU(ctx, sub, provider, chainID, cuToAdd)

	cu, found, key := k.GetTrackedCu(ctx, sub, provider, chainID, block)

	// Note that the trackedCu entry usually has one version since we used
	// the subscription's block which is constant during a specific month
	// (updating an entry using append in the same block acts as ModifyEntry).
	// At most, there can be two trackedCu entries. Two entries occur
	// in the time period after a month has passed but before the payment
	// timer ended (in this time, a provider can still request payment for the previous month)
	if found {
		k.cuTrackerFS.ModifyEntry(ctx, key, block, &types.TrackedCu{Cu: cu + cuToAdd})
	} else {
		err := k.cuTrackerFS.AppendEntry(ctx, key, block, &types.TrackedCu{Cu: cuToAdd})
		if err != nil {
			return utils.LavaFormatError("cannot create new tracked CU entry", err,
				utils.Attribute{Key: "tracked_cu_key", Value: key},
				utils.Attribute{Key: "sub_block", Value: strconv.FormatUint(block, 10)},
				utils.Attribute{Key: "current_cu", Value: strconv.FormatUint(cu, 10)},
				utils.Attribute{Key: "cu_to_be_added", Value: strconv.FormatUint(cuToAdd, 10)})
		}
	}

	utils.LavaFormatDebug("adding tracked cu",
		utils.LogAttr("sub", sub),
		utils.LogAttr("provider", provider),
		utils.LogAttr("chain_id", chainID),
		utils.LogAttr("added_cu", cuToAdd),
		utils.LogAttr("block", block))

	return nil
}

// GetAllSubTrackedCuIndices gets all the trackedCu entries that are related to a specific subscription
func (k Keeper) GetAllSubTrackedCuIndices(ctx sdk.Context, sub string) []string {
	return k.cuTrackerFS.GetAllEntryIndicesWithPrefix(ctx, sub)
}

// removeCuTracker removes a trackedCu entry
func (k Keeper) resetCuTracker(ctx sdk.Context, sub string, info *types.TrackedCuInfo, subBlock uint64) error {
	key := types.CuTrackerKey(sub, info.Provider, info.ChainID)
	var trackedCu types.TrackedCu
	_, _, isLatest, _ := k.cuTrackerFS.FindEntryDetailed(ctx, key, subBlock, &trackedCu)
	if isLatest {
		return k.cuTrackerFS.DelEntry(ctx, key, uint64(ctx.BlockHeight()))
	}
	return nil
}

func (k Keeper) GetSubTrackedCuInfo(ctx sdk.Context, sub string, block uint64) (trackedCuList []*types.TrackedCuInfo, totalCuTracked uint64) {
	keys := k.GetAllSubTrackedCuIndices(ctx, sub)

	for _, key := range keys {
		_, provider, chainID := types.DecodeCuTrackerKey(key)
		cu, found, _ := k.GetTrackedCu(ctx, sub, provider, chainID, block)
		if !found {
			utils.LavaFormatWarning("cannot remove cu tracker", legacyerrors.ErrKeyNotFound,
				utils.Attribute{Key: "sub", Value: sub},
				utils.Attribute{Key: "provider", Value: provider},
				utils.Attribute{Key: "chain_id", Value: chainID},
				utils.Attribute{Key: "block", Value: strconv.FormatUint(block, 10)},
			)
			continue
		}
		trackedCuList = append(trackedCuList, &types.TrackedCuInfo{
			Provider:  provider,
			TrackedCu: cu,
			ChainID:   chainID,
			Block:     block,
		})
		totalCuTracked += cu
	}

	return trackedCuList, totalCuTracked
}

// remove only before the sub is deleted
func (k Keeper) RewardAndResetCuTracker(ctx sdk.Context, cuTrackerTimerKeyBytes []byte, cuTrackerTimerData []byte) {
	sub := string(cuTrackerTimerKeyBytes)
	var timerData types.CuTrackerTimerData
	err := k.cdc.Unmarshal(cuTrackerTimerData, &timerData)
	if err != nil {
		utils.LavaFormatError(types.ErrCuTrackerPayoutFailed.Error(), fmt.Errorf("invalid data from cu tracker timer"),
			utils.Attribute{Key: "timer_data", Value: timerData.String()},
			utils.Attribute{Key: "consumer", Value: sub},
		)
		return
	}
	if !timerData.Validate() {
		return
	}
	trackedCuList, totalCuTracked := k.GetSubTrackedCuInfo(ctx, sub, timerData.Block)

	if len(trackedCuList) == 0 || totalCuTracked == 0 {
		// no tracked CU for this sub, return the credit to the sub
		k.returnCreditToSub(ctx, sub, timerData.Credit.Amount)
		return
	}

	// Note: We take the subscription from the FixationStore, based on the given block.
	// So, even if the plan changed during the month, we still take the original plan, based on the given block.
	block := trackedCuList[0].Block

	totalTokenAmount := timerData.Credit.Amount
	if totalTokenAmount.Quo(sdk.NewIntFromUint64(totalCuTracked)).GT(sdk.NewIntFromUint64(LIMIT_TOKEN_PER_CU)) {
		totalTokenAmount = sdk.NewIntFromUint64(LIMIT_TOKEN_PER_CU * totalCuTracked)
	}

	// get the adjustment factor, and delete the entries
	adjustments := k.GetConsumerAdjustments(ctx, sub)
	adjustmentFactorForProvider := k.GetAdjustmentFactorProvider(ctx, adjustments)
	k.RemoveConsumerAdjustments(ctx, sub)

	details := map[string]string{}

	totalTokenRewarded := sdk.ZeroInt()
	for _, trackedCuInfo := range trackedCuList {
		trackedCu := trackedCuInfo.TrackedCu
		provider := trackedCuInfo.Provider
		chainID := trackedCuInfo.ChainID

		err = k.resetCuTracker(ctx, sub, trackedCuInfo, block)
		if err != nil {
			utils.LavaFormatError("removing/reseting tracked CU entry failed", err,
				utils.Attribute{Key: "provider", Value: provider},
				utils.Attribute{Key: "tracked_cu", Value: trackedCu},
				utils.Attribute{Key: "chain_id", Value: chainID},
				utils.Attribute{Key: "sub", Value: sub},
				utils.Attribute{Key: "block", Value: ctx.BlockHeight()},
			)
			continue
		}

		// provider monthly reward = (tracked_CU / total_CU_used_in_sub_this_month) * totalTokenAmount
		providerAdjustment, ok := adjustmentFactorForProvider[provider]
		if !ok {
			maxRewardBoost := k.rewardsKeeper.MaxRewardBoost(ctx)
			if maxRewardBoost == 0 {
				utils.LavaFormatWarning("maxRewardBoost is zero", fmt.Errorf("critical: Attempt to divide by zero"),
					utils.LogAttr("maxRewardBoost", maxRewardBoost),
				)
				return
			}
			providerAdjustment = sdk.OneDec().Quo(sdk.NewDecFromInt(sdk.NewIntFromUint64(maxRewardBoost)))
		}

		// calculate the provider reward (smaller than totalMonthlyReward
		// because it's shared with delegators)
		totalMonthlyRewardAmount := k.CalcTotalMonthlyReward(ctx, totalTokenAmount, trackedCu, totalCuTracked)
		creditToSub := sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), totalMonthlyRewardAmount)
		totalTokenRewarded = totalTokenRewarded.Add(totalMonthlyRewardAmount)

		k.rewardsKeeper.AggregateRewards(ctx, provider, chainID, providerAdjustment, totalMonthlyRewardAmount)

		// Transfer some of the total monthly reward to validators contribution and community pool
		creditToSub, err = k.rewardsKeeper.ContributeToValidatorsAndCommunityPool(ctx, creditToSub, types.ModuleName)
		if err != nil {
			utils.LavaFormatError("could not contribute to validators and community pool", err,
				utils.Attribute{Key: "total_monthly_reward", Value: creditToSub.String()})
		}

		// Note: if the reward function doesn't reward the provider
		// because he was unstaked, we only print an error and not returning

		_, _, err := k.dualstakingKeeper.RewardProvidersAndDelegators(ctx, provider, chainID, sdk.NewCoins(creditToSub), types.ModuleName, false, false, false)
		if errors.Is(err, epochstoragetypes.ErrProviderNotStaked) || errors.Is(err, epochstoragetypes.ErrStakeStorageNotFound) {
			utils.LavaFormatWarning("sending provider reward with delegations failed", err,
				utils.Attribute{Key: "provider", Value: provider},
				utils.Attribute{Key: "chain_id", Value: chainID},
				utils.Attribute{Key: "block", Value: strconv.FormatInt(ctx.BlockHeight(), 10)},
			)
		} else if err != nil {
			utils.LavaFormatError("sending provider reward with delegations failed", err,
				utils.Attribute{Key: "provider", Value: provider},
				utils.Attribute{Key: "tracked_cu", Value: trackedCu},
				utils.Attribute{Key: "chain_id", Value: chainID},
				utils.Attribute{Key: "sub", Value: sub},
				utils.Attribute{Key: "sub_total_used_cu", Value: totalCuTracked},
				utils.Attribute{Key: "block", Value: ctx.BlockHeight()},
			)
		} else {
			details[provider+" "+chainID] = fmt.Sprintf("cu: %d reward: %s", trackedCu, creditToSub.String())
		}
	}

	details["subscription"] = sub
	details["total_cu"] = strconv.FormatUint(totalCuTracked, 10)
	details["total_rewards"] = totalTokenRewarded.String()
	details["block"] = strconv.FormatInt(ctx.BlockHeight(), 10)

	utils.LogLavaEvent(ctx, k.Logger(ctx), types.SubscriptionPayoutEventName, details, "subscription monthly payout and reset")
}

func (k Keeper) CalcTotalMonthlyReward(ctx sdk.Context, totalAmount math.Int, trackedCu uint64, totalCuUsedBySub uint64) math.Int {
	if totalCuUsedBySub == 0 {
		return math.ZeroInt()
	}

	totalMonthlyReward := totalAmount.Mul(sdk.NewIntFromUint64(trackedCu)).Quo(sdk.NewIntFromUint64(totalCuUsedBySub))
	return totalMonthlyReward
}

func (k Keeper) returnCreditToSub(ctx sdk.Context, sub string, credit math.Int) sdk.Coin {
	var latestSub types.Subscription
	latestEntryBlock, _, _, found := k.subsFS.FindEntryDetailed(ctx, sub, uint64(ctx.BlockHeight()), &latestSub)
	if found {
		latestSub.Credit = latestSub.Credit.AddAmount(credit)
		k.subsFS.ModifyEntry(ctx, latestSub.Consumer, latestEntryBlock, &latestSub)
		return latestSub.Credit
	} else {
		// sub expired (no need to update credit), send rewards remainder to the validators
		pool := rewardstypes.ValidatorsRewardsDistributionPoolName
		err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, string(pool), sdk.NewCoins(sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), credit)))
		if err != nil {
			utils.LavaFormatError("failed sending remainder of rewards to the community pool", err,
				utils.Attribute{Key: "rewards_remainder", Value: credit.String()},
			)
		}
	}

	return sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), math.ZeroInt())
}

// wrapper function for calculating the validators and community participation fees
func (k Keeper) CalculateParticipationFees(ctx sdk.Context, reward sdk.Coin) (sdk.Coins, sdk.Coins, error) {
	return k.rewardsKeeper.CalculateValidatorsAndCommunityParticipationRewards(ctx, reward)
}
