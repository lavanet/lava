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

func createNProviderPaymentStorage(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.ProviderPaymentStorage {
	items := make([]types.ProviderPaymentStorage, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetProviderPaymentStorage(ctx, items[i])
	}
	return items
}

func TestProviderPaymentStorageGet(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNProviderPaymentStorage(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetProviderPaymentStorage(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestProviderPaymentStorageRemove(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNProviderPaymentStorage(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveProviderPaymentStorage(ctx,
			item.Index,
		)
		_, found := keeper.GetProviderPaymentStorage(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestProviderPaymentStorageGetAll(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNProviderPaymentStorage(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllProviderPaymentStorage(ctx)),
	)
}
