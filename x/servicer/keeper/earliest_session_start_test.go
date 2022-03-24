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

func createTestEarliestSessionStart(keeper *keeper.Keeper, ctx sdk.Context) types.EarliestSessionStart {
	item := types.EarliestSessionStart{}
	keeper.SetEarliestSessionStart(ctx, item)
	return item
}

func TestEarliestSessionStartGet(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	item := createTestEarliestSessionStart(keeper, ctx)
	rst, found := keeper.GetEarliestSessionStart(ctx)
	require.True(t, found)
	require.Equal(t,
		nullify.Fill(&item),
		nullify.Fill(&rst),
	)
}

func TestEarliestSessionStartRemove(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	createTestEarliestSessionStart(keeper, ctx)
	keeper.RemoveEarliestSessionStart(ctx)
	_, found := keeper.GetEarliestSessionStart(ctx)
	require.False(t, found)
}
