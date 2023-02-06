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

func createNEpochPayments(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.EpochPayments {
	items := make([]types.EpochPayments, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetEpochPayments(ctx, items[i])
	}
	return items
}

func TestEpochPaymentsGet(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNEpochPayments(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetEpochPayments(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}

func TestEpochPaymentsRemove(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNEpochPayments(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveEpochPayments(ctx,
			item.Index,
		)
		_, found := keeper.GetEpochPayments(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestEpochPaymentsGetAll(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNEpochPayments(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllEpochPayments(ctx)),
	)
}
