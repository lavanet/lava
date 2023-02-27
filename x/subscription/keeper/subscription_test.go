package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/testutil/nullify"
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
		items[i].Block = ctx.BlockHeight()
		items[i].PlanIndex = "testplan"
		items[i].PlanBlock = ctx.BlockHeight()

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
