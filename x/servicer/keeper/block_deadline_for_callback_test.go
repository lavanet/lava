package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/servicer/keeper"
	"github.com/lavanet/lava/x/servicer/types"
)

func createTestBlockDeadlineForCallback(keeper *keeper.Keeper, ctx sdk.Context) types.BlockDeadlineForCallback {
	item := types.BlockDeadlineForCallback{Deadline: types.BlockNum{Num: 0}}
	keeper.SetBlockDeadlineForCallback(ctx, item)
	return item
}

func TestBlockDeadlineForCallbackGet(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	item := createTestBlockDeadlineForCallback(keeper, ctx)
	rst, found := keeper.GetBlockDeadlineForCallback(ctx)
	require.True(t, found)
	require.Equal(t,
		nullify.Fill(&item),
		nullify.Fill(&rst),
	)
}

func TestBlockDeadlineForCallbackRemove(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	createTestBlockDeadlineForCallback(keeper, ctx)
	keeper.RemoveBlockDeadlineForCallback(ctx)
	_, found := keeper.GetBlockDeadlineForCallback(ctx)
	require.False(t, found)
}
