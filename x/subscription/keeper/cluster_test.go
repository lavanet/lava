package keeper_test

import (
	"testing"

	subscriptiontypes "github.com/lavanet/lava/x/subscription/types"
	"github.com/stretchr/testify/require"
)

// TestGetCluster checks that consumer’s properties are properly interpreted into desired cluster.
func TestGetCluster(t *testing.T) {
	// setup 5 sub accounts to buy 5 different plans
	ts := newTester(t)
	ts.SetupAccounts(5, 0, 0) // 5 sub, 0 adm, 0 dev

	_, subFree := ts.Account("sub1")
	_, subBasic := ts.Account("sub2")
	_, subPremium := ts.Account("sub3")
	_, subEnterprise := ts.Account("sub4")
	_, subMock := ts.Account("sub5")

	// add valid plans for clusters ("mock" is invalid since it's not part of PLAN_CRITERION)
	plan := ts.Plan("mock")
	for _, planName := range subscriptiontypes.PLAN_CRITERION {
		plan.Index = planName
		ts.AddPlan(planName, plan)
		ts.AdvanceEpoch()
	}

	template := []struct {
		name  string
		sub   string
		plan  string
		valid bool
	}{
		{name: "free sub", sub: subFree, plan: "free", valid: true},
		{name: "basic sub", sub: subBasic, plan: "basic", valid: true},
		{name: "premium sub", sub: subPremium, plan: "premium", valid: true},
		{name: "enterprise sub", sub: subEnterprise, plan: "enterprise", valid: true},
		{name: "mock sub", sub: subMock, plan: "mock", valid: false},
	}

	for _, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ts.TxSubscriptionBuy(tt.sub, tt.sub, tt.plan, 12)
			if !tt.valid {
				// if got here, the subscription should not have been created
				require.NotNil(t, err)
			} else {
				require.Nil(t, err)
			}

			// check that for all sub usage periods, the sub's cluster is correct
			for _, subUsage := range subscriptiontypes.SUB_USAGE_CRITERION {
				// get current subscription
				subRes, err := ts.QuerySubscriptionCurrent(tt.sub)
				sub := subRes.Sub
				require.Nil(t, err)
				if !tt.valid {
					// Current returns sub=nil and err=nil for non-existent sub
					require.Nil(t, sub)
					return
				}

				// create a cluster to get the expected cluster key
				c := subscriptiontypes.NewCluster(tt.plan, subUsage)
				require.Equal(t, c.String(), sub.Cluster)

				// advance months (effectively 4 months, each iteration should make the sub change clusters)
				ts.AdvanceMonths(5)
			}
		})
	}
}
