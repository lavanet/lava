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

func createNSessionStorageForSpec(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.SessionStorageForSpec {
	items := make([]types.SessionStorageForSpec, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetSessionStorageForSpec(ctx, items[i])
	}
	return items
}

func TestSessionStorageForSpecGet(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	items := createNSessionStorageForSpec(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetSessionStorageForSpec(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestSessionStorageForSpecRemove(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	items := createNSessionStorageForSpec(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveSessionStorageForSpec(ctx,
			item.Index,
		)
		_, found := keeper.GetSessionStorageForSpec(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestSessionStorageForSpecGetAll(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	items := createNSessionStorageForSpec(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllSessionStorageForSpec(ctx)),
	)
}
