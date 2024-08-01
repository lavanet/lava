package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/subscription/types"
)

func (k Keeper) GetAllClusters(ctx sdk.Context) []string {
	clusters := []string{}
	plans := k.plansKeeper.GetAllPlanIndices(ctx)
	for _, plan := range plans {
		for _, usage := range types.SUB_USAGES {
			sub := types.Subscription{PlanIndex: plan, DurationTotal: usage}
			cluster := types.GetClusterKey(sub)
			clusters = append(clusters, cluster)
		}
	}

	return clusters
}
