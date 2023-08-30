package keeper

import (
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/subscription/types"
)

// QoS clustering divides the QoS monitoring into a discrete set of clusters
// such that QoS is maintained separately for each Provider x Cluster.
//
// The clusters are determined based on certain subscription owner properties,
// such as past or recent activity (e.g. aggregate subscription periods), the
// current plan used, etc. Each consumer (subscription owner or project developer)
// QoS report about some provider will be considered only in the cluster matching
// that consumer’s properties. During pairing selection for a particular consumer,
// the QoS data for the pairing calculation will be taken from the cluster matching
// that consumer’s properties.
// Cluster assignment is updated when a subscription renews (every month).

// To add a new cluster criterion, update the Cluster struct, create an array with
// the criterion values (like PLAN_CRITERION) and add it to ConstructAllClusters()
//
// All clusters:
// 	1. For each plan (except "free") a cluster for each subUsage
//  2. "free" cluster (without regarding subUsage)

// GetCluster returns the subscription's best-fit cluster
func (k Keeper) GetCluster(ctx sdk.Context, sub types.Subscription) string {
	if sub.PlanIndex == types.FREE_PLAN {
		return types.FREE_PLAN
	}

	plans := k.plansKeeper.GetAllPlanIndices(ctx)
	for _, plan := range plans {
		for _, subUsage := range types.GetSubUsageCriterion() {
			if sub.PlanIndex == plan && sub.DurationTotal <= subUsage {
				return plan + "_" + strconv.FormatUint(subUsage, 10)
			}
		}
	}

	return ""
}
