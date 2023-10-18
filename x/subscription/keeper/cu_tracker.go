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

func (k Keeper) GetTrackedCu(ctx sdk.Context, sub string, provider string, chainID string, subBlock uint64) (cuWithQos uint64, cuWithoutQos uint64, found bool, key string) {
	cuTrackerKey := types.CuTrackerKey(sub, provider, chainID)
	var trackedCu types.TrackedCu
	found = k.cuTrackerFS.FindEntry(ctx, cuTrackerKey, subBlock, &trackedCu)
	return trackedCu.GetCuWithQos(), trackedCu.GetCuWithoutQos(), found, cuTrackerKey
}

func (k Keeper) AddTrackedCu(ctx sdk.Context, sub string, provider string, chainID string, cu uint64, cuBeforeQos uint64, subBlock uint64) error {
	if sub == "" || provider == "" || subBlock > uint64(ctx.BlockHeight()) {
		return utils.LavaFormatError("cannot add tracked CU",
			fmt.Errorf("sub/provider cannot be empty. subBlock cannot be larger than current block"),
			utils.Attribute{Key: "sub", Value: sub},
			utils.Attribute{Key: "provider", Value: provider},
			utils.Attribute{Key: "sub_block", Value: strconv.FormatUint(subBlock, 10)},
		)
	}
	cuWithQos, cuWithoutQos, _, key := k.GetTrackedCu(ctx, sub, provider, chainID, subBlock)
	err := k.cuTrackerFS.AppendEntry(ctx, key, subBlock, &types.TrackedCu{CuWithQos: cuWithQos + cu, CuWithoutQos: cuWithoutQos + cuBeforeQos})
	if err != nil {
		return utils.LavaFormatError("cannot add tracked CU", err,
			utils.Attribute{Key: "tracked_cu_key", Value: key},
			utils.Attribute{Key: "sub_block", Value: strconv.FormatUint(subBlock, 10)},
			utils.Attribute{Key: "current_cu", Value: cuWithQos},
			utils.Attribute{Key: "cu_to_be_added", Value: strconv.FormatUint(cu, 10)})
	}
	return nil
}

func (k Keeper) GetAllSubTrackedCuIndices(ctx sdk.Context, sub string) []string {
	return k.cuTrackerFS.GetAllEntryIndicesWithPrefix(ctx, sub)
}

func (k Keeper) removeCuTracker(ctx sdk.Context, sub string, provider string, chainID string) error {
	key := types.CuTrackerKey(sub, provider, chainID)
	return k.cuTrackerFS.DelEntry(ctx, key, uint64(ctx.BlockHeight()))
}

type trackedCuInfo struct {
	provider  string
	chainID   string
	trackedCu uint64
}

func (k Keeper) getSubTrackedCuInfo(ctx sdk.Context, sub string, subBlock uint64) (trackedCuList []trackedCuInfo, totalCuUsedBySub uint64) {
	keys := k.GetAllSubTrackedCuIndices(ctx, sub)

	for _, key := range keys {
		_, provider, chainID := types.DecodeCuTrackerKey(key)
		cuWithQos, cuWithoutQos, found, _ := k.GetTrackedCu(ctx, sub, provider, chainID, subBlock)
		if !found {
			utils.LavaFormatWarning("cannot remove cu tracker", legacyerrors.ErrKeyNotFound,
				utils.Attribute{Key: "sub", Value: sub},
				utils.Attribute{Key: "provider", Value: provider},
				utils.Attribute{Key: "block", Value: strconv.FormatUint(subBlock, 10)},
			)
		}
		trackedCuList = append(trackedCuList, trackedCuInfo{
			provider:  provider,
			trackedCu: cuWithQos,
			chainID:   chainID,
		})
		totalCuUsedBySub += cuWithoutQos
	}

	return trackedCuList, totalCuUsedBySub
}

// remove only before the sub is deleted
func (k Keeper) RewardAndResetCuTracker(ctx sdk.Context, cuTrackerTimerKeyBytes []byte) {
	cuTrackerTimerKey := string(cuTrackerTimerKeyBytes)
	sub, subBlock := types.DecodeCuTrackerTimerKey(cuTrackerTimerKey)
	if sub == "" {
		utils.LavaFormatError("invalid block", types.ErrCuTrackerPayoutFailed,
			utils.Attribute{Key: "sub", Value: sub},
			utils.Attribute{Key: "block", Value: subBlock},
		)
		return
	}

	trackedCuList, totalCuUsedBySub := k.getSubTrackedCuInfo(ctx, sub, subBlock)

	plan, err := k.GetPlanFromSubscription(ctx, sub, subBlock)
	if err != nil {
		utils.LavaFormatError("cannot find subscription's plan", types.ErrCuTrackerPayoutFailed,
			utils.Attribute{Key: "sub_consumer", Value: sub},
		)
		return
	}

	for _, trackedCuInfo := range trackedCuList {
		// TODO: deal with the reward's remainder (uint division...)
		if trackedCuInfo.trackedCu > totalCuUsedBySub {
			utils.LavaFormatError("tracked CU of provider is larger than total of CU used by subscription", types.ErrCuTrackerPayoutFailed,
				utils.Attribute{Key: "provider", Value: trackedCuInfo.provider},
				utils.Attribute{Key: "tracked_cu", Value: trackedCuInfo.trackedCu},
				utils.Attribute{Key: "chain_id", Value: trackedCuInfo.chainID},
				utils.Attribute{Key: "sub", Value: sub},
				utils.Attribute{Key: "sub_total_used_cu", Value: totalCuUsedBySub},
				utils.Attribute{Key: "block", Value: strconv.FormatUint(subBlock, 10)},
			)
			return
		}

		// monthly reward = (tracked_CU / total_CU_used_in_sub_this_month) * plan_price
		totalMonthlyReward := plan.Price.Amount.MulRaw(int64(trackedCuInfo.trackedCu)).QuoRaw(int64(totalCuUsedBySub))

		// calculate the provider reward (smaller than totalMonthlyReward because it's shared with delegators)
		providerAddr, err := sdk.AccAddressFromBech32(trackedCuInfo.provider)
		if err != nil {
			utils.LavaFormatError("invalid provider address", err,
				utils.Attribute{Key: "provider", Value: trackedCuInfo.provider},
			)
			return
		}

		// Note: if the reward function doesn't reward the provider because he was unstaked, we only print an error and not returning
		providerReward, err := k.dualstakingKeeper.RewardProvidersAndDelegators(ctx, providerAddr, trackedCuInfo.chainID, uint64(ctx.BlockHeight()), totalMonthlyReward, types.ModuleName)
		if err == dualstakingtypes.ErrProviderNotStaked {
			utils.LavaFormatWarning("sending provider reward with delegations failed", err,
				utils.Attribute{Key: "provider", Value: trackedCuInfo.provider},
				utils.Attribute{Key: "current_block", Value: ctx.BlockHeight()},
			)
		} else if err != nil {
			utils.LavaFormatError("sending provider reward with delegations failed", err,
				utils.Attribute{Key: "provider", Value: trackedCuInfo.provider},
				utils.Attribute{Key: "tracked_cu", Value: trackedCuInfo.trackedCu},
				utils.Attribute{Key: "chain_id", Value: trackedCuInfo.chainID},
				utils.Attribute{Key: "sub", Value: sub},
				utils.Attribute{Key: "sub_total_used_cu", Value: totalCuUsedBySub},
				utils.Attribute{Key: "sub_block", Value: strconv.FormatUint(subBlock, 10)},
				utils.Attribute{Key: "relay_block", Value: strconv.FormatUint(subBlock, 10)},
			)
			return
		} else {
			utils.LogLavaEvent(ctx, k.Logger(ctx), types.MonthlyCuTrackerProviderRewardEventName, map[string]string{
				"provider":   trackedCuInfo.provider,
				"sub":        sub,
				"plan":       plan.Index,
				"tracked_cu": strconv.FormatUint(trackedCuInfo.trackedCu, 10),
				"plan_price": plan.Price.String(),
				"reward":     providerReward.String(),
			}, "Provider got monthly reward successfully")
		}

		err = k.removeCuTracker(ctx, sub, trackedCuInfo.provider, trackedCuInfo.chainID)
		// TODO: what to do if err != nil? the money has been paid but the tracked CU entry was not removed
		if err != nil {
			utils.LavaFormatError("removing tracked CU entry failed", err,
				utils.Attribute{Key: "provider", Value: trackedCuInfo.provider},
				utils.Attribute{Key: "tracked_cu", Value: trackedCuInfo.trackedCu},
				utils.Attribute{Key: "chain_id", Value: trackedCuInfo.chainID},
				utils.Attribute{Key: "sub", Value: sub},
			)
			return
		}
	}
}
