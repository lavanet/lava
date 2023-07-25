package keeper_test

import (
	"strconv"
	"testing"
	"time"

	commontypes "github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/testutil/common"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	planstypes "github.com/lavanet/lava/x/plans/types"
	projectstypes "github.com/lavanet/lava/x/projects/types"
	"github.com/lavanet/lava/x/subscription/types"
	"github.com/stretchr/testify/require"
)

type tester struct {
	common.Tester
}

func newTester(t *testing.T) *tester {
	ts := &tester{Tester: *common.NewTester(t)}
	ts.AddPlan("mock", common.CreateMockPlan())
	return ts
}

func (ts *tester) getSubscription(consumer string) (types.Subscription, bool) {
	sub, err := ts.QuerySubscriptionCurrent(consumer)
	require.Nil(ts.T, err)
	if sub.Sub == nil {
		return types.Subscription{}, false
	}
	return *sub.Sub, true
}

func TestCreateSubscription(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(2, 0, 4) // 2 sub, 0 adm, 4 dev

	_, sub1Addr := ts.Account("sub1")
	_, sub2Addr := ts.Account("sub2")
	_, dev1Addr := ts.Account("dev1")
	_, dev2Addr := ts.Account("dev2")
	_, dev3Addr := ts.Account("dev3")

	consumers := []string{dev1Addr, dev2Addr, dev3Addr, "invalid"}
	creators := []string{sub1Addr, sub2Addr, "invalid"}

	var plans []planstypes.Plan
	for i := 0; i < 3; i++ {
		plan := ts.Plan("mock")
		plan.Index += strconv.Itoa(i + 1)
		plan.Block = ts.BlockHeight()
		err := ts.TxProposalAddPlans(plan)
		require.Nil(t, err)
		plans = append(plans, plan)
	}

	// delete one plan, and advance to next epoch to take effect
	err := ts.TxProposalDelPlans(plans[2].Index)
	require.Nil(t, err)

	ts.AdvanceEpoch()

	template := []struct {
		name      string
		index     string
		creator   int
		consumers []int
		duration  int
		success   bool
	}{
		{
			name:      "create subscriptions",
			index:     plans[0].Index,
			creator:   0,
			consumers: []int{0, 1},
			duration:  1,
			success:   true,
		},
		{
			name:      "invalid creator",
			index:     plans[0].Index,
			creator:   2,
			consumers: []int{2},
			duration:  1,
			success:   false,
		},
		{
			name:      "invalid consumer",
			index:     plans[0].Index,
			creator:   0,
			consumers: []int{3},
			duration:  1,
			success:   false,
		},
		{
			name:      "duration too long",
			index:     plans[0].Index,
			creator:   0,
			consumers: []int{2},
			duration:  13,
			success:   false,
		},
		{
			name:      "insufficient funds",
			index:     plans[0].Index,
			creator:   1,
			consumers: []int{2},
			duration:  1,
			success:   false,
		},
		{
			name:      "invalid plan",
			index:     "",
			creator:   0,
			consumers: []int{2},
			duration:  1,
			success:   false,
		},
		{
			name:      "unknown plan",
			index:     "no-such-plan",
			creator:   0,
			consumers: []int{2},
			duration:  1,
			success:   false,
		},
		{
			name:      "deleted plan",
			index:     plans[2].Index,
			creator:   0,
			consumers: []int{2},
			duration:  1,
			success:   false,
		},
		{
			name:      "double subscription",
			index:     plans[1].Index,
			creator:   0,
			consumers: []int{0},
			duration:  1,
			success:   false,
		},
	}

	for _, tt := range template {
		for _, consumer := range tt.consumers {
			t.Run(tt.name, func(t *testing.T) {
				sub := types.Subscription{
					Creator:   creators[tt.creator],
					Consumer:  consumers[consumer],
					PlanIndex: tt.index,
				}

				_, err := ts.TxSubscriptionBuy(sub.Creator, sub.Consumer, sub.PlanIndex, tt.duration)
				if tt.success {
					require.Nil(t, err, tt.name)
					_, found := ts.getSubscription(sub.Consumer)
					require.True(t, found, tt.name)
				} else {
					require.NotNil(t, err, tt.name)
				}
			})
		}
	}
}

func TestSubscriptionExpiration(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 0) // 2 sub, 0 adm, 0 dev

	_, sub1Addr := ts.Account("sub1")
	plan := ts.Plan("mock")

	_, err := ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 1)
	require.Nil(t, err)
	_, found := ts.getSubscription(sub1Addr)
	require.True(t, found)

	// advance 1 month + epoch, subscription should expire
	ts.AdvanceMonths(1)
	ts.AdvanceEpoch()

	_, found = ts.getSubscription(sub1Addr)
	require.False(t, found)
}

func TestRenewSubscription(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 0) // 1 sub, 0 adm, 0 dev

	_, sub1Addr := ts.Account("sub1")
	plan := ts.Plan("mock")

	_, err := ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 6)
	require.Nil(t, err)
	_, found := ts.getSubscription(sub1Addr)
	require.True(t, found)

	// fast-forward three months
	ts.AdvanceMonths(3).AdvanceEpoch()
	sub, found := ts.getSubscription(sub1Addr)
	require.True(t, found)
	require.Equal(t, uint64(3), sub.DurationLeft)

	// with 3 months duration left, asking for 12 more should fail
	_, err = ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 12)
	require.NotNil(t, err)

	// but 9 additional month (even 10, the extra month extension below)
	_, err = ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 9)
	require.Nil(t, err)
	sub, found = ts.getSubscription(sub1Addr)
	require.True(t, found)

	require.Equal(t, uint64(12), sub.DurationLeft)
	require.Equal(t, uint64(9), sub.DurationTotal)

	// edit the subscription's plan (allow more CU)
	cuPerEpoch := plan.PlanPolicy.EpochCuLimit
	plan.PlanPolicy.EpochCuLimit += 100

	err = keepertest.SimulatePlansAddProposal(ts.Ctx, ts.Keepers.Plans, []planstypes.Plan{plan})
	require.Nil(t, err)

	// try extending the subscription (we could extend with 1 more month,
	// but since the subscription's plan changed, the extension should fail)
	_, err = ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 1)
	require.NotNil(t, err)
	require.Equal(t, uint64(12), sub.DurationLeft)
	require.Equal(t, uint64(9), sub.DurationTotal)

	// get the subscription's plan and make sure it uses the old plan
	plan, found = ts.FindPlan(sub.PlanIndex, sub.PlanBlock)
	require.True(t, found)
	require.Equal(t, cuPerEpoch, plan.PlanPolicy.EpochCuLimit)

	// delete the plan, and try to renew the subscription again
	err = ts.TxProposalDelPlans(plan.Index)
	require.Nil(t, err)

	ts.AdvanceEpoch()

	// fast-forward another month, renewal should fail
	ts.AdvanceMonths(1).AdvanceEpoch()
	_, found = ts.getSubscription(sub1Addr)
	require.True(t, found)
	_, err = ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 10)
	require.NotNil(t, err)
}

func TestSubscriptionAdminProject(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 0) // 1 sub, 0 adm, 0 dev

	_, sub1Addr := ts.Account("sub1")
	plan := ts.Plan("mock")

	_, err := ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 1)
	require.Nil(t, err)

	// a newly created subscription is expected to have one default project,
	// with the subscription address as its developer key
	_, err = ts.GetProjectDeveloperData(sub1Addr, ts.BlockHeight())
	require.Nil(t, err)
}

func TestMonthlyRechargeCU(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 1, 1) // 1 sub, 1 adm, 1 dev

	_, sub1Addr := ts.Account("sub1")
	_, adm1Addr := ts.Account("adm1")
	_, dev1Addr := ts.Account("dev1")
	plan := ts.Plan("mock")

	_, err := ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 3)
	require.Nil(t, err)

	// add another project under the subcscription
	projectData := projectstypes.ProjectData{
		Name:    "another_project",
		Enabled: true,
		ProjectKeys: []projectstypes.ProjectKey{
			projectstypes.ProjectDeveloperKey(dev1Addr),
		},
		Policy: &planstypes.Policy{
			GeolocationProfile: 1,
			TotalCuLimit:       1000,
			EpochCuLimit:       100,
			MaxProvidersToPair: 3,
		},
	}
	err = ts.TxSubscriptionAddProject(sub1Addr, projectData)
	require.Nil(t, err)

	template := []struct {
		name             string
		subscription     string
		developer        string
		usedCuPerProject uint64 // total sub CU is 1000; each project uses 500
	}{
		{"default project", sub1Addr, sub1Addr, 500},
		{"second project (non-default)", sub1Addr, dev1Addr, 500},
	}
	for ti, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			block1 := ts.BlockHeight()
			ts.AdvanceEpoch()

			// charge the subscription
			err = ts.Keepers.Subscription.ChargeComputeUnitsToSubscription(
				ts.Ctx, tt.subscription, block1, tt.usedCuPerProject)
			require.Nil(t, err)

			// verify the CU charge of the subscription is updated correctly
			sub, found := ts.getSubscription(tt.subscription)
			require.True(t, found)
			require.Equal(t, sub.MonthCuLeft, sub.MonthCuTotal-tt.usedCuPerProject)

			// charge the project
			proj, err := ts.GetProjectForDeveloper(tt.developer, block1)
			require.Nil(t, err)
			err = ts.Keepers.Projects.ChargeComputeUnitsToProject(
				ts.Ctx, proj, block1, tt.usedCuPerProject)
			require.Nil(t, err)

			// verify that project used the CU
			proj, err = ts.GetProjectForDeveloper(tt.developer, block1)
			require.Nil(t, err)
			require.Equal(t, tt.usedCuPerProject, proj.UsedCu)

			block2 := ts.BlockHeight()

			// force fixation entry (by adding project key)
			projKey := []projectstypes.ProjectKey{projectstypes.ProjectAdminKey(adm1Addr)}
			ts.Keepers.Projects.AddKeysToProject(ts.Ctx, projectstypes.ADMIN_PROJECT_NAME, tt.developer, projKey)

			// fast-forward one month
			ts.AdvanceMonths(1).AdvanceEpoch()
			sub, found = ts.getSubscription(sub1Addr)
			require.True(t, found)
			require.Equal(t, sub.DurationTotal-uint64(ti+1), sub.DurationLeft)

			block3 := ts.BlockHeight()

			// check that subscription and project have renewed CUs, and that
			// the project created a snapshot for last month
			sub, found = ts.getSubscription(tt.subscription)
			require.True(t, found)
			require.Equal(t, sub.MonthCuLeft, sub.MonthCuTotal)

			proj, err = ts.GetProjectForDeveloper(tt.developer, block1)
			require.Nil(t, err)
			require.Equal(t, tt.usedCuPerProject, proj.UsedCu)
			proj, err = ts.GetProjectForDeveloper(tt.developer, block2)
			require.Nil(t, err)
			require.Equal(t, tt.usedCuPerProject, proj.UsedCu)
			proj, err = ts.GetProjectForDeveloper(tt.developer, block3)
			require.Nil(t, err)
			require.Equal(t, uint64(0), proj.UsedCu)
		})
	}
}

func TestExpiryTime(t *testing.T) {
	ts := newTester(t)

	template := []struct {
		now    [3]int // year, month, day
		res    [3]int // year, month, day
		months int
	}{
		// monthly
		{[3]int{2000, 3, 1}, [3]int{2000, 4, 1}, 1},
		{[3]int{2000, 3, 30}, [3]int{2000, 4, 28}, 1},
		{[3]int{2000, 3, 31}, [3]int{2000, 4, 28}, 1},
		{[3]int{2000, 2, 1}, [3]int{2000, 3, 1}, 1},
		{[3]int{2000, 2, 28}, [3]int{2000, 3, 28}, 1},
		{[3]int{2001, 2, 28}, [3]int{2001, 3, 28}, 1},
		{[3]int{2000, 2, 29}, [3]int{2000, 3, 28}, 1},
		{[3]int{2000, 1, 28}, [3]int{2000, 2, 28}, 1},
		{[3]int{2001, 1, 28}, [3]int{2001, 2, 28}, 1},
		{[3]int{2000, 1, 29}, [3]int{2000, 2, 28}, 1},
		{[3]int{2001, 1, 29}, [3]int{2001, 2, 28}, 1},
		{[3]int{2000, 1, 30}, [3]int{2000, 2, 28}, 1},
		{[3]int{2001, 1, 30}, [3]int{2001, 2, 28}, 1},
		{[3]int{2000, 1, 31}, [3]int{2000, 2, 28}, 1},
		{[3]int{2001, 1, 31}, [3]int{2001, 2, 28}, 1},
		{[3]int{2001, 12, 31}, [3]int{2002, 1, 28}, 1},
		// duration > 1
		{[3]int{2000, 3, 1}, [3]int{2000, 4, 1}, 2},
		{[3]int{2000, 3, 1}, [3]int{2000, 4, 1}, 6},
		{[3]int{2000, 3, 1}, [3]int{2000, 4, 1}, 12},
	}

	plan := ts.Plan("mock")

	for _, tt := range template {
		now := time.Date(tt.now[0], time.Month(tt.now[1]), tt.now[2], 12, 0, 0, 0, time.UTC)

		t.Run(now.Format("2006-01-02"), func(t *testing.T) {
			// new account per attempt
			_, sub1Addr := ts.AddAccount("tmp", 0, 10000)

			delta := now.Sub(ts.BlockTime())
			ts.AdvanceBlock(delta)

			_, err := ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, tt.months)
			require.Nil(t, err)

			sub, found := ts.getSubscription(sub1Addr)
			require.True(t, found)
			require.Equal(t, uint64(tt.months), sub.DurationTotal)

			// will expire and remove
			ts.AdvanceMonths(tt.months).AdvanceEpoch()
			ts.AdvanceBlockUntilStale()
		})
	}
}

func TestSubscriptionExpire(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 0) // 1 sub, 0 adm, 0 dev

	sub1Acct, sub1Addr := ts.Account("sub1")
	plan := ts.Plan("mock")

	coins := common.NewCoins(10000)
	ts.Keepers.BankKeeper.SetBalance(ts.Ctx, sub1Acct.Addr, coins)

	_, err := ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 1)
	require.Nil(t, err)

	block := ts.BlockHeight()

	_, found := ts.getSubscription(sub1Addr)
	require.True(t, found)

	err = ts.Keepers.Subscription.ChargeComputeUnitsToSubscription(
		ts.Ctx, sub1Addr, block, 10)
	require.Nil(t, err)

	// fast-forward one month
	ts.AdvanceMonths(1).AdvanceEpoch()

	// subscription no longer searchable, but can still charge for previous usage
	_, found = ts.getSubscription(sub1Addr)
	require.False(t, found)

	err = ts.Keepers.Subscription.ChargeComputeUnitsToSubscription(
		ts.Ctx, sub1Addr, block, 10)
	require.Nil(t, err)

	ts.AdvanceBlockUntilStale()

	// subscription no longer charge-able for previous usage
	err = ts.Keepers.Subscription.ChargeComputeUnitsToSubscription(
		ts.Ctx, sub1Addr, block, 10)
	require.NotNil(t, err)
}

func TestPrice(t *testing.T) {
	ts := newTester(t)

	template := []struct {
		name     string
		duration int
		discount uint64
		price    int64
		cost     int64
	}{
		{"1 month", 1, 0, 100, 100},
		{"2 months", 2, 0, 100, 200},
		{"11 months", 11, 0, 100, 1100},
		{"yearly without discount", 12, 0, 100, 1200},
		{"yearly with discount", 12, 25, 100, 900},
	}

	for _, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			// new account per attempt
			sub1Acct, sub1Addr := ts.AddAccount("tmp", 0, 10000)

			plan := ts.Plan("mock")
			plan.AnnualDiscountPercentage = tt.discount
			plan.Price = common.NewCoin(tt.price)
			err := ts.TxProposalAddPlans(plan)
			require.Nil(t, err)

			_, err = ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, tt.duration)
			require.Nil(t, err)

			_, found := ts.getSubscription(sub1Addr)
			require.True(t, found)

			balance := ts.GetBalance(sub1Acct.Addr)
			require.Equal(t, balance, 10000-tt.cost)

			// will expire and remove
			ts.AdvanceMonths(tt.duration)
		})
	}
}

func TestAddProjectToSubscription(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 1, 1) // 1 sub, 0 adm, 2 dev

	_, sub1Addr := ts.Account("sub1")
	_, adm1Addr := ts.Account("adm1")
	_, dev1Addr := ts.Account("dev1")
	plan := ts.Plan("mock")

	_, err := ts.TxSubscriptionBuy(sub1Addr, dev1Addr, plan.Index, 1)
	require.Nil(t, err)

	template := []struct {
		name         string
		subscription string
		anotherAdmin string
		projectName  string
		success      bool
	}{
		{"project admin = regular account", dev1Addr, adm1Addr, "test1", true},
		{"project admin = subscription payer account", dev1Addr, sub1Addr, "test2", true},
		{"bad subscription account (regular account)", adm1Addr, dev1Addr, "test3", false},
		{"bad subscription account (subscription payer account)", sub1Addr, dev1Addr, "test4", false},
		{"bad projectName (duplicate)", dev1Addr, adm1Addr, "invalid:name", false},
	}

	for _, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			projectData := projectstypes.ProjectData{
				Name:    tt.projectName,
				Enabled: true,
				ProjectKeys: []projectstypes.ProjectKey{
					projectstypes.ProjectAdminKey(tt.anotherAdmin),
				},
			}
			projectID := projectstypes.ProjectIndex(tt.subscription, tt.projectName)
			err = ts.TxSubscriptionAddProject(tt.subscription, projectData)
			if tt.success {
				require.Nil(t, err)
				proj, err := ts.GetProjectForBlock(projectID, ts.BlockHeight())
				require.Nil(t, err)
				require.Equal(t, tt.subscription, proj.Subscription)
			} else {
				require.NotNil(t, err)
			}
		})
	}
}

func TestGetProjectsForSubscription(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(2, 0, 0) // 2 sub, 0 adm, 0 dev

	_, sub1Addr := ts.Account("sub1")
	_, sub2Addr := ts.Account("sub2")
	plan := ts.Plan("mock")

	// buy two subscriptions
	_, err := ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 1)
	require.Nil(t, err)
	_, err = ts.TxSubscriptionBuy(sub2Addr, sub2Addr, plan.Index, 1)
	require.Nil(t, err)

	// add two projects to the first subscription
	projData1 := projectstypes.ProjectData{
		Name:    "proj1",
		Enabled: true,
		Policy:  &plan.PlanPolicy,
	}
	err = ts.TxSubscriptionAddProject(sub1Addr, projData1)
	require.Nil(t, err)

	projData2 := projectstypes.ProjectData{
		Name:    "proj2",
		Enabled: false,
		Policy:  &plan.PlanPolicy,
	}
	err = ts.TxSubscriptionAddProject(sub1Addr, projData2)
	require.Nil(t, err)

	res1, err := ts.QuerySubscriptionListProjects(sub1Addr)
	require.Nil(t, err)

	res2, err := ts.QuerySubscriptionListProjects(sub2Addr)
	require.Nil(t, err)

	// number of projects +1 to account for auto-generated admin project
	require.Equal(t, 3, len(res1.Projects))
	require.Equal(t, 1, len(res2.Projects))

	err = ts.TxSubscriptionDelProject(sub1Addr, projData2.Name)
	require.Nil(t, err)
}

func TestAddDelProjectForSubscription(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 0) // 1 sub, 0 adm, 0 dev

	_, sub1Addr := ts.Account("sub1")
	plan := ts.Plan("mock")

	// buy subscription and add project
	_, err := ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 1)
	require.Nil(t, err)

	projData := projectstypes.ProjectData{
		Name:    "proj",
		Enabled: true,
		Policy:  &plan.PlanPolicy,
	}
	err = ts.TxSubscriptionAddProject(sub1Addr, projData)
	require.Nil(t, err)

	ts.AdvanceEpoch()

	res, err := ts.QuerySubscriptionListProjects(sub1Addr)
	require.Nil(t, err)
	require.Equal(t, 2, len(res.Projects))

	// del project to the subscription
	err = ts.TxSubscriptionDelProject(sub1Addr, projData.Name)
	require.Nil(t, err)

	ts.AdvanceEpoch()

	res, err = ts.QuerySubscriptionListProjects(sub1Addr)
	require.Nil(t, err)
	require.Equal(t, 1, len(res.Projects))
}

func TestDelProjectEndSubscription(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 0) // 1 sub, 0 adm, 0 dev

	_, sub1Addr := ts.Account("sub1")
	plan := ts.Plan("mock")

	// buy subscription
	_, err := ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 1)
	require.Nil(t, err)

	// time of buy subscription
	start := ts.BlockTime()

	// add project to the subscription
	projData := projectstypes.ProjectData{
		Name:    "proj",
		Enabled: true,
		Policy:  &plan.PlanPolicy,
	}
	err = ts.TxSubscriptionAddProject(sub1Addr, projData)
	require.Nil(t, err)

	ts.AdvanceEpoch()

	res, err := ts.QuerySubscriptionListProjects(sub1Addr)
	require.Nil(t, err)
	require.Equal(t, 2, len(res.Projects))

	// advance time to just before subscription expiry, so project deletion
	// and the subsequent expiry will occur in the same epoch
	ts.AdvanceMonthsFrom(start, 1)

	// del project to the subscription
	err = ts.TxSubscriptionDelProject(sub1Addr, projData.Name)
	require.Nil(t, err)

	// expire subscription (by advancing an epoch, we are close enough to expiry)
	ts.AdvanceEpoch()

	_, err = ts.QuerySubscriptionListProjects(sub1Addr)
	require.NotNil(t, err)

	// should not panic
	ts.AdvanceBlock(2 * commontypes.STALE_ENTRY_TIME)
}
