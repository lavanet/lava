package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/pairing/keeper"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNClientPaymentStorage(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.ClientPaymentStorage {
	items := make([]types.ClientPaymentStorage, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetClientPaymentStorage(ctx, items[i])
	}
	return items
}

func TestClientPaymentStorageGet(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNClientPaymentStorage(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetClientPaymentStorage(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestClientPaymentStorageRemove(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNClientPaymentStorage(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveClientPaymentStorage(ctx,
			item.Index,
		)
		_, found := keeper.GetClientPaymentStorage(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestClientPaymentStorageGetAll(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNClientPaymentStorage(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllClientPaymentStorage(ctx)),
	)
}
