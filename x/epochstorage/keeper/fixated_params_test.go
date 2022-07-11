package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/epochstorage/keeper"
	"github.com/lavanet/lava/x/epochstorage/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNFixatedParams(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.FixatedParams {
	items := make([]types.FixatedParams, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetFixatedParams(ctx, items[i])
	}
	return items
}

func TestFixatedParamsGet(t *testing.T) {
	keeper, ctx := keepertest.EpochstorageKeeper(t)
	items := createNFixatedParams(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetFixatedParams(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestFixatedParamsRemove(t *testing.T) {
	keeper, ctx := keepertest.EpochstorageKeeper(t)
	items := createNFixatedParams(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveFixatedParams(ctx,
			item.Index,
		)
		_, found := keeper.GetFixatedParams(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestFixatedParamsGetAll(t *testing.T) {
	keeper, ctx := keepertest.EpochstorageKeeper(t)
	items := createNFixatedParams(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllFixatedParams(ctx)),
	)
}
