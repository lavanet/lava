package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/user/keeper"
	"github.com/lavanet/lava/x/user/types"
	"github.com/stretchr/testify/require"
)

func createNUnstakingUsersAllSpecs(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.UnstakingUsersAllSpecs {
	items := make([]types.UnstakingUsersAllSpecs, n)
	for i := range items {
		items[i].Id = keeper.AppendUnstakingUsersAllSpecs(ctx, items[i])
	}
	return items
}

func TestUnstakingUsersAllSpecsGet(t *testing.T) {
	keeper, ctx := keepertest.UserKeeper(t)
	items := createNUnstakingUsersAllSpecs(keeper, ctx, 10)
	for _, item := range items {
		got, found := keeper.GetUnstakingUsersAllSpecs(ctx, item.Id)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&got),
		)
	}
}

func TestUnstakingUsersAllSpecsRemove(t *testing.T) {
	keeper, ctx := keepertest.UserKeeper(t)
	items := createNUnstakingUsersAllSpecs(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveUnstakingUsersAllSpecs(ctx, item.Id)
		_, found := keeper.GetUnstakingUsersAllSpecs(ctx, item.Id)
		require.False(t, found)
	}
}

func TestUnstakingUsersAllSpecsGetAll(t *testing.T) {
	keeper, ctx := keepertest.UserKeeper(t)
	items := createNUnstakingUsersAllSpecs(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllUnstakingUsersAllSpecs(ctx)),
	)
}

func TestUnstakingUsersAllSpecsCount(t *testing.T) {
	keeper, ctx := keepertest.UserKeeper(t)
	items := createNUnstakingUsersAllSpecs(keeper, ctx, 10)
	count := uint64(len(items))
	require.Equal(t, count, keeper.GetUnstakingUsersAllSpecsCount(ctx))
}
