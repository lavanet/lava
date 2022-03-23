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

func createTestCurrentSessionStart(keeper *keeper.Keeper, ctx sdk.Context) types.CurrentSessionStart {
	item := types.CurrentSessionStart{}
	keeper.SetCurrentSessionStart(ctx, item)
	return item
}

func TestCurrentSessionStartGet(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	item := createTestCurrentSessionStart(keeper, ctx)
	rst, found := keeper.GetCurrentSessionStart(ctx)
	require.True(t, found)
	require.Equal(t,
		nullify.Fill(&item),
		nullify.Fill(&rst),
	)
}

func TestCurrentSessionStartRemove(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	createTestCurrentSessionStart(keeper, ctx)
	keeper.RemoveCurrentSessionStart(ctx)
	_, found := keeper.GetCurrentSessionStart(ctx)
	require.False(t, found)
}
