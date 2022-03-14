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

func createNSpecStakeStorage(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.SpecStakeStorage {
	items := make([]types.SpecStakeStorage, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetSpecStakeStorage(ctx, items[i])
	}
	return items
}

func TestSpecStakeStorageGet(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	items := createNSpecStakeStorage(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetSpecStakeStorage(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestSpecStakeStorageRemove(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	items := createNSpecStakeStorage(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveSpecStakeStorage(ctx,
			item.Index,
		)
		_, found := keeper.GetSpecStakeStorage(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestSpecStakeStorageGetAll(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	items := createNSpecStakeStorage(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllSpecStakeStorage(ctx)),
	)
}
