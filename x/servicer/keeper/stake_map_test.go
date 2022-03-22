package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/servicer/keeper"
	"github.com/lavanet/lava/x/servicer/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNStakeMap(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.StakeMap {
	items := make([]types.StakeMap, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetStakeMap(ctx, items[i])
	}
	return items
}

func TestStakeMapGet(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	items := createNStakeMap(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetStakeMap(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestStakeMapRemove(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	items := createNStakeMap(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveStakeMap(ctx,
			item.Index,
		)
		_, found := keeper.GetStakeMap(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestStakeMapGetAll(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	items := createNStakeMap(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllStakeMap(ctx)),
	)
}
