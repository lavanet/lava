package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/pairing/keeper"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNFixatedEpochBlocksOverlap(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.FixatedEpochBlocksOverlap {
	items := make([]types.FixatedEpochBlocksOverlap, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetFixatedEpochBlocksOverlap(ctx, items[i])
	}
	return items
}

func TestFixatedEpochBlocksOverlapGet(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNFixatedEpochBlocksOverlap(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetFixatedEpochBlocksOverlap(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestFixatedEpochBlocksOverlapRemove(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNFixatedEpochBlocksOverlap(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveFixatedEpochBlocksOverlap(ctx,
			item.Index,
		)
		_, found := keeper.GetFixatedEpochBlocksOverlap(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestFixatedEpochBlocksOverlapGetAll(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNFixatedEpochBlocksOverlap(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllFixatedEpochBlocksOverlap(ctx)),
	)
}
