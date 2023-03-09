package keeper_test

import (
	"strconv"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/testutil/common"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
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

func TestCreateSubscription(t *testing. T) {
	_, keepers, _ctx := keepertest.InitAllKeepers(t)
	ctx := sdk.UnwrapSDKContext(_ctx)

	keeper := keepers.Subscription
	plansKeeper := keepers.Plans

	plan := common.CreateMockPlan()
	plansKeeper.AddPlan(ctx, plan)

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
			account := common.CreateNewAccount(_ctx, *keepers, creators[i].amount)
			creators[i].address = account.Addr.String()
		}
	}

	consumers := make([]string, 4)
	for i := range consumers {
		account := common.CreateNewAccount(_ctx, *keepers, 1)
		consumers[i] = account.Addr.String()
	}
	consumers[3] = "invalid consumer"

	template := []struct {
		name      string
		index     string
		creator   int
		consumers []int
		success   bool
	}{
		{
			name:      "create subscriptions",
			index:     "mockPlan",
			creator:   0,
			consumers: []int{0, 1},
			success:   true,
		},
		{
			name:      "invalid creator",
			index:     "mockPlan",
			creator:   2,
			consumers: []int{2},
			success:   false,
		},
		{
			name:      "invalid consumer",
			index:     "mockPlan",
			creator:   0,
			consumers: []int{3},
			success:   false,
		},
		{
			name:      "insufficient funds",
			index:     "mockPlan",
			creator:   1,
			consumers: []int{2},
			success:   false,
		},
		{
			name:      "invalid plan",
			index:     "",
			creator:   0,
			consumers: []int{2},
			success:   false,
		},
		{
			name:      "unknown plan",
			index:     "no-such-plan",
			creator:   0,
			consumers: []int{2},
			success:   false,
		},
		{
			name:      "double subscription",
			index:     "mockPlan",
			creator:   0,
			consumers: []int{0},
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
					ctx, sub.Creator, sub.Consumer, sub.PlanIndex, 1)
				if tt.success {
					require.Nil(t, err, tt.name)
					_, found := keeper.GetSubscription(ctx, sub.Consumer)
					require.True(t, found, tt.name)
				} else {
					require.NotNil(t, err, tt.name)
				}
			})
		}
	}
}

func TestSubscriptionDefaultProject(t *testing. T) {
	_, keepers, _ctx := keepertest.InitAllKeepers(t)
	ctx := sdk.UnwrapSDKContext(_ctx)

	keeper := keepers.Subscription
	keepers.Plans.AddPlan(ctx, common.CreateMockPlan())

	account := common.CreateNewAccount(_ctx, *keepers, 10000)
	creator := account.Addr.String()

	err := keeper.CreateSubscription(ctx, creator, creator, "mockPlan", 1)
	require.Nil(t, err)

	block := uint64(ctx.BlockHeight())

	// a newly created subscription is expected to have one default project,
	// with the subscription address as its developer key
	_, err = keepers.Projects.GetProjectIDForDeveloper(ctx, creator, block)
	require.Nil(t, err)
}

func TestExpiryTime(t *testing. T) {
	_, keepers, _ctx := keepertest.InitAllKeepers(t)

	// AdvanceBlock() always users the current time for the first block (and
	// (ignores the time delta arg if given); So call it here first to avoid
	// the call below being the first and having the delta are ignored.
	_ctx = keepertest.AdvanceBlock(_ctx, keepers)
	ctx := sdk.UnwrapSDKContext(_ctx)

	keeper := keepers.Subscription
	plansKeeper := keepers.Plans
	projsKeeper := keepers.Projects

	plan := common.CreateMockPlan()
	plan.Duration = 1 // month
	plansKeeper.AddPlan(ctx, plan)

	template := []struct {
		now    [3]int // year, month, day
		res    [3]int // year, month, day
		months uint64
	}{
		// monthly
		{ [3]int{2000, 3, 1}, [3]int{2000, 4, 1}, 1 },
		{ [3]int{2000, 3, 30}, [3]int{2000, 4, 30}, 1 },
		{ [3]int{2000, 3, 31}, [3]int{2000, 4, 30}, 1 },
		{ [3]int{2000, 2, 1}, [3]int{2000, 3, 1}, 1 },
		{ [3]int{2000, 2, 28}, [3]int{2000, 3, 28}, 1 },
		{ [3]int{2001, 2, 28}, [3]int{2001, 3, 31}, 1 },
		{ [3]int{2000, 2, 29}, [3]int{2000, 3, 31}, 1 },
		{ [3]int{2000, 1, 28}, [3]int{2000, 2, 28}, 1 },
		{ [3]int{2001, 1, 28}, [3]int{2001, 2, 28}, 1 },
		{ [3]int{2000, 1, 29}, [3]int{2000, 2, 29}, 1 },
		{ [3]int{2001, 1, 29}, [3]int{2001, 2, 28}, 1 },
		{ [3]int{2000, 1, 30}, [3]int{2000, 2, 29}, 1 },
		{ [3]int{2001, 1, 30}, [3]int{2001, 2, 28}, 1 },
		{ [3]int{2000, 1, 31}, [3]int{2000, 2, 29}, 1 },
		{ [3]int{2001, 1, 31}, [3]int{2001, 2, 28}, 1 },
		{ [3]int{2001, 12, 31}, [3]int{2002, 1, 31}, 1 },
		// yearly
		{ [3]int{2000, 3, 1}, [3]int{2001, 3, 1}, 12 },
		{ [3]int{2000, 2, 28}, [3]int{2001, 2, 28}, 12 },
		{ [3]int{2000, 2, 29}, [3]int{2001, 2, 28}, 12 },
		{ [3]int{2001, 2, 28}, [3]int{2002, 2, 28}, 12 },
		{ [3]int{2003, 2, 28}, [3]int{2004, 2, 29}, 12 },
	}

	for _, tt := range template {
		now := time.Date(tt.now[0], time.Month(tt.now[1]), tt.now[2], 12, 0, 0, 0, time.UTC)
		res := time.Date(tt.res[0], time.Month(tt.res[1]), tt.res[2], 12, 0, 0, 0, time.UTC)

		t.Run(now.Format("2006-01-02"), func(t *testing.T) {
			// TODO: need new creator because Projects doesn't really delete projects
			creator := common.CreateNewAccount(_ctx, *keepers, 10000).Addr.String()

			delta := now.Sub(ctx.BlockTime())
			_ctx = keepertest.AdvanceBlock(_ctx, keepers, delta)
			ctx = sdk.UnwrapSDKContext(_ctx)

			err := keeper.CreateSubscription(ctx, creator, creator, plan.Index, tt.months)
			require.Nil(t, err)

			sub, found := keeper.GetSubscription(ctx, creator)
			require.True(t, found)
			require.Equal(t, res, time.Unix(int64(sub.ExpiryTime), 0).UTC())

			keeper.RemoveSubscription(ctx, creator)
			// TODO: remove when RemoveSubscriptions properly removes projects
			projsKeeper.DeleteProject(ctx, projectstypes.ProjectIndex(creator, "default"))
		})
	}
}

func TestYearlyPrice(t *testing. T) {
	_, keepers, _ctx := keepertest.InitAllKeepers(t)
	ctx := sdk.UnwrapSDKContext(_ctx)

	keeper := keepers.Subscription
	plansKeeper := keepers.Plans
	projsKeeper := keepers.Projects

	plan := common.CreateMockPlan()

	template := []struct {
		name     string
		duration uint64
		discount uint64
		price    int64
		cost     int64
	}{
		{ "yearly without discount", 1, 0, 100, 1200 },
		{ "yearly with discount", 1, 25, 100, 900 },
	}

	for _, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			// NOTE: need new creator because Projects doesn't really delete projects
			address := common.CreateNewAccount(_ctx, *keepers, 10000).Addr
			creator := address.String()

			plan.Duration = tt.duration
			plan.AnnualDiscountPercentage = tt.discount
			plan.Price = sdk.NewCoin("ulava", sdk.NewInt(tt.price))
			plansKeeper.AddPlan(ctx, plan)

			err := keeper.CreateSubscription(ctx, creator, creator, plan.Index, 12)
			require.Nil(t, err)

			_, found := keeper.GetSubscription(ctx, creator)
			require.True(t, found)

			balance := keepers.BankKeeper.GetBalance(ctx, address, epochstoragetypes.TokenDenom)
			require.Equal(t, balance.Amount.Int64(), int64(10000 - tt.cost))

			keeper.RemoveSubscription(ctx, creator)
			// TODO: remove when RemoveSubscriptions properly removes projects
			projsKeeper.DeleteProject(ctx, projectstypes.ProjectIndex(creator, "default"))
		})
	}
}
