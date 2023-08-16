package types

import "strconv"

type Cluster struct {
	Plan     string
	SubUsage uint64
}

const FREE_PLAN = "free" // gets its own const because it's treated different

// String returns a unique string that describes the cluster (can be used as a key)
func (c Cluster) String() string {
	if c.Plan == FREE_PLAN {
		// free plan key is "free"
		return c.Plan
	}
	return c.Plan + "_" + strconv.FormatUint(c.SubUsage, 10)
}

func NewCluster(plan string, subUsage uint64) Cluster {
	return Cluster{Plan: plan, SubUsage: subUsage}
}

func GetSubUsageCriterion() []uint64 {
	// 0 = under a month, 6 = between 1-6 months, 12 = between 6-12 months
	return []uint64{0, 6, 12}
}
