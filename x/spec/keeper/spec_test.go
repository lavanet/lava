package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/spec/keeper"
	"github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNSpec(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Spec {
	items := make([]types.Spec, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetSpec(ctx, items[i])
	}
	return items
}

func TestSpecGet(t *testing.T) {
	keeper, ctx := keepertest.SpecKeeper(t)
	items := createNSpec(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetSpec(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestSpecRemove(t *testing.T) {
	keeper, ctx := keepertest.SpecKeeper(t)
	items := createNSpec(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveSpec(ctx,
			item.Index,
		)
		_, found := keeper.GetSpec(ctx,
			item.Index,
		)
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
