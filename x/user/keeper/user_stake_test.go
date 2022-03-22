package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/user/keeper"
	"github.com/lavanet/lava/x/user/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNUserStake(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.UserStake {
	items := make([]types.UserStake, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetUserStake(ctx, items[i])
	}
	return items
}

func TestUserStakeGet(t *testing.T) {
	keeper, ctx := keepertest.UserKeeper(t)
	items := createNUserStake(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetUserStake(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestUserStakeRemove(t *testing.T) {
	keeper, ctx := keepertest.UserKeeper(t)
	items := createNUserStake(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveUserStake(ctx,
			item.Index,
		)
		_, found := keeper.GetUserStake(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestUserStakeGetAll(t *testing.T) {
	keeper, ctx := keepertest.UserKeeper(t)
	items := createNUserStake(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllUserStake(ctx)),
	)
}
