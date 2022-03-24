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

func createTestPreviousSessionBlocks(keeper *keeper.Keeper, ctx sdk.Context) types.PreviousSessionBlocks {
	item := types.PreviousSessionBlocks{}
	keeper.SetPreviousSessionBlocks(ctx, item)
	return item
}

func TestPreviousSessionBlocksGet(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	item := createTestPreviousSessionBlocks(keeper, ctx)
	rst, found := keeper.GetPreviousSessionBlocks(ctx)
	require.True(t, found)
	require.Equal(t,
		nullify.Fill(&item),
		nullify.Fill(&rst),
	)
}

func TestPreviousSessionBlocksRemove(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	createTestPreviousSessionBlocks(keeper, ctx)
	keeper.RemovePreviousSessionBlocks(ctx)
	_, found := keeper.GetPreviousSessionBlocks(ctx)
	require.False(t, found)
}
