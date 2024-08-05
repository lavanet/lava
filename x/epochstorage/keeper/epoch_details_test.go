package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/lavanet/lava/v2/testutil/keeper"
	"github.com/lavanet/lava/v2/testutil/nullify"
	"github.com/lavanet/lava/v2/x/epochstorage/keeper"
	"github.com/lavanet/lava/v2/x/epochstorage/types"
)

func createTestEpochDetails(keeper *keeper.Keeper, ctx sdk.Context) types.EpochDetails {
	item := types.EpochDetails{}
	keeper.SetEpochDetails(ctx, item)
	return item
}

func TestEpochDetailsGet(t *testing.T) {
	keeper, ctx := keepertest.EpochstorageKeeper(t)
	item := createTestEpochDetails(keeper, ctx)
	rst, found := keeper.GetEpochDetails(ctx)
	require.True(t, found)
	require.Equal(t,
		nullify.Fill(&item),
		nullify.Fill(&rst),
	)
}

func TestEpochDetailsRemove(t *testing.T) {
	keeper, ctx := keepertest.EpochstorageKeeper(t)
	createTestEpochDetails(keeper, ctx)
	keeper.RemoveEpochDetails(ctx)
	_, found := keeper.GetEpochDetails(ctx)
	require.False(t, found)
}
