package keeper_test

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	commontypes "github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/testutil/common"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/utils/sigs"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
	projectstypes "github.com/lavanet/lava/x/projects/types"
	subscriptionkeeper "github.com/lavanet/lava/x/subscription/keeper"
	"github.com/lavanet/lava/x/subscription/types"
	"github.com/stretchr/testify/require"
)

func createNPlans(ts *testStruct, n int) []planstypes.Plan {
	keeper := &ts.keepers.Plans

	items := make([]planstypes.Plan, n)
	plan := common.CreateMockPlan()

	for i := range items {
		items[i] = plan
		items[i].Index += strconv.Itoa(i + 1)
		keeper.AddPlan(ts.ctx, items[i])
	}

	return items
}

type testStruct struct {
	_ctx    context.Context
	ctx     sdk.Context
	keepers *keepertest.Keepers
	servers *keepertest.Servers
	plans   []planstypes.Plan
}

func setupTestStruct(t *testing.T, numPlans int) *testStruct {
	servers, keepers, _ctx := keepertest.InitAllKeepers(t)

	_ctx = keepertest.AdvanceEpoch(_ctx, keepers)
	ctx := sdk.UnwrapSDKContext(_ctx)

	ts := testStruct{
		_ctx:    _ctx,
		ctx:     ctx,
		keepers: keepers,
		servers: servers,
	}

	ts.plans = createNPlans(&ts, numPlans)

	return &ts
}

func (ts *testStruct) blockHeight() uint64 {
	return uint64(ts.ctx.BlockHeight())
}

func (ts *testStruct) advanceBlock(delta ...time.Duration) {
	ts._ctx = keepertest.AdvanceBlock(ts._ctx, ts.keepers, delta...)
	ts.ctx = sdk.UnwrapSDKContext(ts._ctx)
}

func (ts *testStruct) advanceEpoch(delta ...time.Duration) {
	ts._ctx = keepertest.AdvanceEpoch(ts._ctx, ts.keepers, delta...)
	ts.ctx = sdk.UnwrapSDKContext(ts._ctx)
}

func (ts *testStruct) advanceUntilStale(delta ...time.Duration) {
	for i := 0; i < int(commontypes.STALE_ENTRY_TIME); i += 1 {
		ts.advanceBlock()
	}
}

// advance block time by given months (minus 1 second) from a given time
// (make sure to trigger begin-epoch for each such month)
func (ts *testStruct) advanceMonths(from time.Time, months int) {
	for next := from; months > 0; months -= 1 {
		next = subscriptionkeeper.NextMonth(next)
		delta := next.Sub(ts.ctx.BlockTime()) - time.Second
		ts.advanceBlock(delta)
		ts.advanceEpoch()
	}
}

func TestCreateSubscription(t *testing.T) {
	ts := setupTestStruct(t, 3)
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

	// delete one plan, and advance to next epoch to take effect
	err := ts.keepers.Plans.DelPlan(ts.ctx, ts.plans[2].Index)
	require.Nil(t, err)
	ts._ctx = keepertest.AdvanceEpoch(ts._ctx, ts.keepers)
	ts.ctx = sdk.UnwrapSDKContext(ts._ctx)

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
			index:     ts.plans[0].Index,
			creator:   0,
			consumers: []int{0, 1},
			duration:  1,
			success:   true,
		},
		{
			name:      "invalid creator",
			index:     ts.plans[0].Index,
			creator:   2,
			consumers: []int{2},
			duration:  1,
			success:   false,
		},
		{
			name:      "invalid consumer",
			index:     ts.plans[0].Index,
			creator:   0,
			consumers: []int{3},
			duration:  1,
			success:   false,
		},
		{
			name:      "duration too long",
			index:     ts.plans[0].Index,
			creator:   0,
			consumers: []int{2},
			duration:  13,
			success:   false,
		},
		{
			name:      "insufficient funds",
			index:     ts.plans[0].Index,
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
			index:     ts.plans[2].Index,
			creator:   0,
			consumers: []int{2},
			duration:  1,
			success:   false,
		},
		{
			name:      "double subscription",
			index:     ts.plans[1].Index,
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
					ts.ctx, sub.Creator, sub.Consumer, sub.PlanIndex, tt.duration)
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

func TestSubscriptionExpiration(t *testing.T) {
	ts := setupTestStruct(t, 1)
	keeper := ts.keepers.Subscription

	account := common.CreateNewAccount(ts._ctx, *ts.keepers, 10000)
	creator := account.Addr.String()

	blocksToSave, err := ts.keepers.Epochstorage.BlocksToSave(ts.ctx, uint64(ts.ctx.BlockHeight()))
	require.Nil(t, err)

	// fill memory
	for uint64(ts.ctx.BlockHeight()) < blocksToSave {
		ts.advanceBlock()
	}

	err = keeper.CreateSubscription(ts.ctx, creator, creator, ts.plans[0].Index, 1)
	require.Nil(t, err)

	_, found := keeper.GetSubscription(ts.ctx, creator)
	require.True(t, found)

	timestamp := ts.ctx.BlockTime()

	ts.advanceMonths(timestamp, 1)
	_, found = keeper.GetSubscription(ts.ctx, creator)
	require.False(t, found)
}

func TestRenewSubscription(t *testing.T) {
	ts := setupTestStruct(t, 1)
	keeper := ts.keepers.Subscription

	account := common.CreateNewAccount(ts._ctx, *ts.keepers, 10000)
	creator := account.Addr.String()

	blocksToSave, err := ts.keepers.Epochstorage.BlocksToSave(ts.ctx, uint64(ts.ctx.BlockHeight()))
	require.Nil(t, err)

	// fill memory
	for uint64(ts.ctx.BlockHeight()) < blocksToSave {
		ts.advanceBlock()
	}

	err = keeper.CreateSubscription(ts.ctx, creator, creator, ts.plans[0].Index, 6)
	require.Nil(t, err)

	timestamp := ts.ctx.BlockTime()

	sub, found := keeper.GetSubscription(ts.ctx, creator)
	require.True(t, found)

	// fast-forward three months
	ts.advanceMonths(timestamp, 3)
	sub, found = keeper.GetSubscription(ts.ctx, creator)
	require.True(t, found)
	require.Equal(t, uint64(3), sub.DurationLeft)

	// with 3 months duration left, asking for 12 more should fail
	err = keeper.CreateSubscription(ts.ctx, creator, creator, ts.plans[0].Index, 12)
	require.NotNil(t, err)

	// but asking for additional 9 months (10 would also be fine (the extra month extension below))
	err = keeper.CreateSubscription(ts.ctx, creator, creator, ts.plans[0].Index, 9)
	require.Nil(t, err)

	sub, found = keeper.GetSubscription(ts.ctx, creator)
	require.True(t, found)

	require.Equal(t, uint64(12), sub.DurationLeft)
	require.Equal(t, uint64(9), sub.DurationTotal)

	// edit the subscription's plan (allow more CU)
	subPlan, found := ts.keepers.Plans.FindPlan(ts.ctx, sub.PlanIndex, sub.PlanBlock)
	require.True(t, found)
	oldPlanCuPerEpoch := subPlan.PlanPolicy.EpochCuLimit
	subPlan.PlanPolicy.EpochCuLimit += 100
	err = keepertest.SimulatePlansAddProposal(ts.ctx, ts.keepers.Plans, []planstypes.Plan{subPlan})
	require.Nil(t, err)

	// try extending the subscription (normally we could extend with 1 more month, but since the
	// subscription's plan changed, the extension should fail)
	err = keeper.CreateSubscription(ts.ctx, creator, creator, ts.plans[0].Index, 1)
	require.NotNil(t, err)
	require.Equal(t, uint64(12), sub.DurationLeft)
	require.Equal(t, uint64(9), sub.DurationTotal)

	// get the subscription's plan and make sure it uses the old plan
	subPlan, found = ts.keepers.Plans.FindPlan(ts.ctx, sub.PlanIndex, sub.PlanBlock)
	require.True(t, found)
	require.Equal(t, oldPlanCuPerEpoch, subPlan.PlanPolicy.EpochCuLimit)

	// delete the plan, and try to renew the subscription again
	err = ts.keepers.Plans.DelPlan(ts.ctx, ts.plans[0].Index)
	require.Nil(t, err)
	ts._ctx = keepertest.AdvanceEpoch(ts._ctx, ts.keepers)
	ts.ctx = sdk.UnwrapSDKContext(ts._ctx)

	// fast-forward another month, renewal should fail
	ts.advanceMonths(timestamp, 1)
	_, found = keeper.GetSubscription(ts.ctx, creator)
	require.True(t, found)
	err = keeper.CreateSubscription(ts.ctx, creator, creator, ts.plans[0].Index, 10)
	require.NotNil(t, err)
}

func TestSubscriptionAdminProject(t *testing.T) {
	ts := setupTestStruct(t, 1)
	keeper := ts.keepers.Subscription

	account := common.CreateNewAccount(ts._ctx, *ts.keepers, 10000)
	creator := account.Addr.String()

	err := keeper.CreateSubscription(ts.ctx, creator, creator, ts.plans[0].Index, 1)
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

	blocksToSave, err := ts.keepers.Epochstorage.BlocksToSave(ts.ctx, uint64(ts.ctx.BlockHeight()))
	require.Nil(t, err)

	// fill memory
	for uint64(ts.ctx.BlockHeight()) < blocksToSave {
		ts.advanceBlock()
	}

	err = keeper.CreateSubscription(ts.ctx, creator, creator, ts.plans[0].Index, 3)
	require.Nil(t, err)

	timestamp := ts.ctx.BlockTime()

	// add another project under the subcscription
	projectData := projectstypes.ProjectData{
		Name:    "another_project",
		Enabled: true,
		ProjectKeys: []projectstypes.ProjectKey{
			projectstypes.ProjectDeveloperKey(anotherAccount.Addr.String()),
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
			err = keeper.ChargeComputeUnitsToSubscription(ts.ctx, tt.subscription, block1, tt.usedCuPerProject)
			require.Nil(t, err)

			sub, found := keeper.GetSubscription(ts.ctx, tt.subscription)
			require.True(t, found)

			// verify the CU charge of the subscription is updated correctly
			// it depends on the iteration index since the same subscription is charged for all projects
			require.Equal(t, sub.MonthCuLeft, sub.MonthCuTotal-tt.usedCuPerProject)
			proj, err := projectKeeper.GetProjectForDeveloper(ts.ctx, tt.developer, block1)
			require.Nil(t, err)

			// charge the project
			err = projectKeeper.ChargeComputeUnitsToProject(ts.ctx, proj, block1, tt.usedCuPerProject)
			require.Nil(t, err)

			// verify that project used the CU
			proj, err = projectKeeper.GetProjectForDeveloper(ts.ctx, tt.developer, block1)
			require.Nil(t, err)
			require.Equal(t, tt.usedCuPerProject, proj.UsedCu)

			block2 := uint64(ts.ctx.BlockHeight())

			// force fixation entry (by adding project key)
			admAddr := common.CreateNewAccount(ts._ctx, *ts.keepers, 10000).Addr.String()
			projKey := []projectstypes.ProjectKey{projectstypes.ProjectAdminKey(admAddr)}
			projectKeeper.AddKeysToProject(ts.ctx, projectstypes.ADMIN_PROJECT_NAME, tt.developer, projKey)

			// fast-forward one month (since we expire the subscription in every iteration, it depends on the iteration number)
			ts.advanceMonths(timestamp, ti+1)
			sub, found = keeper.GetSubscription(ts.ctx, creator)
			require.True(t, found)
			require.Equal(t, sub.DurationTotal-uint64(ti+1), sub.DurationLeft)

			block3 := uint64(ts.ctx.BlockHeight())

			// check that subscription and project have renewed CUs, and that the
			// project created a snapshot for last month
			sub, found = keeper.GetSubscription(ts.ctx, tt.subscription)
			require.True(t, found)
			require.Equal(t, sub.MonthCuLeft, sub.MonthCuTotal)

			proj, err = projectKeeper.GetProjectForDeveloper(ts.ctx, tt.developer, block1)
			require.Nil(t, err)
			require.Equal(t, tt.usedCuPerProject, proj.UsedCu)
			proj, err = projectKeeper.GetProjectForDeveloper(ts.ctx, tt.developer, block2)
			require.Nil(t, err)
			require.Equal(t, tt.usedCuPerProject, proj.UsedCu)
			proj, err = projectKeeper.GetProjectForDeveloper(ts.ctx, tt.developer, block3)
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
		{[3]int{2000, 3, 1}, [3]int{2000, 4, 1}, 2},
		{[3]int{2000, 3, 1}, [3]int{2000, 4, 1}, 6},
		{[3]int{2000, 3, 1}, [3]int{2000, 4, 1}, 12},
	}

	plan := ts.plans[0]

	for _, tt := range template {
		now := time.Date(tt.now[0], time.Month(tt.now[1]), tt.now[2], 12, 0, 0, 0, time.UTC)

		t.Run(now.Format("2006-01-02"), func(t *testing.T) {
			// TODO: need new creator because Projects doesn't really delete projects
			creator := common.CreateNewAccount(ts._ctx, *ts.keepers, 10000).Addr.String()

			delta := now.Sub(ts.ctx.BlockTime())
			ts.advanceBlock(delta)

			err := keeper.CreateSubscription(ts.ctx, creator, creator, plan.Index, tt.months)
			require.Nil(t, err)

			sub, found := keeper.GetSubscription(ts.ctx, creator)
			require.True(t, found)
			require.Equal(t, tt.months, sub.DurationTotal)

			// will expire and remove
			timestamp := ts.ctx.BlockTime()
			ts.advanceMonths(timestamp, int(tt.months))
			ts.advanceUntilStale()
		})
	}
}

func TestSubscriptionExpire(t *testing.T) {
	ts := setupTestStruct(t, 1)
	keeper := ts.keepers.Subscription

	_, account := sigs.GenerateFloatingKey()
	coins := sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(10000)))
	ts.keepers.BankKeeper.SetBalance(ts.ctx, account, coins)

	creator := account.String()
	consumer := account.String()

	blocksToSave, err := ts.keepers.Epochstorage.BlocksToSave(ts.ctx, uint64(ts.ctx.BlockHeight()))
	require.Nil(t, err)

	err = keeper.CreateSubscription(ts.ctx, creator, consumer, "mockPlan1", 1)
	require.Nil(t, err)

	block := uint64(ts.ctx.BlockHeight())
	timestamp := ts.ctx.BlockTime()

	// fill memory
	for uint64(ts.ctx.BlockHeight()) < blocksToSave {
		ts.advanceBlock()
	}

	_, found := keeper.GetSubscription(ts.ctx, account.String())
	require.True(t, found)

	err = keeper.ChargeComputeUnitsToSubscription(ts.ctx, account.String(), block, 10)
	require.Nil(t, err)

	// fast-forward one month
	ts.advanceMonths(timestamp, 1)
	_, found = keeper.GetSubscription(ts.ctx, creator)
	require.False(t, found)

	// subscription no longer searchable, but can still charge for previous usage
	_, found = keeper.GetSubscription(ts.ctx, account.String())
	require.False(t, found)

	err = keeper.ChargeComputeUnitsToSubscription(ts.ctx, account.String(), block, 10)
	require.Nil(t, err)

	ts.advanceUntilStale()

	// subscription no longer charge-able for previous usage
	err = keeper.ChargeComputeUnitsToSubscription(ts.ctx, account.String(), block, 10)
	require.NotNil(t, err)
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

			err := keeper.CreateSubscription(ts.ctx, creator, creator, plan.Index, tt.duration)
			require.Nil(t, err)

			_, found := keeper.GetSubscription(ts.ctx, creator)
			require.True(t, found)

			balance := ts.keepers.BankKeeper.GetBalance(ts.ctx, address, epochstoragetypes.TokenDenom)
			require.Equal(t, balance.Amount.Int64(), 10000-tt.cost)

			// will expire and remove
			timestamp := ts.ctx.BlockTime()
			ts.advanceMonths(timestamp, int(tt.duration))
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

	err := keeper.CreateSubscription(ts.ctx, subPayerAddr, consumerAddr, plan.Index, 1)
	require.Nil(t, err)

	defaultProjectName := projectstypes.ADMIN_PROJECT_NAME
	longProjectName := strings.Repeat(defaultProjectName, projectstypes.MAX_PROJECT_NAME_LEN)
	invalidProjectName := "project_name,"

	template := []struct {
		name         string
		subscription string
		anotherAdmin string
		projectName  string
		success      bool
	}{
		{"project admin = regular account", consumerAddr, regularAccountAddr, "test1", true},
		{"project admin = subscription payer account", consumerAddr, subPayerAddr, "test2", true},
		{"bad subscription account (regular account)", regularAccountAddr, consumerAddr, "test4", false},
		{"bad subscription account (subscription payer account)", subPayerAddr, consumerAddr, "test5", false},
		{"bad projectName (duplicate)", consumerAddr, regularAccountAddr, defaultProjectName, false},
		{"bad projectName (too long)", consumerAddr, regularAccountAddr, longProjectName, false},
		{"bad projectName (contains comma)", consumerAddr, regularAccountAddr, invalidProjectName, false},
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
			err = keeper.AddProjectToSubscription(ts.ctx, tt.subscription, projectData)
			if tt.success {
				require.Nil(t, err)
				ts.advanceEpoch()
				proj, err := ts.keepers.Projects.GetProjectForBlock(ts.ctx, projectstypes.ProjectIndex(tt.subscription, tt.projectName), uint64(ts.ctx.BlockHeight()))
				require.Nil(t, err)
				require.Equal(t, tt.subscription, proj.Subscription)
			} else {
				require.NotNil(t, err)
			}
		})
	}
}

func TestGetProjectsForSubscription(t *testing.T) {
	ts := setupTestStruct(t, 1)
	plan := ts.plans[0]

	subAcc1 := common.CreateNewAccount(ts._ctx, *ts.keepers, 10000).Addr.String()
	subAcc2 := common.CreateNewAccount(ts._ctx, *ts.keepers, 10000).Addr.String()

	// buy two subscriptions
	_, err := ts.servers.SubscriptionServer.Buy(ts._ctx, &types.MsgBuy{
		Creator:  subAcc1,
		Consumer: subAcc1,
		Index:    plan.Index,
		Duration: 1,
	})
	require.Nil(t, err)

	_, err = ts.servers.SubscriptionServer.Buy(ts._ctx, &types.MsgBuy{
		Creator:  subAcc2,
		Consumer: subAcc2,
		Index:    plan.Index,
		Duration: 1,
	})
	require.Nil(t, err)

	// add two projects to the first subscription
	projData1 := projectstypes.ProjectData{
		Name:    "proj1",
		Enabled: true,
		Policy:  &plan.PlanPolicy,
	}
	_, err = ts.servers.SubscriptionServer.AddProject(ts._ctx, &types.MsgAddProject{
		Creator:     subAcc1,
		ProjectData: projData1,
	})
	require.Nil(t, err)

	projData2 := projectstypes.ProjectData{
		Name:    "proj2",
		Enabled: false,
		Policy:  &plan.PlanPolicy,
	}
	_, err = ts.servers.SubscriptionServer.AddProject(ts._ctx, &types.MsgAddProject{
		Creator:     subAcc1,
		ProjectData: projData2,
	})
	require.Nil(t, err)

	res1, err := ts.keepers.Subscription.ListProjects(ts._ctx, &types.QueryListProjectsRequest{
		Subscription: subAcc1,
	})
	require.Nil(t, err)

	res2, err := ts.keepers.Subscription.ListProjects(ts._ctx, &types.QueryListProjectsRequest{
		Subscription: subAcc2,
	})
	require.Nil(t, err)

	// number of projects expected (+1 because there's an auto-generated admin project)
	require.Equal(t, 3, len(res1.Projects))
	require.Equal(t, 1, len(res2.Projects))

	_, err = ts.servers.SubscriptionServer.DelProject(ts._ctx, &types.MsgDelProject{
		Creator: subAcc1,
		Name:    projData2.Name,
	})
	require.Nil(t, err)
}

func TestAddDelProjectForSubscription(t *testing.T) {
	ts := setupTestStruct(t, 1)
	plan := ts.plans[0]

	subAcc := common.CreateNewAccount(ts._ctx, *ts.keepers, 10000).Addr.String()

	// buy subscription
	_, err := ts.servers.SubscriptionServer.Buy(ts._ctx, &types.MsgBuy{
		Creator:  subAcc,
		Consumer: subAcc,
		Index:    plan.Index,
		Duration: 1,
	})
	require.Nil(t, err)

	// add project to the subscription
	projData := projectstypes.ProjectData{
		Name:    "proj",
		Enabled: true,
		Policy:  &plan.PlanPolicy,
	}
	_, err = ts.servers.SubscriptionServer.AddProject(ts._ctx, &types.MsgAddProject{
		Creator:     subAcc,
		ProjectData: projData,
	})
	require.Nil(t, err)

	ts.advanceEpoch()

	res, err := ts.keepers.Subscription.ListProjects(ts._ctx, &types.QueryListProjectsRequest{
		Subscription: subAcc,
	})
	require.Nil(t, err)
	require.Equal(t, 2, len(res.Projects))

	// del project to the subscription
	_, err = ts.servers.SubscriptionServer.DelProject(ts._ctx, &types.MsgDelProject{
		Creator: subAcc,
		Name:    projData.Name,
	})
	require.Nil(t, err)

	ts.advanceEpoch()

	res, err = ts.keepers.Subscription.ListProjects(ts._ctx, &types.QueryListProjectsRequest{
		Subscription: subAcc,
	})
	require.Nil(t, err)
	require.Equal(t, 1, len(res.Projects))
}
