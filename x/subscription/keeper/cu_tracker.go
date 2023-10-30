package keeper

import (
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/subscription/types"
)

// GetTrackedCu gets the tracked CU counter (with/without QoS influence) and the trackedCu entry's block
func (k Keeper) GetTrackedCu(ctx sdk.Context, sub string, provider string, chainID string) (cu uint64, entryBlock uint64, found bool, key string) {
	cuTrackerKey := types.CuTrackerKey(sub, provider, chainID)
	var trackedCu types.TrackedCu
	entryBlock, found = k.cuTrackerFS.FindEntry2(ctx, cuTrackerKey, uint64(ctx.BlockHeight()), &trackedCu)
	if !found {
		// entry not found -> this is the first, so not an error. return the current block and CUs=0
		return 0, uint64(ctx.BlockHeight()), found, cuTrackerKey
	}
	return trackedCu.Cu, entryBlock, found, cuTrackerKey
}

// AddTrackedCu adds CU to the CU counters in relevant trackedCu entry
// Note that the trackedCu entry always has one version (updating it using append in the same block acts as ModifyEntry)
func (k Keeper) AddTrackedCu(ctx sdk.Context, sub string, provider string, chainID string, cuToAdd uint64) error {
	cu, entryBlock, _, key := k.GetTrackedCu(ctx, sub, provider, chainID)
	err := k.cuTrackerFS.AppendEntry(ctx, key, entryBlock, &types.TrackedCu{Cu: cu + cuToAdd})
	if err != nil {
		return utils.LavaFormatError("cannot add tracked CU", err,
			utils.Attribute{Key: "tracked_cu_key", Value: key},
			utils.Attribute{Key: "sub_block", Value: strconv.FormatUint(entryBlock, 10)},
			utils.Attribute{Key: "current_cu", Value: strconv.FormatUint(cu, 10)},
			utils.Attribute{Key: "cu_to_be_added", Value: strconv.FormatUint(cuToAdd, 10)})
	}
	return nil
}

// GetAllSubTrackedCuIndices gets all the trackedCu entries that are related to a specific subscription
func (k Keeper) GetAllSubTrackedCuIndices(ctx sdk.Context, sub string) []string {
	return k.cuTrackerFS.GetAllEntryIndicesWithPrefix(ctx, sub)
}

// removeCuTracker removes a trackedCu entry
func (k Keeper) resetCuTracker(ctx sdk.Context, sub string, info trackedCuInfo, shouldRemove bool) error {
	key := types.CuTrackerKey(sub, info.provider, info.chainID)
	if shouldRemove {
		return k.cuTrackerFS.DelEntry(ctx, key, uint64(ctx.BlockHeight()))
	}

	var trackedCu types.TrackedCu
	found := k.cuTrackerFS.FindEntry(ctx, key, info.block, &trackedCu)
	if !found {
		return utils.LavaFormatError("cannot reset CU tracker", fmt.Errorf("trackedCU entry is not found"),
			utils.Attribute{Key: "sub", Value: sub},
			utils.Attribute{Key: "provider", Value: info.provider},
			utils.Attribute{Key: "chainID", Value: info.chainID},
			utils.Attribute{Key: "block", Value: strconv.FormatUint(info.block, 10)},
		)
	}

	trackedCu.Cu = 0
	k.cuTrackerFS.ModifyEntry(ctx, key, info.block, &trackedCu)
	return nil
}

type trackedCuInfo struct {
	provider  string
	chainID   string
	trackedCu uint64
	block     uint64
}

func (k Keeper) getSubTrackedCuInfo(ctx sdk.Context, sub string) (trackedCuList []trackedCuInfo, totalCuUsedBySub uint64) {
	keys := k.GetAllSubTrackedCuIndices(ctx, sub)

	for _, key := range keys {
		_, provider, chainID := types.DecodeCuTrackerKey(key)
		cu, entryBlock, found, _ := k.GetTrackedCu(ctx, sub, provider, chainID)
		if !found {
			utils.LavaFormatWarning("cannot remove cu tracker", legacyerrors.ErrKeyNotFound,
				utils.Attribute{Key: "sub", Value: sub},
				utils.Attribute{Key: "provider", Value: provider},
				utils.Attribute{Key: "block", Value: strconv.FormatUint(entryBlock, 10)},
			)
		}
		trackedCuList = append(trackedCuList, trackedCuInfo{
			provider:  provider,
			trackedCu: cu,
			chainID:   chainID,
			block:     entryBlock,
		})
		totalCuUsedBySub += cu
	}

	return trackedCuList, totalCuUsedBySub
}

// remove only before the sub is deleted
func (k Keeper) RewardAndResetCuTracker(ctx sdk.Context, cuTrackerTimerKeyBytes []byte, cuTrackerTimerData []byte) {
	sub := string(cuTrackerTimerKeyBytes)
	shouldRemove := len(cuTrackerTimerData) != 0
	trackedCuList, totalCuUsedBySub := k.getSubTrackedCuInfo(ctx, sub)

	var block uint64
	if len(trackedCuList) == 0 {
		utils.LavaFormatWarning("no tracked CU", types.ErrCuTrackerPayoutFailed,
			utils.Attribute{Key: "sub_consumer", Value: sub},
		)
		return
	}

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

		err := k.resetCuTracker(ctx, sub, trackedCuInfo, shouldRemove)
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
		totalMonthlyReward := plan.Price.Amount.MulRaw(int64(trackedCu)).QuoRaw(int64(totalCuUsedBySub))

		// calculate the provider reward (smaller than totalMonthlyReward because it's shared with delegators)
		providerAddr, err := sdk.AccAddressFromBech32(provider)
		if err != nil {
			utils.LavaFormatError("invalid provider address", err,
				utils.Attribute{Key: "provider", Value: provider},
			)
			return
		}

		// Note: if the reward function doesn't reward the provider because he was unstaked, we only print an error and not returning
		providerReward, err := k.dualstakingKeeper.RewardProvidersAndDelegators(ctx, providerAddr, chainID, totalMonthlyReward, types.ModuleName)
		if err == epochstoragetypes.ErrProviderNotStaked || err == epochstoragetypes.ErrStakeStorageNotFound {
			utils.LavaFormatWarning("sending provider reward with delegations failed", err,
				utils.Attribute{Key: "provider", Value: provider},
				utils.Attribute{Key: "block", Value: ctx.BlockHeight()},
			)
		} else if err != nil {
			utils.LavaFormatError("sending provider reward with delegations failed", err,
				utils.Attribute{Key: "provider", Value: provider},
				utils.Attribute{Key: "tracked_cu", Value: trackedCu},
				utils.Attribute{Key: "chain_id", Value: chainID},
				utils.Attribute{Key: "sub", Value: sub},
				utils.Attribute{Key: "sub_total_used_cu", Value: totalCuUsedBySub},
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
