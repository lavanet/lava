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

func createNUniquePaymentStorageClientProvider(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.UniquePaymentStorageClientProvider {
	items := make([]types.UniquePaymentStorageClientProvider, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetUniquePaymentStorageClientProvider(ctx, items[i])
	}
	return items
}

func TestUniquePaymentStorageClientProviderGet(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNUniquePaymentStorageClientProvider(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetUniquePaymentStorageClientProvider(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}

func TestUniquePaymentStorageClientProviderRemove(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNUniquePaymentStorageClientProvider(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveUniquePaymentStorageClientProvider(ctx,
			item.Index,
		)
		_, found := keeper.GetUniquePaymentStorageClientProvider(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestUniquePaymentStorageClientProviderGetAll(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNUniquePaymentStorageClientProvider(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllUniquePaymentStorageClientProvider(ctx)),
	)
}
