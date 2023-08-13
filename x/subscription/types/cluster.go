package types

import "strconv"

// To add a new criterion, update the Cluster struct, create an array with
// the criterion values (like PLAN_CRITERION) and add it to constructAllClusters()

type Cluster struct {
	plan     string
	subUsage uint64
}

var AllClusters []Cluster

var (
	PLAN_CRITERION      = []string{"free", "basic", "premium", "enterprise"}
	SUB_USAGE_CRITERION = []uint64{6, 12}
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
		if sub.PlanIndex == cluster.plan && sub.DurationTotal == cluster.subUsage {
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
