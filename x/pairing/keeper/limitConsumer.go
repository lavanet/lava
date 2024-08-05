package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/lavanet/lava/v2/utils"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
)

func (k Keeper) EnforceClientCUsUsageInEpoch(ctx sdk.Context, relayCU, epochAllowedCU, totalCUInEpochForUserProvider uint64, clientAddr sdk.AccAddress, chainID string, epoch uint64) (uint64, error) {
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

	_, effectivePolicyTotalCu := k.CalculateEffectiveAllowedCuPerEpochFromPolicies(policies, project.UsedCu, sub.MonthCuLeft)
	if !planstypes.VerifyTotalCuUsage(effectivePolicyTotalCu, totalCUInEpochForUserProvider) {
		return effectivePolicyTotalCu - project.UsedCu, nil
	}

	epochCuLimit := epochAllowedCU * k.downtimeKeeper.GetDowntimeFactor(ctx, epoch)
	// Check if the total CUs used in the epoch is larger than the CU left for this epoch
	if totalCUInEpochForUserProvider > epochCuLimit {
		// Return remaining CU allowed for this epoch
		originalTotalCUInEpochForUserProvider := totalCUInEpochForUserProvider - relayCU
		if originalTotalCUInEpochForUserProvider <= epochCuLimit {
			// Remaining CU allowed for this epoch is the epoch limit minus the original total used
			return epochCuLimit - originalTotalCUInEpochForUserProvider, nil
		}

		utils.LavaFormatInfo("Client exceeded epoch CU limit",
			utils.LogAttr("epochCuLimit", epochCuLimit),
			utils.LogAttr("totalCUInEpochForUserProvider", totalCUInEpochForUserProvider),
			utils.LogAttr("relayCU", relayCU),
			utils.LogAttr("clientAddr", clientAddr.String()),
			utils.LogAttr("chainID", chainID),
			utils.LogAttr("epoch", epoch),
		)

		// There's no CU left for this epoch, so relay usage is 0
		return 0, nil
	}

	return relayCU, nil
}
