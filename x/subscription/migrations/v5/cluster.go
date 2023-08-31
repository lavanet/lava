package v5

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

import "strconv"

const FREE_PLAN = "free" // gets its own const because it's treated differently

func GetSubUsageCriterion(sub Subscription) uint64 {
	switch {
	case sub.DurationTotal == 0:
		return 0
	case sub.DurationTotal > 6:
		return 7
	default:
		return 6
	}
}

// GetClusterKey returns the subscription's best-fit cluster
func GetClusterKey(sub Subscription) string {
	if sub.PlanIndex == FREE_PLAN {
		return FREE_PLAN
	}

	return sub.PlanIndex + "_" + strconv.FormatUint(GetSubUsageCriterion(sub), 10)
}
