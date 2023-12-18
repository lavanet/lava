package keeper_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/lavanet/lava/testutil/common"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/utils/sigs"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
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
	freePlan := common.CreateMockPlan()
	freePlan.Block = ts.BlockHeight()
	ts.AddPlan("free", freePlan)

	premiumPlan := common.CreateMockPlan()
	premiumPlan.Index = "premium"
	premiumPlan.Price = freePlan.Price.AddAmount(math.NewInt(100))
	premiumPlan.Block = ts.BlockHeight()
	ts.AddPlan(premiumPlan.Index, premiumPlan)

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

func getSubscriptionAndFailTestIfNotFound(t *testing.T, ts *tester, consumer string) types.Subscription {
	sub, found := ts.getSubscription(consumer)
	require.True(t, found)
	require.NotNil(t, sub)
	return sub
}

func getProjectAndFailTestIfNotFound(t *testing.T, ts *tester, consumer string, block uint64) projectstypes.Project {
	project, err := ts.GetProjectForDeveloper(consumer, block)
	require.Nil(t, err)
	require.NotNil(t, project)
	return project
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
		plan := ts.Plan("free")
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
			name:      "upgrade subscription",
			index:     plans[1].Index,
			creator:   0,
			consumers: []int{0},
			duration:  1,
			success:   true,
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

				_, err := ts.TxSubscriptionBuy(sub.Creator, sub.Consumer, sub.PlanIndex, tt.duration, false, false)
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
	plan := ts.Plan("free")

	_, err := ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 1, false, false)
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
	plan := ts.Plan("free")

	_, err := ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 6, false, false)
	require.Nil(t, err)
	_, found := ts.getSubscription(sub1Addr)
	require.True(t, found)

	// fast-forward three months
	ts.AdvanceMonths(3).AdvanceEpoch()
	sub, found := ts.getSubscription(sub1Addr)
	require.True(t, found)
	require.Equal(t, uint64(3), sub.DurationLeft)

	// with 3 months duration left, asking for 12 more should fail
	_, err = ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 12, false, false)
	require.NotNil(t, err)

	// but 9 additional month (even 10, the extra month extension below)
	_, err = ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 9, false, false)
	require.Nil(t, err)
	sub, found = ts.getSubscription(sub1Addr)
	require.True(t, found)

	require.Equal(t, uint64(12), sub.DurationLeft)
	require.Equal(t, uint64(9), sub.DurationBought)

	// edit the subscription's plan (allow more CU)
	cuPerEpoch := plan.PlanPolicy.EpochCuLimit
	plan.PlanPolicy.EpochCuLimit += 100
	plan.Price.Amount = plan.Price.Amount.MulRaw(2)

	err = keepertest.SimulatePlansAddProposal(ts.Ctx, ts.Keepers.Plans, []planstypes.Plan{plan}, false)
	require.Nil(t, err)

	// try extending the subscription (we could extend with 1 more month,
	// but since the subscription's plan changed, it should create a future subscription instead
	_, err = ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 1, false, false)
	require.NoError(t, err)
	sub = getSubscriptionAndFailTestIfNotFound(t, ts, sub.Consumer)
	require.NotNil(t, sub.FutureSubscription)
	require.Equal(t, uint64(12), sub.DurationLeft)
	require.Equal(t, uint64(9), sub.DurationBought)

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
	_, err = ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 10, false, false)
	require.NotNil(t, err)
}

func TestSubscriptionAdminProject(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 0) // 1 sub, 0 adm, 0 dev

	_, sub1Addr := ts.Account("sub1")
	plan := ts.Plan("free")

	_, err := ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 1, false, false)
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
	plan := ts.Plan("free")

	_, err := ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 3, false, false)
	require.Nil(t, err)

	// add another project under the subscription
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
			_, err = ts.Keepers.Subscription.ChargeComputeUnitsToSubscription(
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
			require.Equal(t, sub.DurationBought-uint64(ti+1), sub.DurationLeft)

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

	plan := ts.Plan("free")

	for _, tt := range template {
		now := time.Date(tt.now[0], time.Month(tt.now[1]), tt.now[2], 12, 0, 0, 0, time.UTC)

		t.Run(now.Format("2006-01-02"), func(t *testing.T) {
			// new account per attempt
			_, sub1Addr := ts.AddAccount("tmp", 0, 10000)

			delta := now.Sub(ts.BlockTime())
			ts.AdvanceBlock(delta)

			_, err := ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, tt.months, false, false)
			require.Nil(t, err)

			sub, found := ts.getSubscription(sub1Addr)
			require.True(t, found)
			require.Equal(t, uint64(tt.months), sub.DurationBought)

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
	plan := ts.Plan("free")

	coins := common.NewCoins(ts.TokenDenom(), 10000)
	ts.Keepers.BankKeeper.SetBalance(ts.Ctx, sub1Acct.Addr, coins)

	_, err := ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 1, false, false)
	require.Nil(t, err)

	block := ts.BlockHeight()

	_, found := ts.getSubscription(sub1Addr)
	require.True(t, found)

	_, err = ts.Keepers.Subscription.ChargeComputeUnitsToSubscription(
		ts.Ctx, sub1Addr, block, 10)
	require.Nil(t, err)

	// fast-forward one month
	ts.AdvanceMonths(1).AdvanceEpoch()

	// subscription no longer searchable, but can still charge for previous usage
	_, found = ts.getSubscription(sub1Addr)
	require.False(t, found)

	_, err = ts.Keepers.Subscription.ChargeComputeUnitsToSubscription(
		ts.Ctx, sub1Addr, block, 10)
	require.Nil(t, err)

	ts.AdvanceBlockUntilStale()

	// subscription no longer charge-able for previous usage
	_, err = ts.Keepers.Subscription.ChargeComputeUnitsToSubscription(
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

			plan := ts.Plan("free")
			plan.AnnualDiscountPercentage = tt.discount
			plan.Price = common.NewCoin(ts.TokenDenom(), tt.price)
			err := ts.TxProposalAddPlans(plan)
			require.Nil(t, err)

			_, err = ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, tt.duration, false, false)
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
	plan := ts.Plan("free")

	_, err := ts.TxSubscriptionBuy(sub1Addr, dev1Addr, plan.Index, 1, false, false)
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
	plan := ts.Plan("free")

	// buy two subscriptions
	_, err := ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 1, false, false)
	require.Nil(t, err)
	_, err = ts.TxSubscriptionBuy(sub2Addr, sub2Addr, plan.Index, 1, false, false)
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
	plan := ts.Plan("free")

	// buy subscription and add project
	_, err := ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 1, false, false)
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
	plan := ts.Plan("free")

	// buy subscription
	_, err := ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 1, false, false)
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
	ts.AdvanceBlocks(2 * ts.BlocksToSave())
}

// TestDurationTotal tests that the total duration of the subscription is updated correctly
func TestDurationTotal(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 0) // 1 sub, 0 adm, 0 dev
	months := 12
	plan := ts.Plan("free")

	_, subAddr := ts.Account("sub1")
	_, err := ts.TxSubscriptionBuy(subAddr, subAddr, plan.Index, months, false, false)
	require.Nil(t, err)

	for i := 0; i < months-1; i++ {
		subRes, err := ts.QuerySubscriptionCurrent(subAddr)
		sub := subRes.Sub
		require.Nil(t, err)
		require.Equal(t, uint64(i), sub.DurationTotal)
		ts.AdvanceMonths(1)
		ts.AdvanceEpoch()
	}

	// buy extra 4 months and check duration total continues from last count
	subRes, err := ts.QuerySubscriptionCurrent(subAddr)
	require.Nil(t, err)
	durationSoFar := subRes.Sub.DurationTotal

	extraMonths := 4
	_, err = ts.TxSubscriptionBuy(subAddr, subAddr, plan.Index, extraMonths, false, false)
	require.Nil(t, err)

	for i := 0; i < extraMonths; i++ {
		subRes, err := ts.QuerySubscriptionCurrent(subAddr)
		sub := subRes.Sub
		require.Nil(t, err)
		require.Equal(t, uint64(i)+durationSoFar, sub.DurationTotal)
		ts.AdvanceMonths(1)
		ts.AdvanceEpoch()
	}

	// expire subscription and buy a new one. verify duration total starts from scratch
	ts.AdvanceMonths(1)
	ts.AdvanceEpoch()
	subRes, err = ts.QuerySubscriptionCurrent(subAddr)
	require.Nil(t, err)
	require.Nil(t, subRes.Sub)

	_, err = ts.TxSubscriptionBuy(subAddr, subAddr, plan.Index, extraMonths, false, false)
	require.Nil(t, err)
	subRes, err = ts.QuerySubscriptionCurrent(subAddr)
	require.Nil(t, err)
	require.Equal(t, uint64(0), subRes.Sub.DurationTotal)
}

// TestSubAutoRenewal is a happy flow test for subscription auto-renewal
// checks that the two methods for enabling auto renewal works
// verifies that subs with auto-renewal enabled get renewed automatically
func TestSubAutoRenewal(t *testing.T) {
	ts := newTester(t)
	subA := "A"
	subB := "B"
	subC := "C"

	plan := ts.Plan("free")

	testCases := []struct {
		creator                   string
		consumer                  string
		immediatelyBuyAutoRenewal bool
		buyAutoRenewal            bool
		autoRenewalCreator        string
		autoRenewalConsumer       string
		shouldFail                bool
	}{
		{
			creator:                   subA,
			consumer:                  subA,
			immediatelyBuyAutoRenewal: true,
			buyAutoRenewal:            false,
			autoRenewalCreator:        subA,
			autoRenewalConsumer:       subA,
			shouldFail:                false,
		},
		{
			creator:                   subA,
			consumer:                  subA,
			immediatelyBuyAutoRenewal: false,
			buyAutoRenewal:            true,
			autoRenewalCreator:        subA,
			autoRenewalConsumer:       subA,
			shouldFail:                false,
		},
		{
			creator:                   subA,
			consumer:                  subA,
			immediatelyBuyAutoRenewal: false,
			buyAutoRenewal:            false,
			autoRenewalCreator:        subA,
			autoRenewalConsumer:       subA,
			shouldFail:                false,
		},
		{
			creator:                   subA,
			consumer:                  subA,
			immediatelyBuyAutoRenewal: true,
			buyAutoRenewal:            true,
			autoRenewalCreator:        subA,
			autoRenewalConsumer:       subA,
			shouldFail:                false,
		},
		{
			creator:                   subA,
			consumer:                  subB,
			immediatelyBuyAutoRenewal: true,
			buyAutoRenewal:            false,
			autoRenewalCreator:        subA,
			autoRenewalConsumer:       subB,
			shouldFail:                false,
		},
		{
			creator:                   subA,
			consumer:                  subB,
			immediatelyBuyAutoRenewal: false,
			buyAutoRenewal:            true,
			autoRenewalCreator:        subA,
			autoRenewalConsumer:       subB,
			shouldFail:                false,
		},
		{
			creator:                   subA,
			consumer:                  subB,
			immediatelyBuyAutoRenewal: true,
			buyAutoRenewal:            true,
			autoRenewalCreator:        subA,
			autoRenewalConsumer:       subB,
			shouldFail:                false,
		},
		{
			creator:                   subA,
			consumer:                  subB,
			immediatelyBuyAutoRenewal: false,
			buyAutoRenewal:            true,
			autoRenewalCreator:        subA,
			autoRenewalConsumer:       subA,
			shouldFail:                true,
		},
		{
			creator:                   subA,
			consumer:                  subB,
			immediatelyBuyAutoRenewal: false,
			buyAutoRenewal:            true,
			autoRenewalCreator:        subB,
			autoRenewalConsumer:       subB,
			shouldFail:                false,
		},
		{
			creator:                   subA,
			consumer:                  subB,
			immediatelyBuyAutoRenewal: false,
			buyAutoRenewal:            true,
			autoRenewalCreator:        subC,
			autoRenewalConsumer:       subB,
			shouldFail:                true,
		},
	}

	addAccounts := func(idx int) []string {
		creatorAccountName := testCases[idx].creator
		consumerAccountName := testCases[idx].consumer
		autoRenewalCreatorAccountName := testCases[idx].autoRenewalCreator
		autoRenewalConsumerAccountName := testCases[idx].autoRenewalConsumer

		accounts := map[string]struct{}{}
		if _, ok := accounts[creatorAccountName]; !ok {
			accounts[creatorAccountName] = struct{}{}
		}
		if _, ok := accounts[consumerAccountName]; !ok {
			accounts[consumerAccountName] = struct{}{}
		}
		if _, ok := accounts[autoRenewalCreatorAccountName]; !ok {
			accounts[autoRenewalCreatorAccountName] = struct{}{}
		}
		if _, ok := accounts[autoRenewalConsumerAccountName]; !ok {
			accounts[autoRenewalConsumerAccountName] = struct{}{}
		}

		for sub := range accounts {
			ts.AddAccount(sub, idx, 1000000)
		}

		return []string{creatorAccountName, consumerAccountName, autoRenewalCreatorAccountName, autoRenewalConsumerAccountName}
	}

	allAccounts := map[int][]string{}

	for i := 0; i < len(testCases); i++ {
		allAccounts[i] = addAccounts(i)
	}

	for i := 0; i < len(testCases); i++ {
		_, creatorAccountAddr := ts.GetAccount(allAccounts[i][0], i)
		_, consumerAccountAddr := ts.GetAccount(allAccounts[i][1], i)
		_, autoRenewalCreatorAccountAddr := ts.GetAccount(allAccounts[i][2], i)
		_, autoRenewalConsumerAccountAddr := ts.GetAccount(allAccounts[i][3], i)

		testCase := testCases[i]

		_, err := ts.TxSubscriptionBuy(creatorAccountAddr, consumerAccountAddr, plan.Index, 1, testCase.immediatelyBuyAutoRenewal, false)
		require.Nil(t, err)

		if testCase.buyAutoRenewal {
			err = ts.TxSubscriptionAutoRenewal(autoRenewalCreatorAccountAddr, autoRenewalConsumerAccountAddr, plan.Index, true)
			if !testCase.shouldFail {
				require.Nil(t, err, testCase)
			} else {
				require.NotNil(t, err, testCase)
			}
		}

		sub, found := ts.getSubscription(consumerAccountAddr)
		require.True(t, found)
		if (testCase.immediatelyBuyAutoRenewal || testCase.buyAutoRenewal) && !testCase.shouldFail {
			require.Equal(t, sub.AutoRenewalNextPlan, plan.Index, testCase)
			require.Equal(t, sub.Creator, autoRenewalCreatorAccountAddr, testCase)
		} else {
			require.Equal(t, sub.AutoRenewalNextPlan, types.AUTO_RENEWAL_PLAN_NONE, testCase)
		}
	}

	// advance a couple of months to expire and automatically
	// extend all subscriptions. verify that sub1 and sub2 can
	// still be found and their duration left is always 1
	for month := 0; month < 5; month++ {
		ts.AdvanceMonths(1).AdvanceEpoch()

		for i := 0; i < len(testCases); i++ {
			testCase := testCases[i]

			_, consumerAccountAddr := ts.GetAccount(allAccounts[i][1], i)
			newSub, found := ts.getSubscription(consumerAccountAddr)
			if (testCase.immediatelyBuyAutoRenewal || testCase.buyAutoRenewal) && !testCase.shouldFail {
				require.True(t, found, testCase)
				require.Equal(t, uint64(1), newSub.DurationLeft, testCase)
			} else {
				require.False(t, found, testCase)
			}
		}
	}
}

func TestSubAutoRenewalDisable(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(3, 0, 0) // 3 sub, 0 adm, 0 dev

	freePlan := ts.Plan("free")
	_, consumer1 := ts.Account("sub1")
	_, creator2 := ts.Account("sub2")
	_, consumer2 := ts.Account("sub3")

	// Buy subscription with auto-renewal on
	_, err := ts.TxSubscriptionBuy(consumer1, consumer1, freePlan.Index, 1, true, false)
	require.Nil(t, err)
	// Check subscription has auto-renewal
	sub, found := ts.getSubscription(consumer1)
	require.True(t, found)
	require.Equal(t, freePlan.Index, sub.AutoRenewalNextPlan)

	// Disable auto-renewal
	err = ts.TxSubscriptionAutoRenewal(consumer1, consumer1, freePlan.Index, false)
	require.Nil(t, err)
	// Check subscription does not have auto-renewal
	sub, found = ts.getSubscription(consumer1)
	require.True(t, found)
	require.Equal(t, types.AUTO_RENEWAL_PLAN_NONE, sub.AutoRenewalNextPlan)

	ts.AdvanceMonths(1).AdvanceEpoch()

	// Check subscription is expired
	sub, found = ts.getSubscription(consumer1)
	require.False(t, found)

	// Buy subscription with auto-renewal on
	_, err = ts.TxSubscriptionBuy(creator2, consumer2, freePlan.Index, 1, true, false)
	require.Nil(t, err)
	// Check subscription has auto-renewal
	sub, found = ts.getSubscription(consumer2)
	require.True(t, found)
	require.Equal(t, freePlan.Index, sub.AutoRenewalNextPlan)

	// Disable auto-renewal
	err = ts.TxSubscriptionAutoRenewal(creator2, consumer2, freePlan.Index, false)
	require.Nil(t, err)
	// Check subscription does not have auto-renewal
	sub, found = ts.getSubscription(consumer2)
	require.True(t, found)
	require.Equal(t, types.AUTO_RENEWAL_PLAN_NONE, sub.AutoRenewalNextPlan)

	ts.AdvanceMonths(1).AdvanceEpoch()

	// Check subscription is expired
	sub, found = ts.getSubscription(consumer2)
	require.False(t, found)
}

func TestSubAutoRenewalDifferentPlanIndexOnSubBuy(t *testing.T) {
	ts := newTester(t)

	freePlan := ts.Plan("free")
	premiumPlan := ts.Plan("premium")
	creatorAcc, creatorAddr := ts.AddAccount("sub", 1, 20000)
	creatorBalance := ts.GetBalance(creatorAcc.Addr)
	_, consumerAddr := ts.AddAccount("sub", 2, 200000)

	// Buy subscription with auto-renewal on
	_, err := ts.TxSubscriptionBuy(creatorAddr, consumerAddr, freePlan.Index, 1, true, false)
	require.Nil(t, err)
	// Check subscription has auto-renewal
	sub, found := ts.getSubscription(consumerAddr)
	require.True(t, found)
	require.Equal(t, freePlan.Index, sub.AutoRenewalNextPlan)
	// Check creator paid
	creatorBalance -= freePlan.Price.Amount.Int64()
	require.Equal(t, creatorBalance, ts.GetBalance(creatorAcc.Addr))

	ts.AdvanceMonths(1).AdvanceEpoch()

	// Check new subscription
	sub, found = ts.getSubscription(consumerAddr)
	require.True(t, found)
	require.Equal(t, freePlan.Index, sub.AutoRenewalNextPlan)
	require.Equal(t, freePlan.Index, sub.PlanIndex)

	// Check creator paid for free plan
	creatorBalance -= freePlan.Price.Amount.Int64()
	require.Equal(t, creatorBalance, ts.GetBalance(creatorAcc.Addr))

	// Set auto-renewal to premium
	err = ts.TxSubscriptionAutoRenewal(creatorAddr, consumerAddr, premiumPlan.Index, true)
	require.Nil(t, err)
	// Check subscription has auto-renewal
	sub, found = ts.getSubscription(consumerAddr)
	require.True(t, found)
	require.Equal(t, premiumPlan.Index, sub.AutoRenewalNextPlan)
	require.Equal(t, creatorAddr, sub.Creator)

	ts.AdvanceMonths(1).AdvanceEpoch()

	// Check new subscription
	sub, found = ts.getSubscription(consumerAddr)
	require.True(t, found)
	require.Equal(t, premiumPlan.Index, sub.AutoRenewalNextPlan)
	require.Equal(t, premiumPlan.Index, sub.PlanIndex)

	// Check creator paid for new sub
	creatorBalance -= premiumPlan.Price.Amount.Int64()
	require.Equal(t, creatorBalance, ts.GetBalance(creatorAcc.Addr))
}

func TestSubAutoRenewalDifferentPlanIndexOnSubBuyDifferentCreator(t *testing.T) {
	ts := newTester(t)

	freePlan := ts.Plan("free")
	premiumPlan := ts.Plan("premium")
	creatorAcc, creatorAddr := ts.AddAccount("sub", 1, 20000)
	creatorBalance := ts.GetBalance(creatorAcc.Addr)
	consumerAcc, consumerAddr := ts.AddAccount("sub", 2, 200000)
	consumerBalance := ts.GetBalance(consumerAcc.Addr)

	// Buy subscription with auto-renewal on
	_, err := ts.TxSubscriptionBuy(creatorAddr, consumerAddr, freePlan.Index, 1, true, false)
	require.Nil(t, err)
	// Check subscription has auto-renewal
	sub, found := ts.getSubscription(consumerAddr)
	require.True(t, found)
	require.Equal(t, freePlan.Index, sub.AutoRenewalNextPlan)
	// Check creator paid
	creatorBalance -= freePlan.Price.Amount.Int64()
	require.Equal(t, creatorBalance, ts.GetBalance(creatorAcc.Addr))

	ts.AdvanceMonths(1).AdvanceEpoch()

	// Check new subscription
	sub, found = ts.getSubscription(consumerAddr)
	require.True(t, found)
	require.Equal(t, freePlan.Index, sub.AutoRenewalNextPlan)
	require.Equal(t, freePlan.Index, sub.PlanIndex)

	// Check creator paid for free plan
	creatorBalance -= freePlan.Price.Amount.Int64()
	require.Equal(t, creatorBalance, ts.GetBalance(creatorAcc.Addr))

	// Set auto-renewal to premium
	err = ts.TxSubscriptionAutoRenewal(consumerAddr, consumerAddr, premiumPlan.Index, true)
	require.Nil(t, err)
	// Check subscription has auto-renewal
	sub, found = ts.getSubscription(consumerAddr)
	require.True(t, found)
	require.Equal(t, premiumPlan.Index, sub.AutoRenewalNextPlan)
	require.Equal(t, consumerAddr, sub.Creator)

	ts.AdvanceMonths(1).AdvanceEpoch()

	// Check new subscription
	sub, found = ts.getSubscription(consumerAddr)
	require.True(t, found)
	require.Equal(t, premiumPlan.Index, sub.AutoRenewalNextPlan)
	require.Equal(t, premiumPlan.Index, sub.PlanIndex)

	// Check creator paid for new sub
	consumerBalance -= premiumPlan.Price.Amount.Int64()
	require.Equal(t, consumerBalance, ts.GetBalance(consumerAcc.Addr))
}

func TestSubAutoRenewalDifferentPlanIndexOnAutoRenewTx(t *testing.T) {
	ts := newTester(t)

	freePlan := ts.Plan("free")
	premiumPlan := ts.Plan("premium")
	creatorAcc, creatorAddr := ts.AddAccount("sub", 1, 20000)
	creatorBalance := ts.GetBalance(creatorAcc.Addr)
	_, consumerAddr := ts.AddAccount("sub", 2, 200000)

	// Buy subscription with auto-renewal off
	_, err := ts.TxSubscriptionBuy(creatorAddr, consumerAddr, freePlan.Index, 1, false, false)
	require.Nil(t, err)
	// Check subscription does not have auto-renewal
	sub, found := ts.getSubscription(consumerAddr)
	require.True(t, found)
	require.Equal(t, types.AUTO_RENEWAL_PLAN_NONE, sub.AutoRenewalNextPlan)
	// Check creator paid
	creatorBalance -= freePlan.Price.Amount.Int64()
	require.Equal(t, creatorBalance, ts.GetBalance(creatorAcc.Addr))

	// Enable auto-renewal for premium
	err = ts.TxSubscriptionAutoRenewal(creatorAddr, consumerAddr, premiumPlan.Index, true)
	require.Nil(t, err)
	// Check subscription has auto-renewal
	sub, found = ts.getSubscription(consumerAddr)
	require.True(t, found)
	require.Equal(t, premiumPlan.Index, sub.AutoRenewalNextPlan)
	require.Equal(t, creatorAddr, sub.Creator)

	ts.AdvanceMonths(1).AdvanceEpoch()

	// Check new subscription
	sub, found = ts.getSubscription(consumerAddr)
	require.True(t, found)
	require.Equal(t, premiumPlan.Index, sub.AutoRenewalNextPlan)
	require.Equal(t, premiumPlan.Index, sub.PlanIndex)

	// Check creator paid for new sub
	creatorBalance -= premiumPlan.Price.Amount.Int64()
	require.Equal(t, creatorBalance, ts.GetBalance(creatorAcc.Addr))
}

// TestNextToMonthExpiryQuery checks that the NextToMonthExpiry query works as intended
// scenario - buy 3 subs: 2 at the same time, and one a little after. The query should return the two subs
// then, expire those and expect to get the last one from the query
func TestNextToMonthExpiryQuery(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(3, 0, 0) // 1 sub, 0 adm, 0 dev
	months := 1
	plan := ts.Plan("free")

	_, sub1 := ts.Account("sub1")
	_, sub2 := ts.Account("sub2")
	_, sub3 := ts.Account("sub3")

	// buy 3 subs - 2 at the same time and one a second later
	_, err := ts.TxSubscriptionBuy(sub1, sub1, plan.Index, months, false, false)
	require.Nil(t, err)
	_, err = ts.TxSubscriptionBuy(sub2, sub2, plan.Index, months, false, false)
	require.Nil(t, err)
	sub1Obj, found := ts.getSubscription(sub1)
	require.True(t, found)

	ts.AdvanceBlock(time.Second)
	_, err = ts.TxSubscriptionBuy(sub3, sub3, plan.Index, months, false, false)
	require.Nil(t, err)
	sub3Obj, found := ts.getSubscription(sub3)
	require.True(t, found)
	require.Equal(t, sub3Obj.MonthExpiryTime, sub1Obj.MonthExpiryTime+1) // sub3 should expire one second after sub1

	// query - expect subs 1 and 2 in the output
	res, err := ts.QuerySubscriptionNextToMonthExpiry()
	require.Nil(t, err)
	require.Equal(t, 2, len(res.Subscriptions))

	for _, sub := range res.Subscriptions {
		if sub.Consumer != sub1 && sub.Consumer != sub2 {
			require.Fail(t, "resulting subscription are not sub1 or sub2")
		}
		require.Equal(t, sub1Obj.MonthExpiryTime, sub.MonthExpiry)
	}

	// advance month minus 4 seconds
	ts.AdvanceMonths(1).AdvanceBlock(4 * time.Second)
	ts.AdvanceBlock(time.Second) // separate so advanceMonth would trigger

	// query - expect sub 3 in the output
	res, err = ts.QuerySubscriptionNextToMonthExpiry()
	require.Nil(t, err)
	require.Equal(t, 1, len(res.Subscriptions))
	require.Equal(t, sub3, res.Subscriptions[0].Consumer)
	require.Equal(t, sub3Obj.MonthExpiryTime, res.Subscriptions[0].MonthExpiry)

	// advance another second to expire sub3. Expect empty output from the query
	ts.AdvanceBlock(time.Second)
	res, err = ts.QuerySubscriptionNextToMonthExpiry()
	require.Nil(t, err)
	require.Equal(t, 0, len(res.Subscriptions))
}

func TestSubBuySamePlanBlockUpdated(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 0) // 1 sub, 0 adm, 0 dev

	consumerAcc, consumerAddr := ts.Account("sub1")
	consumerBalance := ts.GetBalance(consumerAcc.Addr)
	plan := ts.Plan("free")

	_, err := ts.TxSubscriptionBuy(consumerAddr, consumerAddr, plan.Index, 1, false, false)
	require.Nil(t, err)
	consumerBalance -= plan.Price.Amount.Int64()
	_, found := ts.getSubscription(consumerAddr)
	require.True(t, found)

	// Advance block so the new plan will get a new block
	ts.AdvanceBlock()

	// Edit the subscription's plan (increase the price)
	plan.PlanPolicy.EpochCuLimit += 100
	plan.Price.Amount = plan.Price.Amount.AddRaw(10)
	planBlock := ts.BlockHeight()

	// Propose new plan
	err = keepertest.SimulatePlansAddProposal(ts.Ctx, ts.Keepers.Plans, []planstypes.Plan{plan}, false)
	require.Nil(t, err)

	// Advance epoch so the new plan will be appended
	ts.AdvanceEpoch()

	// Buy the plan again
	_, err = ts.TxSubscriptionBuy(consumerAddr, consumerAddr, plan.Index, 1, false, false)
	require.Nil(t, err)
	// Make sure that the consumer paid for it
	consumerBalance -= plan.Price.Amount.Int64()
	require.Equal(t, consumerBalance, ts.GetBalance(consumerAcc.Addr))

	// Make sure that the new plan is now a FutureSubscription
	sub, found := ts.getSubscription(consumerAddr)
	require.True(t, found)
	require.NotNil(t, sub.FutureSubscription)
	require.Equal(t, plan.Index, sub.FutureSubscription.PlanIndex)
	require.Equal(t, planBlock, sub.FutureSubscription.PlanBlock)
	require.Equal(t, consumerAddr, sub.FutureSubscription.Creator)
}

func TestSubscriptionUpgrade(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 0) // 1 sub, 0 adm, 0 dev

	_, consumer := ts.Account("sub1")
	freePlan := ts.Plan("free")
	premiumPlan := ts.Plan("premium")

	// Buy free plan
	_, err := ts.TxSubscriptionBuy(consumer, consumer, freePlan.Index, 1, false, false)
	require.Nil(t, err)
	// Verify subscription found inside getSubscription
	getSubscriptionAndFailTestIfNotFound(t, ts, consumer)

	// Charge CU from project so we can differentiate the old project from the new one
	projectCuUsed := uint64(100)
	project := getProjectAndFailTestIfNotFound(t, ts, consumer, ts.BlockHeight())
	ts.Keepers.Projects.ChargeComputeUnitsToProject(ts.Ctx, project, ts.BlockHeight(), projectCuUsed)

	// Validate the charge of CU
	project = getProjectAndFailTestIfNotFound(t, ts, consumer, ts.BlockHeight())
	require.Equal(t, projectCuUsed, project.UsedCu)

	ts.AdvanceEpochs(2)
	sub := getSubscriptionAndFailTestIfNotFound(t, ts, consumer)
	currentDurationTotal := sub.DurationTotal

	// Buy premium plan
	_, err = ts.TxSubscriptionBuy(consumer, consumer, premiumPlan.Index, 1, false, false)
	require.Nil(t, err)

	nextEpoch := ts.GetNextEpoch()

	// Test that the subscription and project are not changed until next epoch
	for ts.BlockHeight() < nextEpoch {
		sub := getSubscriptionAndFailTestIfNotFound(t, ts, consumer)
		require.Equal(t, freePlan.Index, sub.PlanIndex, "plan should be free until next epoch. Block: %v", ts.BlockHeight())
		require.Equal(t, currentDurationTotal, sub.DurationTotal)

		project = getProjectAndFailTestIfNotFound(t, ts, consumer, ts.BlockHeight())
		require.Equal(t, projectCuUsed, project.UsedCu)

		ts.AdvanceBlock()
	}

	// Test that the subscription is now updated
	sub = getSubscriptionAndFailTestIfNotFound(t, ts, consumer)
	require.Equal(t, premiumPlan.Index, sub.PlanIndex)

	// Test that the project is now updated
	project = getProjectAndFailTestIfNotFound(t, ts, consumer, ts.BlockHeight())
	require.Equal(t, uint64(0), project.UsedCu)
}

func TestSubscriptionDowngradeFails(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 0) // 1 sub, 0 adm, 0 dev

	_, consumer := ts.Account("sub1")
	freePlan := ts.Plan("free")
	premiumPlan := ts.Plan("premium")

	// Buy premium plan
	_, err := ts.TxSubscriptionBuy(consumer, consumer, premiumPlan.Index, 1, false, false)
	require.Nil(t, err)
	// Verify subscription found inside getSubscription
	getSubscriptionAndFailTestIfNotFound(t, ts, consumer)

	ts.AdvanceEpochs(2)

	// Buy premium plan
	_, err = ts.TxSubscriptionBuy(consumer, consumer, freePlan.Index, 1, false, false)
	require.NotNil(t, err)

	ts.AdvanceEpoch()

	sub := getSubscriptionAndFailTestIfNotFound(t, ts, consumer)
	require.Equal(t, premiumPlan.Index, sub.PlanIndex)
}

func TestSubscriptionCuExhaustAndUpgrade(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 0) // 1 sub, 0 adm, 0 dev
	consumerAcc, consumerAddr := ts.Account("sub1")

	spec := ts.AddSpec("testSpec", common.CreateMockSpec()).Spec("testSpec")

	// Setup validator and provider
	testBalance := int64(1000000)
	testStake := int64(100000)

	validationAcc, _ := ts.AddAccount(common.VALIDATOR, 0, testBalance)
	ts.TxCreateValidator(validationAcc, math.NewInt(testBalance))

	_, providerAddr := ts.AddAccount(common.PROVIDER, 0, testBalance)
	err := ts.StakeProviderExtra(providerAddr, spec, testStake, nil, 0, "provider")
	require.Nil(t, err)

	// Trigger changes
	ts.AdvanceEpoch()

	freePlan := ts.Plan("free")
	premiumPlan := ts.Plan("premium")

	// Buy free plan
	_, err = ts.TxSubscriptionBuy(consumerAddr, consumerAddr, freePlan.Index, 3, false, false)
	require.Nil(t, err)

	// Verify subscription found inside getSubscription
	getSubscriptionAndFailTestIfNotFound(t, ts, consumerAddr)

	// Send relay
	sessionId := uint64(1)
	relayNum := uint64(1)
	sendRelayPayment := func() {
		relaySession := &pairingtypes.RelaySession{
			Provider:    providerAddr,
			ContentHash: []byte(spec.ApiCollections[0].Apis[0].Name),
			SessionId:   sessionId,
			SpecId:      spec.Index,
			CuSum:       1000,
			Epoch:       int64(ts.EpochStart(ts.BlockHeight())),
			RelayNum:    relayNum,
		}

		sig, err := sigs.Sign(consumerAcc.SK, *relaySession)
		require.Nil(ts.T, err)
		relaySession.Sig = sig

		_, err = ts.TxPairingRelayPayment(providerAddr, relaySession)
		require.Nil(t, err)

		sessionId++
		relayNum++
	}

	// Send relay under the free subscription
	sendRelayPayment()

	// Buy premium plan
	_, err = ts.TxSubscriptionBuy(consumerAddr, consumerAddr, premiumPlan.Index, 1, false, false)
	require.Nil(t, err)

	// Trigger new subscription
	ts.AdvanceEpoch()

	// Test that the subscription is now updated
	sub := getSubscriptionAndFailTestIfNotFound(t, ts, consumerAddr)
	require.Equal(t, premiumPlan.Index, sub.PlanIndex)

	// Test that the project is now updated
	project := getProjectAndFailTestIfNotFound(t, ts, consumerAddr, ts.BlockHeight())
	require.Equal(t, uint64(0), project.UsedCu)

	// Send relay under the premium subscription
	sendRelayPayment()

	// Advance month + blocksToSave + 1 to trigger the provider monthly payment
	ts.AdvanceMonths(1)
	ts.AdvanceBlocks(ts.BlocksToSave() + 1)

	// Query provider's rewards
	rewards, err := ts.QueryDualstakingDelegatorRewards(providerAddr, providerAddr, spec.Index)
	require.Nil(t, err)
	require.Len(t, rewards.Rewards, 1)
	reward := rewards.Rewards[0]

	// Verify that provider got rewarded for both subscriptions
	require.Equal(t, freePlan.Price.AddAmount(premiumPlan.Price.Amount), reward.Amount)
}

// ### Advance Purchase Tests ###

func TestSubscriptionAdvancePurchaseStartsOnExpirationOfCurrent(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 0) // 1 sub, 0 adm, 0 dev

	consumerAcc, consumerAddr := ts.Account("sub1")
	freePlan := ts.Plan("free")
	premiumPlan := ts.Plan("premium")
	consumerBalance := ts.GetBalance(consumerAcc.Addr)

	// Buy free plan
	freePlanDuration := int64(2)
	_, err := ts.TxSubscriptionBuy(consumerAddr, consumerAddr, freePlan.Index, int(freePlanDuration), false, false)
	require.Nil(t, err)
	// Verify subscription found inside getSubscription
	getSubscriptionAndFailTestIfNotFound(t, ts, consumerAddr)
	consumerBalance -= freePlan.Price.Amount.MulRaw(freePlanDuration).Int64()
	// Make sure the balance checks out
	require.Equal(t, consumerBalance, ts.GetBalance(consumerAcc.Addr))

	newSubDuration := uint64(4)
	_, err = ts.TxSubscriptionBuy(consumerAddr, consumerAddr, premiumPlan.Index, int(newSubDuration), false, true)
	require.Nil(t, err)

	// Verify that the consumer charged with the correct amount
	consumerShouldPay := premiumPlan.Price.Amount.MulRaw(int64(newSubDuration))
	expectedConsumerBalance := consumerBalance - consumerShouldPay.Int64()
	require.Equal(t, expectedConsumerBalance, ts.GetBalance(consumerAcc.Addr))

	// Verify new future subscription
	sub := getSubscriptionAndFailTestIfNotFound(t, ts, consumerAddr)
	futureSub := sub.FutureSubscription
	require.NotNil(t, futureSub)
	require.Equal(t, premiumPlan.Index, futureSub.PlanIndex)
	require.Equal(t, premiumPlan.Block, futureSub.PlanBlock)
	require.Equal(t, newSubDuration, futureSub.DurationBought)

	ts.AdvanceMonths(1).AdvanceEpoch()

	// Should still be the same before the subscription expires
	sub = getSubscriptionAndFailTestIfNotFound(t, ts, consumerAddr)
	futureSub = sub.FutureSubscription
	require.NotNil(t, futureSub)
	require.Equal(t, premiumPlan.Index, futureSub.PlanIndex)
	require.Equal(t, premiumPlan.Block, futureSub.PlanBlock)
	require.Equal(t, newSubDuration, futureSub.DurationBought)

	ts.AdvanceMonths(1).AdvanceEpoch()

	// New subscription should now be active
	sub = getSubscriptionAndFailTestIfNotFound(t, ts, consumerAddr)
	require.Nil(t, sub.FutureSubscription)
	require.Equal(t, premiumPlan.Index, sub.PlanIndex)
	require.Equal(t, premiumPlan.Block, sub.PlanBlock)
	require.Equal(t, newSubDuration, sub.DurationBought)
	require.Equal(t, newSubDuration, sub.DurationLeft)
	require.Equal(t, uint64(2), sub.DurationTotal)
}

func TestSubscriptionAdvancePurchaseSuccessOnPricierPlan_SameBlock(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 0) // 1 sub, 0 adm, 0 dev

	CHEAP := "cheap"
	MEDIUM := "medium"
	EXPENSIVE := "expensive"

	cheapPlan := common.CreateMockPlan()
	cheapPlan.Index = CHEAP
	cheapPlan.Price = common.NewCoin(ts.TokenDenom(), 100)
	cheapPlan.Block = ts.BlockHeight()
	ts.AddPlan(cheapPlan.Index, cheapPlan)

	mediumPlan := common.CreateMockPlan()
	mediumPlan.Index = MEDIUM
	mediumPlan.Price = common.NewCoin(ts.TokenDenom(), 200)
	mediumPlan.Block = ts.BlockHeight()
	ts.AddPlan(mediumPlan.Index, mediumPlan)

	expansivePlan := common.CreateMockPlan()
	expansivePlan.Index = EXPENSIVE
	expansivePlan.Price = common.NewCoin(ts.TokenDenom(), 400)
	expansivePlan.Block = ts.BlockHeight()
	ts.AddPlan(expansivePlan.Index, expansivePlan)

	// We start with the medium plan.
	// All these test cases should be with a new plan that is more expensive than the plan before them:
	// 		1. Expansive plan && less duration
	// 		2. Cheaper plan && more duration
	// 		3. Same plan && more duration
	// 		4. Expansive plan && same duration
	// 		5. Expansive plan && more duration

	startingDuration := int64(3)
	originalPlanCost := mediumPlan.Price.Amount.MulRaw(startingDuration).Int64()
	testCases := []struct {
		name     string
		plan     *planstypes.Plan
		duration int64
		price    int64
	}{
		{
			name:     "Expansive plan && less duration",
			plan:     &expansivePlan,                                                  //   400
			duration: startingDuration - 1,                                            // * 2
			price:    expansivePlan.Price.Amount.MulRaw(startingDuration - 1).Int64(), // = 800,
		},
		{
			name:     "Cheaper plan && more duration",
			plan:     &cheapPlan,                                                  //   100
			duration: startingDuration + 6,                                        // * 9
			price:    cheapPlan.Price.Amount.MulRaw(startingDuration + 6).Int64(), // = 900
		},
		{
			name:     "Same plan && more duration",
			plan:     &mediumPlan,                                                  //   200
			duration: startingDuration + 2,                                         // * 5
			price:    mediumPlan.Price.Amount.MulRaw(startingDuration + 2).Int64(), // = 1000
		},
		{
			name:     "Expansive plan && same duration",
			plan:     &expansivePlan,                                              //   400
			duration: startingDuration,                                            // * 3
			price:    expansivePlan.Price.Amount.MulRaw(startingDuration).Int64(), // = 1200,
		},
		{
			name:     "Expansive plan && more duration",
			plan:     &expansivePlan,                                                  //   400
			duration: startingDuration + 1,                                            // * 4
			price:    expansivePlan.Price.Amount.MulRaw(startingDuration + 1).Int64(), // = 1600,
		},
	}

	consumerAcc, consumerAddr := ts.Account("sub1")
	consumerBalance := ts.GetBalance(consumerAcc.Addr)

	// Buy medium plan
	_, err := ts.TxSubscriptionBuy(consumerAddr, consumerAddr, mediumPlan.Index, 1, false, false)
	require.Nil(t, err)
	// Verify subscription found inside getSubscription
	getSubscriptionAndFailTestIfNotFound(t, ts, consumerAddr)

	consumerBalance -= mediumPlan.Price.Amount.Int64()
	// Make sure the balance checks out
	require.Equal(t, consumerBalance, ts.GetBalance(consumerAcc.Addr))

	// Buy future medium plan
	_, err = ts.TxSubscriptionBuy(consumerAddr, consumerAddr, mediumPlan.Index, int(startingDuration), false, true)
	require.Nil(t, err)
	// Verify subscription found inside getSubscription
	getSubscriptionAndFailTestIfNotFound(t, ts, consumerAddr)

	consumerBalance -= mediumPlan.Price.Amount.MulRaw(startingDuration).Int64()
	// Make sure the balance checks out
	require.Equal(t, consumerBalance, ts.GetBalance(consumerAcc.Addr))

	for _, testCase := range testCases {
		testName := fmt.Sprintf("%s -> Price: %d", testCase.name, testCase.price)
		// Buy new plan
		_, err := ts.TxSubscriptionBuy(consumerAddr, consumerAddr, testCase.plan.Index, int(testCase.duration), false, true)
		require.Nil(t, err, testName)

		priceDiff := testCase.price - originalPlanCost
		consumerBalance -= priceDiff

		// Make sure the balance is updated
		require.Equal(t, consumerBalance, ts.GetBalance(consumerAcc.Addr), testName)
	}
}

func TestSubscriptionAdvancePurchaseSuccessOnPricierPlan_NewBlock(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 0) // 1 sub, 0 adm, 0 dev

	MEDIUM := "medium"

	mediumPlan := common.CreateMockPlan()
	mediumPlan.Index = MEDIUM
	mediumPlan.Price = common.NewCoin(ts.TokenDenom(), 200)
	mediumPlan.Block = ts.BlockHeight()
	ts.AddPlan(mediumPlan.Index, mediumPlan)

	// We start with the medium plan.
	// All these test cases should be with a new plan that is more expensive than current:
	// 		1. Same plan && cheaper && more duration
	// 		2. Same plan && more expensive && less duration
	// 		3. Same plan && more expensive && same duration
	// 		4. Same plan && more expensive && more duration

	startingDuration := int64(2)
	originalPlanCost := mediumPlan.Price.Amount.MulRaw(startingDuration)
	// Original cost: 200 * 2 = 400

	consumerAcc, consumerAddr := ts.Account("sub1")
	consumerBalance := ts.GetBalance(consumerAcc.Addr)

	// Buy medium plan
	_, err := ts.TxSubscriptionBuy(consumerAddr, consumerAddr, mediumPlan.Index, 1, false, false)
	require.Nil(t, err)
	// Verify subscription found inside getSubscription
	getSubscriptionAndFailTestIfNotFound(t, ts, consumerAddr)

	// Make sure the balance checks out
	consumerBalance -= mediumPlan.Price.Amount.Int64()
	require.Equal(t, consumerBalance, ts.GetBalance(consumerAcc.Addr))

	// Buy future medium plan
	_, err = ts.TxSubscriptionBuy(consumerAddr, consumerAddr, mediumPlan.Index, int(startingDuration), false, true)
	require.Nil(t, err)
	// Verify subscription found inside getSubscription
	getSubscriptionAndFailTestIfNotFound(t, ts, consumerAddr)

	// Make sure the balance checks out
	consumerBalance -= mediumPlan.Price.Amount.MulRaw(startingDuration).Int64()
	require.Equal(t, consumerBalance, ts.GetBalance(consumerAcc.Addr))

	ts.AdvanceBlock()

	// Create plan with same index - cheaper price
	mediumPlanCheaper := common.CreateMockPlan()
	mediumPlanCheaper.Index = MEDIUM
	mediumPlanCheaper.Price = common.NewCoin(ts.TokenDenom(), 100)
	mediumPlanCheaper.Block = ts.BlockHeight()
	ts.AddPlan(mediumPlanCheaper.Index, mediumPlanCheaper)

	// 1. Buy new plan
	newPlanDuration := startingDuration + 3                               // 5
	newPlanCost := mediumPlanCheaper.Price.Amount.MulRaw(newPlanDuration) // 500
	_, err = ts.TxSubscriptionBuy(consumerAddr, consumerAddr, mediumPlanCheaper.Index, int(newPlanDuration), false, true)
	require.Nil(t, err, "Same plan && cheaper && more duration -> Price: "+newPlanCost.String())

	priceDiff := newPlanCost.Sub(originalPlanCost).Int64()
	consumerBalance -= priceDiff

	// Make sure the balance has changed
	require.Equal(t, consumerBalance, ts.GetBalance(consumerAcc.Addr))

	ts.AdvanceBlock()

	// Create plan with same index - higher price
	mediumPlanExpensive := common.CreateMockPlan()
	mediumPlanExpensive.Index = MEDIUM
	mediumPlanExpensive.Price = common.NewCoin(ts.TokenDenom(), 600)
	mediumPlanExpensive.Block = ts.BlockHeight()
	ts.AddPlan(mediumPlanExpensive.Index, mediumPlanExpensive)

	// 2. Buy new plan
	newPlanDuration = startingDuration - 1                                 // 1
	newPlanCost = mediumPlanExpensive.Price.Amount.MulRaw(newPlanDuration) // 600
	_, err = ts.TxSubscriptionBuy(consumerAddr, consumerAddr, mediumPlanExpensive.Index, int(newPlanDuration), false, true)
	require.Nil(t, err, "Same plan && more expensive && less duration -> Price: "+newPlanCost.String())

	priceDiff = newPlanCost.Sub(originalPlanCost).Int64()
	consumerBalance -= priceDiff

	// Make sure the balance has changed
	require.Equal(t, consumerBalance, ts.GetBalance(consumerAcc.Addr))

	// 3. Buy new plan
	newPlanDuration = startingDuration                                     // 2
	newPlanCost = mediumPlanExpensive.Price.Amount.MulRaw(newPlanDuration) // 1200
	_, err = ts.TxSubscriptionBuy(consumerAddr, consumerAddr, mediumPlanExpensive.Index, int(newPlanDuration), false, true)
	require.Nil(t, err, "Same plan && more expensive && same duration -> Price: "+newPlanCost.String())

	priceDiff = newPlanCost.Sub(originalPlanCost).Int64()
	consumerBalance -= priceDiff

	// Make sure the balance has changed
	require.Equal(t, consumerBalance, ts.GetBalance(consumerAcc.Addr))

	// 4. Buy new plan
	newPlanDuration = startingDuration + 1                                 // 3
	newPlanCost = mediumPlanExpensive.Price.Amount.MulRaw(newPlanDuration) // 1800
	_, err = ts.TxSubscriptionBuy(consumerAddr, consumerAddr, mediumPlanExpensive.Index, int(newPlanDuration), false, true)
	require.Nil(t, err, "Same plan && more expensive && more duration -> Price: "+newPlanCost.String())

	priceDiff = newPlanCost.Sub(originalPlanCost).Int64()
	consumerBalance -= priceDiff

	// Make sure the balance has changed
	require.Equal(t, consumerBalance, ts.GetBalance(consumerAcc.Addr))
}

func TestSubscriptionAdvancePurchaseFailOnCheaperPlans_SameBlock(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 0) // 1 sub, 0 adm, 0 dev

	CHEAP := "cheap"
	MEDIUM := "medium"
	EXPENSIVE := "expensive"

	cheapPlan := common.CreateMockPlan()
	cheapPlan.Index = CHEAP
	cheapPlan.Price = common.NewCoin(ts.TokenDenom(), 100)
	cheapPlan.Block = ts.BlockHeight()
	ts.AddPlan(cheapPlan.Index, cheapPlan)

	mediumPlan := common.CreateMockPlan()
	mediumPlan.Index = MEDIUM
	mediumPlan.Price = common.NewCoin(ts.TokenDenom(), 200)
	mediumPlan.Block = ts.BlockHeight()
	ts.AddPlan(mediumPlan.Index, mediumPlan)

	expansivePlan := common.CreateMockPlan()
	expansivePlan.Index = EXPENSIVE
	expansivePlan.Price = common.NewCoin(ts.TokenDenom(), 300)
	expansivePlan.Block = ts.BlockHeight()
	ts.AddPlan(expansivePlan.Index, expansivePlan)

	// We start with the medium plan.
	// All these test cases should be with a new plan that is equal or cheaper than current:
	// 		1. Same plan && same duration = equal
	// 		2. Same plan && less duration = cheaper
	// 		3. Cheaper plan && same duration = cheaper
	// 		4. Cheaper plan && more duration = equal
	// 		5. Cheaper plan && more duration = cheaper
	// 		6. Expansive plan && less duration = equal
	// 		7. Expansive plan && less duration = cheaper

	startingDuration := int64(3)
	// Original cost: 200 * 3 = 600
	testCases := []struct {
		name     string
		plan     *planstypes.Plan
		duration int64
		price    int64
	}{
		{
			name:     "Same plan && same duration",
			plan:     &mediumPlan,                                              //   200
			duration: startingDuration,                                         // * 3
			price:    mediumPlan.Price.Amount.MulRaw(startingDuration).Int64(), // = 600
		},
		{
			name:     "Same plan && less duration",
			plan:     &mediumPlan,                                                  //   200
			duration: startingDuration - 1,                                         // * 2
			price:    mediumPlan.Price.Amount.MulRaw(startingDuration - 1).Int64(), // = 400
		},
		{
			name:     "Cheaper plan && same duration",
			plan:     &cheapPlan,                                              //   100
			duration: startingDuration,                                        // * 3
			price:    cheapPlan.Price.Amount.MulRaw(startingDuration).Int64(), // = 300,
		},
		{
			name:     "Cheaper plan && more duration",
			plan:     &cheapPlan,                                                  //   100
			duration: startingDuration + 3,                                        // * 6
			price:    cheapPlan.Price.Amount.MulRaw(startingDuration + 3).Int64(), // = 600,
		},
		{
			name:     "Cheaper plan && more duration",
			plan:     &cheapPlan,                                                  //   100
			duration: startingDuration + 1,                                        // * 4
			price:    cheapPlan.Price.Amount.MulRaw(startingDuration + 1).Int64(), // = 400,
		},
		{
			name:     "Expansive plan && less duration",
			plan:     &expansivePlan,                                                  //   300
			duration: startingDuration - 1,                                            // * 2
			price:    expansivePlan.Price.Amount.MulRaw(startingDuration - 1).Int64(), // = 600,
		},
		{
			name:     "Expansive plan && less duration",
			plan:     &expansivePlan,                                                  //   300
			duration: startingDuration - 2,                                            // * 1
			price:    expansivePlan.Price.Amount.MulRaw(startingDuration - 2).Int64(), // = 300,
		},
	}

	consumerAcc, consumerAddr := ts.Account("sub1")
	consumerBalance := ts.GetBalance(consumerAcc.Addr)

	// Buy medium plan
	_, err := ts.TxSubscriptionBuy(consumerAddr, consumerAddr, mediumPlan.Index, 1, false, false)
	require.Nil(t, err)
	// Verify subscription found inside getSubscription
	getSubscriptionAndFailTestIfNotFound(t, ts, consumerAddr)

	consumerBalance -= mediumPlan.Price.Amount.Int64()
	// Make sure the balance checks out
	require.Equal(t, consumerBalance, ts.GetBalance(consumerAcc.Addr))

	// Buy future medium plan
	_, err = ts.TxSubscriptionBuy(consumerAddr, consumerAddr, mediumPlan.Index, int(startingDuration), false, true)
	require.Nil(t, err)
	// Verify subscription found inside getSubscription
	getSubscriptionAndFailTestIfNotFound(t, ts, consumerAddr)

	consumerBalance -= mediumPlan.Price.Amount.MulRaw(startingDuration).Int64()
	// Make sure the balance checks out
	require.Equal(t, consumerBalance, ts.GetBalance(consumerAcc.Addr))

	for _, testCase := range testCases {
		testName := fmt.Sprintf("%s -> Price: %d", testCase.name, testCase.price)

		// Buy new plan
		_, err := ts.TxSubscriptionBuy(consumerAddr, consumerAddr, testCase.plan.Index, int(testCase.duration), false, true)
		require.NotNil(t, err, testName)

		// Make sure the balance is not changed
		require.Equal(t, consumerBalance, ts.GetBalance(consumerAcc.Addr), testName)
	}
}

func TestSubscriptionAdvancePurchaseFailOnCheaperPlans_NewBlock(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 0) // 1 sub, 0 adm, 0 dev

	MEDIUM := "medium"

	mediumPlan := common.CreateMockPlan()
	mediumPlan.Index = MEDIUM
	mediumPlan.Price = common.NewCoin(ts.TokenDenom(), 200)
	mediumPlan.Block = ts.BlockHeight()
	ts.AddPlan(mediumPlan.Index, mediumPlan)

	// We start with the medium plan.
	// All these test cases should be with a new plan that is equal or cheaper than current:
	// 		1. Same plan && more expensive && less duration = equal
	// 		2. Same plan && more expensive && less duration = cheaper
	// 		3. Same plan && cheaper && more duration = equal
	// 		4. Same plan && cheaper && more duration = cheaper

	startingDuration := int64(3)
	// Original cost: 200 * 3 = 600

	consumerAcc, consumerAddr := ts.Account("sub1")
	consumerBalance := ts.GetBalance(consumerAcc.Addr)

	// Buy medium plan
	_, err := ts.TxSubscriptionBuy(consumerAddr, consumerAddr, mediumPlan.Index, 1, false, false)
	require.Nil(t, err)
	// Verify subscription found inside getSubscription
	getSubscriptionAndFailTestIfNotFound(t, ts, consumerAddr)

	// Make sure the balance checks out
	consumerBalance -= mediumPlan.Price.Amount.Int64()
	require.Equal(t, consumerBalance, ts.GetBalance(consumerAcc.Addr))

	// Buy future medium plan
	_, err = ts.TxSubscriptionBuy(consumerAddr, consumerAddr, mediumPlan.Index, int(startingDuration), false, true)
	require.Nil(t, err)
	// Verify subscription found inside getSubscription
	getSubscriptionAndFailTestIfNotFound(t, ts, consumerAddr)

	// Make sure the balance checks out
	consumerBalance -= mediumPlan.Price.Amount.MulRaw(startingDuration).Int64()
	require.Equal(t, consumerBalance, ts.GetBalance(consumerAcc.Addr))

	ts.AdvanceBlock()

	// Create plan with same index - cheaper price
	mediumPlanCheaper := common.CreateMockPlan()
	mediumPlanCheaper.Index = MEDIUM
	mediumPlanCheaper.Price = common.NewCoin(ts.TokenDenom(), 100)
	mediumPlanCheaper.Block = ts.BlockHeight()
	ts.AddPlan(mediumPlanCheaper.Index, mediumPlanCheaper)

	// Buy new plan
	_, err = ts.TxSubscriptionBuy(consumerAddr, consumerAddr, mediumPlanCheaper.Index, int(startingDuration+3), false, true)
	require.NotNil(t, err, "Same plan && cheaper && more duration -> Price: "+
		mediumPlanCheaper.Price.Amount.MulRaw(startingDuration+3).String()) // 600

	// Make sure the balance is not changed
	require.Equal(t, consumerBalance, ts.GetBalance(consumerAcc.Addr))

	// Buy new plan
	_, err = ts.TxSubscriptionBuy(consumerAddr, consumerAddr, mediumPlanCheaper.Index, int(startingDuration+2), false, true)
	require.NotNil(t, err, "Same plan && cheaper && more duration -> Price: "+
		mediumPlanCheaper.Price.Amount.MulRaw(startingDuration+2).String()) // 500

	// Make sure the balance is not changed
	require.Equal(t, consumerBalance, ts.GetBalance(consumerAcc.Addr))

	ts.AdvanceBlock()

	// Create plan with same index - higher price
	mediumPlanExpensive := common.CreateMockPlan()
	mediumPlanExpensive.Index = MEDIUM
	mediumPlanExpensive.Price = common.NewCoin(ts.TokenDenom(), 300)
	mediumPlanExpensive.Block = ts.BlockHeight()
	ts.AddPlan(mediumPlanExpensive.Index, mediumPlanExpensive)

	// Buy new plan
	_, err = ts.TxSubscriptionBuy(consumerAddr, consumerAddr, mediumPlanCheaper.Index, int(startingDuration-1), false, true)
	require.NotNil(t, err, "Same plan && cheaper && more duration -> Price: "+
		mediumPlanCheaper.Price.Amount.MulRaw(startingDuration-1).String()) // 600

	// Make sure the balance is not changed
	require.Equal(t, consumerBalance, ts.GetBalance(consumerAcc.Addr))

	// Buy new plan
	_, err = ts.TxSubscriptionBuy(consumerAddr, consumerAddr, mediumPlanCheaper.Index, int(startingDuration-2), false, true)
	require.NotNil(t, err, "Same plan && cheaper && more duration -> Price: "+
		mediumPlanCheaper.Price.Amount.MulRaw(startingDuration-2).String()) // 500

	// Make sure the balance is not changed
	require.Equal(t, consumerBalance, ts.GetBalance(consumerAcc.Addr))
}

func TestSubscriptionAdvancePurchaseFailOnNoSubscription(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 0) // 1 sub, 0 adm, 0 dev

	consumerAcc, consumerAddr := ts.Account("sub1")
	premiumPlan := ts.Plan("premium")
	consumerBalance := ts.GetBalance(consumerAcc.Addr)

	// Advance purchase the subscription with no active subscription
	newSubDuration := int64(4)
	_, err := ts.TxSubscriptionBuy(consumerAddr, consumerAddr, premiumPlan.Index, int(newSubDuration), false, true)
	require.NotNil(t, err)

	// Verify that the consumer is not charged
	require.Equal(t, consumerBalance, ts.GetBalance(consumerAcc.Addr))

	// Verify that there is no new subscription
	_, found := ts.getSubscription(consumerAddr)
	require.False(t, found)
}

func TestSubscriptionAdvancePurchaseNewCreator(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 0) // 1 sub, 0 adm, 0 dev

	creator1Acc, creator1Addr := ts.AddAccount("sugar1", 0, 20000)
	creator1Balance := ts.GetBalance(creator1Acc.Addr)

	creator2Acc, creator2Addr := ts.AddAccount("sugar2", 1, 20000)
	creator2Balance := ts.GetBalance(creator2Acc.Addr)

	consumerAcc, consumerAddr := ts.Account("sub1")
	consumerBalance := ts.GetBalance(consumerAcc.Addr)
	freePlan := ts.Plan("free")
	premiumPlan := ts.Plan("premium")

	// Buy free plan
	freePlanDuration := int64(2)
	_, err := ts.TxSubscriptionBuy(consumerAddr, consumerAddr, freePlan.Index, int(freePlanDuration), false, false)
	require.Nil(t, err)
	// Verify subscription found inside getSubscription
	getSubscriptionAndFailTestIfNotFound(t, ts, consumerAddr)
	consumerBalance -= freePlan.Price.Amount.MulRaw(freePlanDuration).Int64()
	// Make sure that creator1 paid for the subscription
	require.Equal(t, consumerBalance, ts.GetBalance(consumerAcc.Addr))

	// Creator1 buys a future subscription
	newSubDuration := uint64(1)
	_, err = ts.TxSubscriptionBuy(creator1Addr, consumerAddr, premiumPlan.Index, int(newSubDuration), false, true)
	require.Nil(t, err)

	// Make sure that creator1 paid for the new subscription
	creator1Balance -= premiumPlan.Price.Amount.MulRaw(int64(newSubDuration)).Int64()
	require.Equal(t, creator1Balance, ts.GetBalance(creator1Acc.Addr))

	// Trigger new subscription
	ts.AdvanceMonths(int(freePlanDuration)).AdvanceEpoch()

	// Make sure that creator1 is now the creator of the subscription
	sub := getSubscriptionAndFailTestIfNotFound(t, ts, consumerAddr)
	require.Nil(t, sub.FutureSubscription)
	require.Equal(t, creator1Addr, sub.Creator)

	// Creator2 buys a future subscription
	_, err = ts.TxSubscriptionBuy(creator2Addr, consumerAddr, premiumPlan.Index, int(newSubDuration), false, true)
	require.Nil(t, err)

	// Make sure that creator2 paid for the new subscription
	creator2Balance -= premiumPlan.Price.Amount.MulRaw(int64(newSubDuration)).Int64()
	require.Equal(t, creator2Balance, ts.GetBalance(creator2Acc.Addr))

	// Trigger new subscription
	ts.AdvanceMonths(int(newSubDuration)).AdvanceEpoch()

	// Make sure that creator2 is now the creator of the subscription
	sub = getSubscriptionAndFailTestIfNotFound(t, ts, consumerAddr)
	require.Nil(t, sub.FutureSubscription)
	require.Equal(t, creator2Addr, sub.Creator)

	// Original consumer buys a future subscription
	_, err = ts.TxSubscriptionBuy(consumerAddr, consumerAddr, premiumPlan.Index, int(newSubDuration), false, true)
	require.Nil(t, err)

	// Make sure that consumer paid for the new subscription
	consumerBalance -= premiumPlan.Price.Amount.MulRaw(int64(newSubDuration)).Int64()
	require.Equal(t, consumerBalance, ts.GetBalance(consumerAcc.Addr))

	// Trigger new subscription
	ts.AdvanceMonths(int(newSubDuration)).AdvanceEpoch()

	// Make sure that consumer is now the creator of the subscription
	sub = getSubscriptionAndFailTestIfNotFound(t, ts, consumerAddr)
	require.Nil(t, sub.FutureSubscription)
	require.Equal(t, consumerAddr, sub.Creator)
}
