package keeper

import (
	"strconv"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
	"github.com/lavanet/lava/x/subscription/types"
)

// GetTrackedCu gets the tracked CU counter (with/without QoS influence) and the trackedCu entry's block
func (k Keeper) GetTrackedCu(ctx sdk.Context, sub string, provider string, chainID string, block uint64) (cu uint64, found bool, key string) {
	cuTrackerKey := types.CuTrackerKey(sub, provider, chainID, block)
	var trackedCu types.TrackedCu
	found = k.cuTrackerFS.FindEntry(ctx, cuTrackerKey, block, &trackedCu)
	if !found {
		// entry not found -> this is the first, so not an error. return CU=0
		return 0, found, cuTrackerKey
	}
	return trackedCu.Cu, found, cuTrackerKey
}

// AddTrackedCu adds CU to the CU counters in relevant trackedCu entry
func (k Keeper) AddTrackedCu(ctx sdk.Context, sub string, provider string, chainID string, cuToAdd uint64, block uint64) error {
	cu, _, key := k.GetTrackedCu(ctx, sub, provider, chainID, block)

	// Note that the trackedCu entry usually has one version since we used
	// the subscription's block which is constant during a specific month
	// (updating an entry using append in the same block acts as ModifyEntry).
	// At most, there can be two trackedCu entries. Two entries occur
	// in the time period after a month has passed but before the payment
	// timer ended (in this time, a provider can still request payment for the previous month)
	err := k.cuTrackerFS.AppendEntry(ctx, key, block, &types.TrackedCu{Cu: cu + cuToAdd})
	if err != nil {
		return utils.LavaFormatError("cannot add tracked CU", err,
			utils.Attribute{Key: "tracked_cu_key", Value: key},
			utils.Attribute{Key: "sub_block", Value: strconv.FormatUint(block, 10)},
			utils.Attribute{Key: "current_cu", Value: strconv.FormatUint(cu, 10)},
			utils.Attribute{Key: "cu_to_be_added", Value: strconv.FormatUint(cuToAdd, 10)})
	}
	return nil
}

// GetAllSubTrackedCuIndices gets all the trackedCu entries that are related to a specific subscription
func (k Keeper) GetAllSubTrackedCuIndices(ctx sdk.Context, sub string, blockStr string) []string {
	if blockStr == "" && sub == "" {
		return k.cuTrackerFS.GetAllEntryIndicesWithPrefix(ctx, "")
	}
	return k.cuTrackerFS.GetAllEntryIndicesWithPrefix(ctx, sub+" "+blockStr)
}

// removeCuTracker removes a trackedCu entry
func (k Keeper) resetCuTracker(ctx sdk.Context, sub string, info trackedCuInfo) error {
	key := types.CuTrackerKey(sub, info.provider, info.chainID, info.block)
	return k.cuTrackerFS.DelEntry(ctx, key, uint64(ctx.BlockHeight()))
}

type trackedCuInfo struct {
	provider  string
	chainID   string
	trackedCu uint64
	block     uint64
}

func (k Keeper) GetSubTrackedCuInfo(ctx sdk.Context, sub string, subBlockStr string) (trackedCuList []trackedCuInfo, totalCuTracked uint64) {
	keys := k.GetAllSubTrackedCuIndices(ctx, sub, subBlockStr)

	for _, key := range keys {
		_, provider, chainID, blockStr := types.DecodeCuTrackerKey(key)
		block, err := strconv.ParseUint(blockStr, 10, 64)
		if err != nil {
			utils.LavaFormatError("cannot remove cu tracker", err,
				utils.Attribute{Key: "sub", Value: sub},
				utils.Attribute{Key: "provider", Value: provider},
				utils.Attribute{Key: "chain_id", Value: chainID},
				utils.Attribute{Key: "block_str", Value: blockStr},
			)
			continue
		}

		cu, found, _ := k.GetTrackedCu(ctx, sub, provider, chainID, block)
		if !found {
			utils.LavaFormatWarning("cannot remove cu tracker", legacyerrors.ErrKeyNotFound,
				utils.Attribute{Key: "sub", Value: sub},
				utils.Attribute{Key: "provider", Value: provider},
				utils.Attribute{Key: "chain_id", Value: chainID},
				utils.Attribute{Key: "block", Value: blockStr},
			)
			continue
		}
		trackedCuList = append(trackedCuList, trackedCuInfo{
			provider:  provider,
			trackedCu: cu,
			chainID:   chainID,
			block:     block,
		})
		totalCuTracked += cu
	}

	return trackedCuList, totalCuTracked
}

// remove only before the sub is deleted
func (k Keeper) RewardAndResetCuTracker(ctx sdk.Context, cuTrackerTimerKeyBytes []byte, cuTrackerTimerData []byte) {
	sub := string(cuTrackerTimerKeyBytes)
	blockStr := string(cuTrackerTimerData)
	_, err := strconv.ParseUint(blockStr, 10, 64)
	if err != nil {
		utils.LavaFormatError(types.ErrCuTrackerPayoutFailed.Error(), err,
			utils.Attribute{Key: "blockStr", Value: blockStr},
		)
		return
	}
	trackedCuList, totalCuTracked := k.GetSubTrackedCuInfo(ctx, sub, blockStr)

	var block uint64
	if len(trackedCuList) == 0 {
		utils.LavaFormatWarning("no tracked CU", types.ErrCuTrackerPayoutFailed,
			utils.Attribute{Key: "sub_consumer", Value: sub},
		)
		return
	}

	// note: there is an implicit assumption here that the subscription's
	// plan didn't change throughout the month. Currently there is no way
	// of altering the subscription's plan after being bought, but if there
	// might be in the future, this code should change
	block = trackedCuList[0].block
	plan, err := k.GetPlanFromSubscription(ctx, sub, block)
	if err != nil {
		utils.LavaFormatError("cannot find subscription's plan", types.ErrCuTrackerPayoutFailed,
			utils.Attribute{Key: "sub_consumer", Value: sub},
		)
		return
	}

	for _, trackedCuInfo := range trackedCuList {
		trackedCu := trackedCuInfo.trackedCu
		provider := trackedCuInfo.provider
		chainID := trackedCuInfo.chainID

		err := k.resetCuTracker(ctx, sub, trackedCuInfo)
		if err != nil {
			utils.LavaFormatError("removing/reseting tracked CU entry failed", err,
				utils.Attribute{Key: "provider", Value: provider},
				utils.Attribute{Key: "tracked_cu", Value: trackedCu},
				utils.Attribute{Key: "chain_id", Value: chainID},
				utils.Attribute{Key: "sub", Value: sub},
				utils.Attribute{Key: "block", Value: ctx.BlockHeight()},
			)
			return
		}

		// monthly reward = (tracked_CU / total_CU_used_in_sub_this_month) * plan_price
		// TODO: deal with the reward's remainder (uint division...)
		totalMonthlyReward := k.CalcTotalMonthlyReward(ctx, plan, trackedCu, totalCuTracked)

		// calculate the provider reward (smaller than totalMonthlyReward
		// because it's shared with delegators)
		providerAddr, err := sdk.AccAddressFromBech32(provider)
		if err != nil {
			utils.LavaFormatError("invalid provider address", err,
				utils.Attribute{Key: "provider", Value: provider},
			)
			return
		}

		// Note: if the reward function doesn't reward the provider
		// because he was unstaked, we only print an error and not returning
		providerReward, err := k.dualstakingKeeper.RewardProvidersAndDelegators(ctx, providerAddr, chainID, totalMonthlyReward, types.ModuleName, false)
		if err == epochstoragetypes.ErrProviderNotStaked || err == epochstoragetypes.ErrStakeStorageNotFound {
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
			return
		} else {
			utils.LogLavaEvent(ctx, k.Logger(ctx), types.MonthlyCuTrackerProviderRewardEventName, map[string]string{
				"provider":   provider,
				"sub":        sub,
				"plan":       plan.Index,
				"tracked_cu": strconv.FormatUint(trackedCu, 10),
				"plan_price": plan.Price.String(),
				"reward":     providerReward.String(),
				"block":      strconv.FormatInt(ctx.BlockHeight(), 10),
			}, "Provider got monthly reward successfully")
		}
	}
}

func (k Keeper) CalcTotalMonthlyReward(ctx sdk.Context, plan planstypes.Plan, trackedCu uint64, totalCuUsedBySub uint64) math.Int {
	// TODO: deal with the reward's remainder (uint division...)
	// monthly reward = (tracked_CU / total_CU_used_in_sub_this_month) * plan_price
	if totalCuUsedBySub == 0 {
		return math.ZeroInt()
	}
	totalMonthlyReward := plan.Price.Amount.MulRaw(int64(trackedCu)).QuoRaw(int64(totalCuUsedBySub))
	return totalMonthlyReward
}
