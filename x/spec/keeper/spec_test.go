package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/spec/keeper"
	"github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

func createNSpec(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Spec {
	items := make([]types.Spec, n)
	for i := range items {
		items[i].Id = keeper.AppendSpec(ctx, items[i])
	}
	return items
}

func TestSpecGet(t *testing.T) {
	keeper, ctx := keepertest.SpecKeeper(t)
	items := createNSpec(keeper, ctx, 10)
	for _, item := range items {
		got, found := keeper.GetSpec(ctx, item.Id)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&got),
		)
	}
}

func TestSpecRemove(t *testing.T) {
	keeper, ctx := keepertest.SpecKeeper(t)
	items := createNSpec(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveSpec(ctx, item.Id)
		_, found := keeper.GetSpec(ctx, item.Id)
		require.False(t, found)
	}
}

func TestSpecGetAll(t *testing.T) {
	keeper, ctx := keepertest.SpecKeeper(t)
	items := createNSpec(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllSpec(ctx)),
	)
}

func TestSpecCount(t *testing.T) {
	keeper, ctx := keepertest.SpecKeeper(t)
	items := createNSpec(keeper, ctx, 10)
	count := uint64(len(items))
	require.Equal(t, count, keeper.GetSpecCount(ctx))
}
