package keeper_test

import (
	"strconv"
	"testing"

	"cosmossdk.io/math"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/rewards/keeper"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createTestMinIprpcCost(keeper *keeper.Keeper, ctx sdk.Context) sdk.Coin {
	item := sdk.NewCoin("denom", math.OneInt())
	keeper.SetMinIprpcCost(ctx, item)
	return item
}

func TestMinIprpcCostGet(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	item := createTestMinIprpcCost(keeper, ctx)
	rst := keeper.GetMinIprpcCost(ctx)
	require.Equal(t,
		nullify.Fill(&item),
		nullify.Fill(&rst),
	)
}

func createNIprpcSubscriptions(keeper *keeper.Keeper, ctx sdk.Context, n int) []string {
	items := make([]string, n)
	for i := range items {
		items[i] = strconv.Itoa(i)
		keeper.SetIprpcSubscription(ctx, items[i])
	}
	return items
}

func TestIprpcSubscriptionsGet(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNIprpcSubscriptions(keeper, ctx, 10)
	for _, item := range items {
		require.True(t, keeper.IsIprpcSubscription(ctx, item))
	}
}

func TestIprpcSubscriptionsRemove(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNIprpcSubscriptions(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveIprpcSubscription(ctx, item)
		require.False(t, keeper.IsIprpcSubscription(ctx, item))
	}
}

func TestIprpcSubscriptionsGetAll(t *testing.T) {
	keeper, ctx := keepertest.RewardsKeeper(t)
	items := createNIprpcSubscriptions(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllIprpcSubscription(ctx)),
	)
}
