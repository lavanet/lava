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

func createNStakeStorage(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.StakeStorage {
	items := make([]types.StakeStorage, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetStakeStorage(ctx, items[i])
	}
	return items
}

func TestStakeStorageGet(t *testing.T) {
	keeper, ctx := keepertest.EpochstorageKeeper(t)
	items := createNStakeStorage(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetStakeStorage(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestStakeStorageRemove(t *testing.T) {
	keeper, ctx := keepertest.EpochstorageKeeper(t)
	items := createNStakeStorage(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveStakeStorage(ctx,
			item.Index,
		)
		_, found := keeper.GetStakeStorage(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestStakeStorageGetAll(t *testing.T) {
	keeper, ctx := keepertest.EpochstorageKeeper(t)
	items := createNStakeStorage(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllStakeStorage(ctx)),
	)
}
