package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/lavanet/lava/utils"
	planstypes "github.com/lavanet/lava/x/plans/types"
)

func (k Keeper) EnforceClientCUsUsageInEpoch(ctx sdk.Context, relayCU, allowedCU, totalCUInEpochForUserProvider uint64, clientAddr sdk.AccAddress, chainID string, epoch uint64) (uint64, error) {
	project, err := k.GetProjectData(ctx, clientAddr, chainID, epoch)
	if err != nil {
		return 0, err
	}

	plan, err := k.subscriptionKeeper.GetPlanFromSubscription(ctx, project.GetSubscription(), epoch)
	if err != nil {
		return 0, err
	}

	planPolicy := plan.GetPlanPolicy()
	policies := []*planstypes.Policy{&planPolicy, project.AdminPolicy, project.SubscriptionPolicy}

	sub, found := k.subscriptionKeeper.GetSubscription(ctx, project.GetSubscription())
	if !found {
		return 0, utils.LavaFormatError("can't find subscription", fmt.Errorf("EnforceClientCUsUsageInEpoch_cant_find_subscription"), utils.Attribute{Key: "subscriptionKey", Value: project.GetSubscription()})
	}

	if sub.GetMonthCuLeft() == 0 {
		return 0, utils.LavaFormatError("total cu in epoch for consumer exceeded the amount of CU left in the subscription", fmt.Errorf("consumer CU limit exceeded for subscription"), []utils.Attribute{{Key: "subscriptionCuLeft", Value: sub.GetMonthCuLeft()}}...)
	}

	_, effectiveTotalCu := k.CalculateEffectiveAllowedCuPerEpochFromPolicies(policies, project.UsedCu, sub.MonthCuLeft)
	if !planstypes.VerifyTotalCuUsage(effectiveTotalCu, totalCUInEpochForUserProvider) {
		return effectiveTotalCu - project.UsedCu, nil
	}

	culimit := allowedCU * k.downtimeKeeper.GetDowntimeFactor(ctx, epoch)
	if totalCUInEpochForUserProvider > culimit {
		originalTotalCUInEpochForUserProvider := totalCUInEpochForUserProvider - relayCU
		return culimit - originalTotalCUInEpochForUserProvider, nil
	}

	return relayCU, nil
}
