package keeper

import (
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

// constructs all the possible cluster keys
func (k Keeper) ConstructAllClusters(ctx sdk.Context) []types.Cluster {
	var clusters []types.Cluster
	plans := k.plansKeeper.GetAllPlanIndices(ctx)
	for _, plan := range plans {
		for _, subUsage := range types.GetSubUsageCriterion() {
			if plan == types.FREE_PLAN {
				// all free plan users use a single cluster
				clusters = append(clusters, types.NewCluster(plan, 0))
				break
			}
			clusters = append(clusters, types.NewCluster(plan, subUsage))
		}
	}

	return clusters
}

// GetCluster returns the subscription's best-fit cluster
func (k Keeper) GetCluster(sub types.Subscription, clusters []types.Cluster) string {
	for _, cluster := range clusters {
		if sub.PlanIndex == cluster.Plan && (sub.PlanIndex == types.FREE_PLAN || sub.DurationTotal <= cluster.SubUsage) {
			return cluster.String()
		}
	}
	return ""
}
