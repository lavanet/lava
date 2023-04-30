package keeper_test

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/utils/sigs"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	planskeeper "github.com/lavanet/lava/x/plans/keeper"
	planstypes "github.com/lavanet/lava/x/plans/types"
	projectstypes "github.com/lavanet/lava/x/projects/types"
	"github.com/lavanet/lava/x/subscription/keeper"
	"github.com/lavanet/lava/x/subscription/types"
	"github.com/stretchr/testify/require"
)

func createNSubscription(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Subscription {
	items := make([]types.Subscription, n)
	_, creator := sigs.GenerateFloatingKey()

	for i := range items {
		items[i].Creator = creator.String()
		items[i].Consumer = "consumer-" + strconv.Itoa(i)
		items[i].Block = uint64(ctx.BlockHeight())
		items[i].PlanIndex = "testplan"
		items[i].PlanBlock = uint64(ctx.BlockHeight())

		keeper.SetSubscription(ctx, items[i])
	}
	return items
}

func createNPlans(keeper *planskeeper.Keeper, ctx sdk.Context, n int) []planstypes.Plan {
	items := make([]planstypes.Plan, n)
	plan := common.CreateMockPlan()

	for i := range items {
		items[i] = plan
		items[i].Index += strconv.Itoa(i + 1)
		keeper.AddPlan(ctx, items[i])
	}

	return items
}

func TestSubscriptionGet(t *testing.T) {
	keeper, ctx := keepertest.SubscriptionKeeper(t)
	items := createNSubscription(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetSubscription(ctx,
			item.Consumer,
		)
		require.True(t, found)

		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}

func TestSubscriptionRemove(t *testing.T) {
	keeper, ctx := keepertest.SubscriptionKeeper(t)
	items := createNSubscription(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveSubscription(ctx,
			item.Creator,
		)
		_, found := keeper.GetSubscription(ctx,
			item.Creator,
		)
		require.False(t, found)
	}
}

func TestSubscriptionGetAll(t *testing.T) {
	keeper, ctx := keepertest.SubscriptionKeeper(t)
	items := createNSubscription(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllSubscription(ctx)),
	)
}

type testStruct struct {
	_ctx    context.Context
	ctx     sdk.Context
	keepers *keepertest.Keepers
	plans   []planstypes.Plan
}

func (ts *testStruct) advanceBlock(delta ...time.Duration) {
	ts._ctx = keepertest.AdvanceBlock(ts._ctx, ts.keepers, delta...)
	ts.ctx = sdk.UnwrapSDKContext(ts._ctx)
}

func (ts *testStruct) expireSubscription(sub types.Subscription) types.Subscription {
	keeper := ts.keepers.Subscription

	// expedite: change expiration time to 1 second ago
	sub.MonthExpiryTime = uint64(ts.ctx.BlockTime().Add(-time.Second).UTC().Unix())
	keeper.SetSubscription(ts.ctx, sub)

	// trigger EpochStart() processing
	ts._ctx = keepertest.AdvanceEpoch(ts._ctx, ts.keepers)
	ts.ctx = sdk.UnwrapSDKContext(ts._ctx)
	keeper.EpochStart(ts.ctx)

	// might not be found - that's ok
	sub, _ = keeper.GetSubscription(ts.ctx, sub.Consumer)
	return sub
}

func setupTestStruct(t *testing.T, numPlans int) testStruct {
	_, keepers, _ctx := keepertest.InitAllKeepers(t)

	_ctx = keepertest.AdvanceEpoch(_ctx, keepers)
	ctx := sdk.UnwrapSDKContext(_ctx)

	plans := createNPlans(&keepers.Plans, ctx, numPlans)

	ts := testStruct{
		_ctx:    _ctx,
		ctx:     ctx,
		keepers: keepers,
		plans:   plans,
	}

	return ts
}

func TestCreateSubscription(t *testing.T) {
	ts := setupTestStruct(t, 2)
	keeper := ts.keepers.Subscription

	creators := []struct {
		address string
		amount  int64
	}{
		{
			address: "FILL",
			amount:  100000,
		},
		{
			address: "FILL",
			amount:  1,
		},
		{
			address: "invalid creator",
			amount:  0,
		},
	}

	for i := range creators {
		if creators[i].address == "FILL" {
			account := common.CreateNewAccount(ts._ctx, *ts.keepers, creators[i].amount)
			creators[i].address = account.Addr.String()
		}
	}

	consumers := make([]string, 4)
	for i := range consumers {
		account := common.CreateNewAccount(ts._ctx, *ts.keepers, 1)
		consumers[i] = account.Addr.String()
	}
	consumers[3] = "invalid consumer"

	template := []struct {
		name      string
		index     string
		creator   int
		consumers []int
		duration  uint64
		success   bool
	}{
		{
			name:      "create subscriptions",
			index:     "mockPlan1",
			creator:   0,
			consumers: []int{0, 1},
			duration:  1,
			success:   true,
		},
		{
			name:      "invalid creator",
			index:     "mockPlan1",
			creator:   2,
			consumers: []int{2},
			duration:  1,
			success:   false,
		},
		{
			name:      "invalid consumer",
			index:     "mockPlan1",
			creator:   0,
			consumers: []int{3},
			duration:  1,
			success:   false,
		},
		{
			name:      "duration too long",
			index:     "mockPlan1",
			creator:   0,
			consumers: []int{2},
			duration:  13,
			success:   false,
		},
		{
			name:      "insufficient funds",
			index:     "mockPlan1",
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
			name:      "double subscription",
			index:     "mockPlan2",
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
					Creator:   creators[tt.creator].address,
					Consumer:  consumers[consumer],
					PlanIndex: tt.index,
				}

				err := keeper.CreateSubscription(
					ts.ctx, sub.Creator, sub.Consumer, sub.PlanIndex, tt.duration, "")
				if tt.success {
					require.Nil(t, err, tt.name)
					_, found := keeper.GetSubscription(ts.ctx, sub.Consumer)
					require.True(t, found, tt.name)
				} else {
					require.NotNil(t, err, tt.name)
				}
			})
		}
	}
}

func TestRenewSubscription(t *testing.T) {
	ts := setupTestStruct(t, 1)
	keeper := ts.keepers.Subscription

	account := common.CreateNewAccount(ts._ctx, *ts.keepers, 10000)
	creator := account.Addr.String()

	err := keeper.CreateSubscription(ts.ctx, creator, creator, ts.plans[0].Index, 6, "")
	require.Nil(t, err)

	sub, found := keeper.GetSubscription(ts.ctx, creator)
	require.True(t, found)

	// fast-forward three months
	sub = ts.expireSubscription(sub)
	sub = ts.expireSubscription(sub)
	sub = ts.expireSubscription(sub)
	require.Equal(t, uint64(3), sub.DurationLeft)

	// with 3 months duration left, asking for 12 more should fail
	err = keeper.CreateSubscription(ts.ctx, creator, creator, ts.plans[0].Index, 12, "")
	require.NotNil(t, err)

	// but asking for additional 9 months (10 would also be fine)
	err = keeper.CreateSubscription(ts.ctx, creator, creator, ts.plans[0].Index, 9, "")
	require.Nil(t, err)

	sub, found = keeper.GetSubscription(ts.ctx, creator)
	require.True(t, found)

	require.Equal(t, uint64(12), sub.DurationLeft)
	require.Equal(t, uint64(9), sub.DurationTotal)

	// edit the subscription's plan
	subPlan, found := ts.keepers.Plans.FindPlan(ts.ctx, sub.PlanIndex, sub.PlanBlock)
	require.True(t, found)
	oldPlanGeoLocation := subPlan.PlanPolicy.GeolocationProfile
	subPlan.PlanPolicy.GeolocationProfile += 1
	err = ts.keepers.Plans.AddPlan(ts.ctx, subPlan)
	require.Nil(t, err)

	// try extending the subscription (normally we could extend with 1 more month, but since the
	// subscription's plan changed, the extension should fail)
	err = keeper.CreateSubscription(ts.ctx, creator, creator, ts.plans[0].Index, 1, "")
	require.NotNil(t, err)
	require.Equal(t, uint64(12), sub.DurationLeft)
	require.Equal(t, uint64(9), sub.DurationTotal)

	// get the subscription's plan and make sure it uses the old plan
	subPlan, found = ts.keepers.Plans.FindPlan(ts.ctx, sub.PlanIndex, sub.PlanBlock)
	require.True(t, found)
	require.Equal(t, oldPlanGeoLocation, subPlan.PlanPolicy.GeolocationProfile)
}

func TestSubscriptionAdminProject(t *testing.T) {
	ts := setupTestStruct(t, 1)
	keeper := ts.keepers.Subscription

	account := common.CreateNewAccount(ts._ctx, *ts.keepers, 10000)
	creator := account.Addr.String()

	err := keeper.CreateSubscription(ts.ctx, creator, creator, "mockPlan1", 1, "")
	require.Nil(t, err)

	block := uint64(ts.ctx.BlockHeight())

	// a newly created subscription is expected to have one default project,
	// with the subscription address as its developer key
	_, err = ts.keepers.Projects.GetProjectDeveloperData(ts.ctx, creator, block)
	require.Nil(t, err)
}

func TestMonthlyRechargeCU(t *testing.T) {
	ts := setupTestStruct(t, 1)
	keeper := ts.keepers.Subscription
	projectKeeper := ts.keepers.Projects

	account := common.CreateNewAccount(ts._ctx, *ts.keepers, 10000)
	anotherAccount := common.CreateNewAccount(ts._ctx, *ts.keepers, 10000)
	creator := account.Addr.String()

	err := keeper.CreateSubscription(ts.ctx, creator, creator, ts.plans[0].Index, 3, "")
	require.Nil(t, err)

	// add another project under the subcscription
	projectData := projectstypes.ProjectData{
		Name:        "anotherProject",
		Description: "dummyDesc",
		Enabled:     true,
		ProjectKeys: []projectstypes.ProjectKey{
			{
				Key:   anotherAccount.Addr.String(),
				Types: []projectstypes.ProjectKey_KEY_TYPE{projectstypes.ProjectKey_DEVELOPER},
				Vrfpk: "",
			},
		},
		Policy: &projectstypes.Policy{
			GeolocationProfile: uint64(1),
			TotalCuLimit:       1000,
			EpochCuLimit:       100,
			MaxProvidersToPair: 3,
		},
	}
	err = keeper.AddProjectToSubscription(ts.ctx, creator, projectData)
	require.Nil(t, err)

	// we'll test both the default project and the second project, which differ in their developers
	template := []struct {
		name             string
		subscription     string
		developer        string
		usedCuPerProject uint64 // total CU in sub is 1000 -- let each project use 500
	}{
		{"default project", creator, creator, 500},
		{"second project (non-default)", creator, anotherAccount.Addr.String(), 500},
	}
	for ti, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			block1 := uint64(ts.ctx.BlockHeight())

			ts._ctx = keepertest.AdvanceEpoch(ts._ctx, ts.keepers)
			ts.ctx = sdk.UnwrapSDKContext(ts._ctx)

			// charge the subscription
			err = keeper.ChargeComputeUnitsToSubscription(ts.ctx, tt.subscription, tt.usedCuPerProject)
			require.Nil(t, err)

			sub, found := keeper.GetSubscription(ts.ctx, tt.subscription)
			require.True(t, found)

			// verify the CU charge of the subscription is updated correctly
			// it depends on the iteration index since the same subscription is charged for all projects
			require.Equal(t, sub.MonthCuLeft, sub.MonthCuTotal-tt.usedCuPerProject)
			proj, _, err := projectKeeper.GetProjectForDeveloper(ts.ctx, tt.developer, block1)
			require.Nil(t, err)

			// charge the project
			err = projectKeeper.ChargeComputeUnitsToProject(ts.ctx, proj, tt.usedCuPerProject)
			require.Nil(t, err)

			// verify that project used the CU
			proj, _, err = projectKeeper.GetProjectForDeveloper(ts.ctx, tt.developer, block1)
			require.Nil(t, err)
			require.Equal(t, tt.usedCuPerProject, proj.UsedCu)

			block2 := uint64(ts.ctx.BlockHeight())

			// force fixation entry (by adding project key)
			projKey := []projectstypes.ProjectKey{
				{
					Key:   common.CreateNewAccount(ts._ctx, *ts.keepers, 10000).Addr.String(),
					Types: []projectstypes.ProjectKey_KEY_TYPE{projectstypes.ProjectKey_ADMIN},
				},
			}
			projectKeeper.AddKeysToProject(ts.ctx, projectstypes.ADMIN_PROJECT_NAME, tt.developer, projKey)

			// fast-forward one month (since we expire the subscription in every iteration, it depends on the iteration number)
			sub = ts.expireSubscription(sub)
			require.Equal(t, sub.DurationTotal-uint64(ti+1), sub.DurationLeft)

			block3 := uint64(ts.ctx.BlockHeight())

			// check that subscription and project have renewed CUs, and that the
			// project created a snapshot for last month
			sub, found = keeper.GetSubscription(ts.ctx, tt.subscription)
			require.True(t, found)
			require.Equal(t, sub.MonthCuLeft, sub.MonthCuTotal)

			proj, _, err = projectKeeper.GetProjectForDeveloper(ts.ctx, tt.developer, block1)
			require.Nil(t, err)
			require.Equal(t, tt.usedCuPerProject, proj.UsedCu)
			proj, _, err = projectKeeper.GetProjectForDeveloper(ts.ctx, tt.developer, block2)
			require.Nil(t, err)
			require.Equal(t, tt.usedCuPerProject, proj.UsedCu)
			proj, _, err = projectKeeper.GetProjectForDeveloper(ts.ctx, tt.developer, block3)
			require.Nil(t, err)
			require.Equal(t, uint64(0), proj.UsedCu)
		})
	}
}

func TestExpiryTime(t *testing.T) {
	ts := setupTestStruct(t, 1)
	keeper := ts.keepers.Subscription

	// AdvanceBlock() always uses the current time for the first block (and
	// (ignores the time delta arg if given); So call it here first to avoid
	// the call below being the first and having the delta are ignored.
	ts.advanceBlock()

	template := []struct {
		now    [3]int // year, month, day
		res    [3]int // year, month, day
		months uint64
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
		{[3]int{2000, 3, 1}, [3]int{2000, 5, 1}, 2},
		{[3]int{2000, 3, 1}, [3]int{2000, 9, 1}, 6},
		{[3]int{2000, 3, 1}, [3]int{2001, 3, 1}, 12},
	}

	plan := ts.plans[0]

	for _, tt := range template {
		now := time.Date(tt.now[0], time.Month(tt.now[1]), tt.now[2], 12, 0, 0, 0, time.UTC)
		res := time.Date(tt.res[0], time.Month(tt.res[1]), tt.res[2], 12, 0, 0, 0, time.UTC)

		t.Run(now.Format("2006-01-02"), func(t *testing.T) {
			// TODO: need new creator because Projects doesn't really delete projects
			creator := common.CreateNewAccount(ts._ctx, *ts.keepers, 10000).Addr.String()

			delta := now.Sub(ts.ctx.BlockTime())
			ts.advanceBlock(delta)

			err := keeper.CreateSubscription(ts.ctx, creator, creator, plan.Index, tt.months, "")
			require.Nil(t, err)

			sub, found := keeper.GetSubscription(ts.ctx, creator)
			require.True(t, found)
			require.Equal(t, res, time.Unix(int64(sub.MonthExpiryTime), 0).UTC())
			require.Equal(t, tt.months, sub.DurationTotal)

			keeper.RemoveSubscription(ts.ctx, creator)

			// TODO: remove when RemoveSubscriptions properly removes projects
			projectID := projectstypes.ProjectIndex(creator, projectstypes.ADMIN_PROJECT_NAME)
			ts.keepers.Projects.DeleteProject(ts.ctx, projectID)
		})
	}
}

func TestPrice(t *testing.T) {
	ts := setupTestStruct(t, 1)
	keeper := ts.keepers.Subscription

	template := []struct {
		name     string
		duration uint64
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
			// NOTE: need new creator because Projects doesn't really delete projects
			address := common.CreateNewAccount(ts._ctx, *ts.keepers, 10000).Addr
			creator := address.String()

			plan := ts.plans[0]
			plan.AnnualDiscountPercentage = tt.discount
			plan.Price = sdk.NewCoin("ulava", sdk.NewInt(tt.price))
			ts.keepers.Plans.AddPlan(ts.ctx, plan)

			err := keeper.CreateSubscription(ts.ctx, creator, creator, plan.Index, tt.duration, "")
			require.Nil(t, err)

			_, found := keeper.GetSubscription(ts.ctx, creator)
			require.True(t, found)

			balance := ts.keepers.BankKeeper.GetBalance(ts.ctx, address, epochstoragetypes.TokenDenom)
			require.Equal(t, balance.Amount.Int64(), int64(10000-tt.cost))

			keeper.RemoveSubscription(ts.ctx, creator)

			// TODO: remove when RemoveSubscriptions properly removes projects
			projectID := projectstypes.ProjectIndex(creator, projectstypes.ADMIN_PROJECT_NAME)
			ts.keepers.Projects.DeleteProject(ts.ctx, projectID)
		})
	}
}

func TestAddProjectToSubscription(t *testing.T) {
	ts := setupTestStruct(t, 1)
	keeper := ts.keepers.Subscription
	plan := ts.plans[0]

	subPayer := common.CreateNewAccount(ts._ctx, *ts.keepers, 10000)
	consumer := common.CreateNewAccount(ts._ctx, *ts.keepers, 10000)
	regularAccount := common.CreateNewAccount(ts._ctx, *ts.keepers, 10000)

	subPayerAddr := subPayer.Addr.String()
	consumerAddr := consumer.Addr.String()
	regularAccountAddr := regularAccount.Addr.String()

	err := keeper.CreateSubscription(ts.ctx, subPayerAddr, consumerAddr, plan.Index, 1, "")
	require.Nil(t, err)

	defaultProjectName := projectstypes.ADMIN_PROJECT_NAME
	longProjectName := strings.Repeat(defaultProjectName, projectstypes.MAX_PROJECT_NAME_LEN)
	projectNameWithComma := "projectName,"
	nonAsciiProjectName := "projectName¢"

	projectDescription := "test project"
	longProjectDescription := strings.Repeat(projectDescription, projectstypes.MAX_PROJECT_DESCRIPTION_LEN)
	nonAsciiProjectDescription := "projectDesc¢"

	template := []struct {
		name               string
		subscription       string
		anotherAdmin       string
		projectName        string
		projectDescription string
		success            bool
	}{
		{"project admin = regular account", consumerAddr, regularAccountAddr, "test1", projectDescription, true},
		{"project admin = subscription payer account", consumerAddr, subPayerAddr, "test2", projectDescription, true},
		{"bad subscription account (regular account)", regularAccountAddr, consumerAddr, "test4", projectDescription, false},
		{"bad subscription account (subscription payer account)", subPayerAddr, consumerAddr, "test5", projectDescription, false},
		{"bad projectName (duplicate)", consumerAddr, regularAccountAddr, defaultProjectName, projectDescription, false},
		{"bad projectName (too long)", consumerAddr, regularAccountAddr, longProjectName, projectDescription, false},
		{"bad projectName (contains comma)", consumerAddr, regularAccountAddr, projectNameWithComma, projectDescription, false},
		{"bad projectName (non ascii)", consumerAddr, regularAccountAddr, nonAsciiProjectName, projectDescription, false},
		{"bad projectName (empty)", consumerAddr, regularAccountAddr, "", projectDescription, false},
		{"bad projectDescription (too long)", consumerAddr, regularAccountAddr, "test6", longProjectDescription, false},
		{"bad projectDescription (non ascii)", consumerAddr, regularAccountAddr, "test7", nonAsciiProjectDescription, false},
	}

	for _, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			projectData := projectstypes.ProjectData{
				Name:        tt.projectName,
				Description: tt.projectDescription,
				Enabled:     true,
				ProjectKeys: []projectstypes.ProjectKey{{
					Key:   tt.anotherAdmin,
					Types: []projectstypes.ProjectKey_KEY_TYPE{projectstypes.ProjectKey_ADMIN},
					Vrfpk: "",
				}},
			}
			err = keeper.AddProjectToSubscription(ts.ctx, tt.subscription, projectData)
			if tt.success {
				require.Nil(t, err)
				proj, err := ts.keepers.Projects.GetProjectForBlock(ts.ctx, projectstypes.ProjectIndex(tt.subscription, tt.projectName), uint64(ts.ctx.BlockHeight()))
				require.Nil(t, err)
				require.Equal(t, tt.subscription, proj.Subscription)
			} else {
				require.NotNil(t, err)
			}
		})
	}
}
