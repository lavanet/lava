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

func createNUserPaymentStorage(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.UserPaymentStorage {
	items := make([]types.UserPaymentStorage, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetUserPaymentStorage(ctx, items[i])
	}
	return items
}

func TestUserPaymentStorageGet(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	items := createNUserPaymentStorage(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetUserPaymentStorage(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestUserPaymentStorageRemove(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	items := createNUserPaymentStorage(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveUserPaymentStorage(ctx,
			item.Index,
		)
		_, found := keeper.GetUserPaymentStorage(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestUserPaymentStorageGetAll(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	items := createNUserPaymentStorage(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllUserPaymentStorage(ctx)),
	)
}
