package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/dualstaking/keeper"
	"github.com/lavanet/lava/x/dualstaking/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNDelegatorReward(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.DelegatorReward {
	items := make([]types.DelegatorReward, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetDelegatorReward(ctx, items[i])
	}
	return items
}

func TestDelegatorRewardGet(t *testing.T) {
	keeper, ctx := keepertest.DualstakingKeeper(t)
	items := createNDelegatorReward(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetDelegatorReward(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}

func TestDelegatorRewardRemove(t *testing.T) {
	keeper, ctx := keepertest.DualstakingKeeper(t)
	items := createNDelegatorReward(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveDelegatorReward(ctx,
			item.Index,
		)
		_, found := keeper.GetDelegatorReward(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestDelegatorRewardGetAll(t *testing.T) {
	keeper, ctx := keepertest.DualstakingKeeper(t)
	items := createNDelegatorReward(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllDelegatorReward(ctx)),
	)
}
