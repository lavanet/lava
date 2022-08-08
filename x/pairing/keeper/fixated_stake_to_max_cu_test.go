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

func createNFixatedStakeToMaxCu(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.FixatedStakeToMaxCu {
	items := make([]types.FixatedStakeToMaxCu, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetFixatedStakeToMaxCu(ctx, items[i])
	}
	return items
}

func TestFixatedStakeToMaxCuGet(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNFixatedStakeToMaxCu(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetFixatedStakeToMaxCu(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestFixatedStakeToMaxCuRemove(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNFixatedStakeToMaxCu(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveFixatedStakeToMaxCu(ctx,
			item.Index,
		)
		_, found := keeper.GetFixatedStakeToMaxCu(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestFixatedStakeToMaxCuGetAll(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNFixatedStakeToMaxCu(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllFixatedStakeToMaxCu(ctx)),
	)
}
