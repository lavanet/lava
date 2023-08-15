package types

import "strconv"

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
// the criterion values (like PLAN_CRITERION) and add it to constructAllClusters()

type Cluster struct {
	plan     string
	subUsage uint64
}

var AllClusters []Cluster

var (
	PLAN_CRITERION = []string{"free", "basic", "premium", "enterprise"}
	// 0 = under a month, 6 = between 1-6 months, 12 = between 6-12 months
	SUB_USAGE_CRITERION = []uint64{0, 6, 12}
)

func init() {
	AllClusters = constructAllClusters()
}

// constructs all the possible cluster keys
func constructAllClusters() []Cluster {
	var clusters []Cluster
	for _, plan := range PLAN_CRITERION {
		for _, subUsage := range SUB_USAGE_CRITERION {
			if plan == "free" {
				// all free plan users use a single cluster
				clusters = append(clusters, NewCluster(plan, 0))
				break
			}
			clusters = append(clusters, NewCluster(plan, subUsage))
		}
	}

	return clusters
}

// GetCluster returns the subscription's best-fit cluster
func GetCluster(sub Subscription) string {
	for _, cluster := range AllClusters {
		if sub.PlanIndex == cluster.plan && sub.DurationTotal <= cluster.subUsage {
			return cluster.String()
		}
	}
	return ""
}

func NewCluster(plan string, subUsage uint64) Cluster {
	return Cluster{plan: plan, subUsage: subUsage}
}

// String returns a unique string that describes the cluster (can be used as a key)
func (c Cluster) String() string {
	if c.plan == "free" {
		// free plan key is "free"
		return c.plan
	}
	return c.plan + "_" + strconv.FormatUint(c.subUsage, 10)
}
