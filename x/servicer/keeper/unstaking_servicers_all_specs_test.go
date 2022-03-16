package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/servicer/keeper"
	"github.com/lavanet/lava/x/servicer/types"
	"github.com/stretchr/testify/require"
)

func createNUnstakingServicersAllSpecs(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.UnstakingServicersAllSpecs {
	items := make([]types.UnstakingServicersAllSpecs, n)
	for i := range items {
		items[i].Id = keeper.AppendUnstakingServicersAllSpecs(ctx, items[i])
	}
	return items
}

func TestUnstakingServicersAllSpecsGet(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	items := createNUnstakingServicersAllSpecs(keeper, ctx, 10)
	for _, item := range items {
		got, found := keeper.GetUnstakingServicersAllSpecs(ctx, item.Id)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&got),
		)
	}
}

func TestUnstakingServicersAllSpecsRemove(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	items := createNUnstakingServicersAllSpecs(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveUnstakingServicersAllSpecs(ctx, item.Id)
		_, found := keeper.GetUnstakingServicersAllSpecs(ctx, item.Id)
		require.False(t, found)
	}
}

func TestUnstakingServicersAllSpecsGetAll(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	items := createNUnstakingServicersAllSpecs(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllUnstakingServicersAllSpecs(ctx)),
	)
}

func TestUnstakingServicersAllSpecsCount(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	items := createNUnstakingServicersAllSpecs(keeper, ctx, 10)
	count := uint64(len(items))
	require.Equal(t, count, keeper.GetUnstakingServicersAllSpecsCount(ctx))
}
