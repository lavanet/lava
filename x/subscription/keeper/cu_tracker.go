package keeper

import (
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/utils"
	dualstakingtypes "github.com/lavanet/lava/x/dualstaking/types"
	"github.com/lavanet/lava/x/subscription/types"
)

// GetTrackedCu gets the tracked CU counter (with/without QoS influence) and the trackedCu entry's block
func (k Keeper) GetTrackedCu(ctx sdk.Context, sub string, provider string, chainID string) (cuWithQos uint64, cuWithoutQos uint64, entryBlock uint64, found bool, key string) {
	cuTrackerKey := types.CuTrackerKey(sub, provider, chainID)
	var trackedCu types.TrackedCu
	entryBlock, found = k.cuTrackerFS.FindEntry2(ctx, cuTrackerKey, uint64(ctx.BlockHeight()), &trackedCu)
	if !found {
		// entry not found -> this is the first, so not an error. return the current block and CUs=0
		return 0, 0, uint64(ctx.BlockHeight()), found, cuTrackerKey
	}
	return trackedCu.CuWithQos, trackedCu.CuWithoutQos, entryBlock, found, cuTrackerKey
}

// AddTrackedCu adds CU to the CU counters in relevant trackedCu entry
// Note that the trackedCu entry always has one version (updating it using append in the same block acts as ModifyEntry)
func (k Keeper) AddTrackedCu(ctx sdk.Context, sub string, provider string, chainID string, cu uint64, cuBeforeQos uint64) error {
	if sub == "" || provider == "" {
		return utils.LavaFormatError("cannot add tracked CU", fmt.Errorf("sub/provider cannot be empty"),
			utils.Attribute{Key: "sub", Value: sub},
			utils.Attribute{Key: "provider", Value: provider},
		)
	}
	cuWithQos, cuWithoutQos, entryBlock, _, key := k.GetTrackedCu(ctx, sub, provider, chainID)
	err := k.cuTrackerFS.AppendEntry(ctx, key, entryBlock, &types.TrackedCu{CuWithQos: cuWithQos + cu, CuWithoutQos: cuWithoutQos + cuBeforeQos})
	if err != nil {
		return utils.LavaFormatError("cannot add tracked CU", err,
			utils.Attribute{Key: "tracked_cu_key", Value: key},
			utils.Attribute{Key: "sub_block", Value: strconv.FormatUint(entryBlock, 10)},
			utils.Attribute{Key: "current_cu", Value: cuWithQos},
			utils.Attribute{Key: "cu_to_be_added", Value: strconv.FormatUint(cu, 10)})
	}
	return nil
}

// GetAllSubTrackedCuIndices gets all the trackedCu entries that are related to a specific subscription
func (k Keeper) GetAllSubTrackedCuIndices(ctx sdk.Context, sub string) []string {
	return k.cuTrackerFS.GetAllEntryIndicesWithPrefix(ctx, sub)
}

// removeCuTracker removes a trackedCu entry
func (k Keeper) removeCuTracker(ctx sdk.Context, sub string, provider string, chainID string) error {
	key := types.CuTrackerKey(sub, provider, chainID)
	return k.cuTrackerFS.DelEntry(ctx, key, uint64(ctx.BlockHeight()))
}

type trackedCuInfo struct {
	provider  string
	chainID   string
	trackedCu uint64
}

func (k Keeper) getSubTrackedCuInfo(ctx sdk.Context, sub string) (trackedCuList []trackedCuInfo, totalCuUsedBySub uint64, entryBlock uint64) {
	keys := k.GetAllSubTrackedCuIndices(ctx, sub)

	for _, key := range keys {
		_, provider, chainID := types.DecodeCuTrackerKey(key)
		cuWithQos, cuWithoutQos, entryBlock, found, _ := k.GetTrackedCu(ctx, sub, provider, chainID)
		if !found {
			utils.LavaFormatWarning("cannot remove cu tracker", legacyerrors.ErrKeyNotFound,
				utils.Attribute{Key: "sub", Value: sub},
				utils.Attribute{Key: "provider", Value: provider},
				utils.Attribute{Key: "block", Value: strconv.FormatUint(entryBlock, 10)},
			)
		}
		trackedCuList = append(trackedCuList, trackedCuInfo{
			provider:  provider,
			trackedCu: cuWithQos,
			chainID:   chainID,
		})
		totalCuUsedBySub += cuWithoutQos
	}

	return trackedCuList, totalCuUsedBySub, entryBlock
}

// remove only before the sub is deleted
func (k Keeper) RewardAndResetCuTracker(ctx sdk.Context, cuTrackerTimerKeyBytes []byte) {
	sub := string(cuTrackerTimerKeyBytes)
	trackedCuList, totalCuUsedBySub, entryBlock := k.getSubTrackedCuInfo(ctx, sub)
	if entryBlock == 0 {
		utils.LavaFormatError("cannot find trackedCuEntry", types.ErrCuTrackerPayoutFailed,
			utils.Attribute{Key: "sub", Value: sub},
		)
		return
	}

	plan, err := k.GetPlanFromSubscription(ctx, sub, entryBlock)
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

		err = k.removeCuTracker(ctx, sub, provider, chainID)
		if err != nil {
			utils.LavaFormatError("removing tracked CU entry failed", err,
				utils.Attribute{Key: "provider", Value: provider},
				utils.Attribute{Key: "tracked_cu", Value: trackedCu},
				utils.Attribute{Key: "chain_id", Value: chainID},
				utils.Attribute{Key: "sub", Value: sub},
				utils.Attribute{Key: "block", Value: ctx.BlockHeight()},
			)
			return
		}

		// TODO: deal with the reward's remainder (uint division...)
		if trackedCu > totalCuUsedBySub {
			utils.LavaFormatError("tracked CU of provider is larger than total of CU used by subscription", types.ErrCuTrackerPayoutFailed,
				utils.Attribute{Key: "provider", Value: provider},
				utils.Attribute{Key: "tracked_cu", Value: trackedCu},
				utils.Attribute{Key: "chain_id", Value: chainID},
				utils.Attribute{Key: "sub", Value: sub},
				utils.Attribute{Key: "sub_total_used_cu", Value: totalCuUsedBySub},
				utils.Attribute{Key: "block", Value: strconv.FormatUint(uint64(ctx.BlockHeight()), 10)},
			)
			return
		}

		// monthly reward = (tracked_CU / total_CU_used_in_sub_this_month) * plan_price
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
		if err == dualstakingtypes.ErrProviderNotStaked {
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
