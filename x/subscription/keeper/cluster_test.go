package keeper_test

import (
	"testing"

	subscriptiontypes "github.com/lavanet/lava/x/subscription/types"
	"github.com/stretchr/testify/require"
)

// TestGetCluster checks that consumer’s properties are properly interpreted into desired cluster.
func TestGetCluster(t *testing.T) {
	// setup 4 sub accounts to buy 4 different plans
	ts := newTester(t)
	ts.SetupAccounts(4, 0, 0) // 5 sub, 0 adm, 0 dev

	_, subFree := ts.Account("sub1")
	_, subBasic := ts.Account("sub2")
	_, subPremium := ts.Account("sub3")
	_, subEnterprise := ts.Account("sub4")

	// add valid plans for clusters
	plan := ts.Plan("mock")
	plans := []string{"free", "basic", "premium", "enterprise"}
	for _, planName := range plans {
		plan.Index = planName
		ts.AddPlan(planName, plan)
		ts.AdvanceEpoch()
	}

	template := []struct {
		name string
		sub  string
		plan string
	}{
		{name: "free sub", sub: subFree, plan: "free"},
		{name: "basic sub", sub: subBasic, plan: "basic"},
		{name: "premium sub", sub: subPremium, plan: "premium"},
		{name: "enterprise sub", sub: subEnterprise, plan: "enterprise"},
	}

	for _, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ts.TxSubscriptionBuy(tt.sub, tt.sub, tt.plan, 12)
			require.Nil(t, err)

			// check that for all sub usage periods, the sub's cluster is correct
			for _, subUsage := range subscriptiontypes.GetSubUsageCriterion() {
				// get current subscription
				subRes, err := ts.QuerySubscriptionCurrent(tt.sub)
				sub := subRes.Sub
				require.Nil(t, err)

				// create a cluster to get the expected cluster key
				c := subscriptiontypes.NewCluster(tt.plan, subUsage)
				require.Equal(t, c.String(), sub.Cluster)

				// advance months (4 months - 5 sec + epochTime, each iteration should make the sub change clusters)
				ts.AdvanceMonths(4)
				ts.AdvanceEpoch()
			}
		})
	}
}
