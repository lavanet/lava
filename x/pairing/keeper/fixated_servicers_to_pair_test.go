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

func createNFixatedServicersToPair(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.FixatedServicersToPair {
	items := make([]types.FixatedServicersToPair, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetFixatedServicersToPair(ctx, items[i])
	}
	return items
}

func TestFixatedServicersToPairGet(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNFixatedServicersToPair(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetFixatedServicersToPair(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestFixatedServicersToPairRemove(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNFixatedServicersToPair(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveFixatedServicersToPair(ctx,
			item.Index,
		)
		_, found := keeper.GetFixatedServicersToPair(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestFixatedServicersToPairGetAll(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNFixatedServicersToPair(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllFixatedServicersToPair(ctx)),
	)
}
