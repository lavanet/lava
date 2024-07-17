package keeper_test

import (
	"testing"

	"github.com/lavanet/lava/v2/x/subscription/types"
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
	plan := ts.Plan("free")
	plans := []string{"basic", "premium", "enterprise"}
	for _, planName := range plans {
		plan.Index = planName
		ts.AddPlan(planName, plan)
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
			_, err := ts.TxSubscriptionBuy(tt.sub, tt.sub, tt.plan, 12, false, false)
			require.NoError(t, err)

			prevCluster := ""
			for i := 0; i < 3; i++ {
				// get current subscription
				subRes, err := ts.QuerySubscriptionCurrent(tt.sub)
				sub := subRes.Sub
				require.NoError(t, err)

				// create a cluster to get the expected cluster key
				c := types.GetClusterKey(*sub)
				require.Equal(t, c, sub.Cluster)

				if tt.plan == types.FREE_PLAN {
					require.Equal(t, types.FREE_PLAN, c)
				} else {
					require.NotEqual(t, prevCluster, c)
				}
				prevCluster = c

				// advance months (4 months - 5 sec + epochTime, each iteration should make the sub change clusters)
				ts.AdvanceMonths(4)
				ts.AdvanceEpoch()
			}
		})
	}
}
