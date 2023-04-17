package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	projectstypes "github.com/lavanet/lava/x/projects/types"
)

func (k Keeper) GetAllowedCUForBlock(ctx sdk.Context, blockHeight uint64, entry *epochstoragetypes.StakeEntry) (uint64, error) {
	var allowedCU uint64 = 0
	stakeToMaxCUMap, err := k.StakeToMaxCUList(ctx, blockHeight)
	if err != nil {
		return 0, err
	}

	for _, stakeToCU := range stakeToMaxCUMap.List {
		if entry.Stake.IsGTE(stakeToCU.StakeThreshold) {
			allowedCU = stakeToCU.MaxComputeUnits
		} else {
			break
		}
	}
	return allowedCU, nil
}

func (k Keeper) EnforceClientCUsUsageInEpoch(ctx sdk.Context, allowedCU uint64, totalCUInEpochForUserProvider uint64, clientAddr sdk.AccAddress, chainID string) error {
	project, _, err := k.GetProjectData(ctx, clientAddr, chainID, uint64(ctx.BlockHeight()))
	// if client is not legacy (works through a project), the CU verification is different
	if err == nil {
		plan, err := k.subscriptionKeeper.GetPlanFromSubscription(ctx, project.GetSubscription())
		if err != nil {
			return err
		}

		planPolicy := plan.GetPlanPolicy()
		policies := []*projectstypes.Policy{&planPolicy, project.AdminPolicy, project.SubscriptionPolicy}
		if !projectstypes.VerifyCuUsage(policies, project.GetUsedCu()) {
			return utils.LavaFormatError("total cu in epoch for consumer exceeded the allowed amount for the project", fmt.Errorf("consumer CU limit exceeded for project"), []utils.Attribute{{Key: "projectUsedCu", Value: project.GetUsedCu()}}...)
		}

		sub, found := k.subscriptionKeeper.GetSubscription(ctx, project.GetSubscription())
		if !found {
			return utils.LavaFormatError("can't find subscription", fmt.Errorf("EnforceClientCUsUsageInEpoch_cant_find_subscription"), utils.Attribute{Key: "subscriptionKey", Value: project.GetSubscription()})
		}
		if !projectstypes.VerifyCuUsage(policies, sub.GetMonthCuTotal()-sub.GetMonthCuLeft()) {
			return utils.LavaFormatError("total cu in epoch for consumer exceeded the allowed amount for the subscription", fmt.Errorf("consumer CU limit exceeded for subscription"), []utils.Attribute{{Key: "subscriptionUsedCu", Value: sub.GetMonthCuTotal() - sub.GetMonthCuLeft()}}...)
		}
		return nil
	}

	if totalCUInEpochForUserProvider > allowedCU {
		// if cu limit reached we return an error.
		return utils.LavaFormatError("total cu in epoch for consumer exceeded the allowed amount", fmt.Errorf("consumer CU limit exceeded"), []utils.Attribute{{Key: "totalCUInEpochForUserProvider", Value: totalCUInEpochForUserProvider}, {Key: "allowedCUProvider", Value: allowedCU}}...)
	}
	return nil
}

func (k Keeper) ClientMaxCUProviderForBlock(ctx sdk.Context, blockHeight uint64, clientEntry *epochstoragetypes.StakeEntry) (uint64, error) {
	allowedCU, err := k.GetAllowedCUForBlock(ctx, blockHeight, clientEntry)
	if err != nil {
		return 0, fmt.Errorf("user %s, MaxCU was not found for stake of: %d", clientEntry, clientEntry.Stake.Amount.Int64())
	}

	servicersToPairCount, err := k.ServicersToPairCount(ctx, blockHeight)
	if err != nil {
		return 0, err
	}

	allowedCU /= servicersToPairCount
	return allowedCU, nil
}
