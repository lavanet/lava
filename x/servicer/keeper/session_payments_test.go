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

func createNSessionPayments(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.SessionPayments {
	items := make([]types.SessionPayments, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetSessionPayments(ctx, items[i])
	}
	return items
}

func TestSessionPaymentsGet(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	items := createNSessionPayments(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetSessionPayments(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestSessionPaymentsRemove(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	items := createNSessionPayments(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveSessionPayments(ctx,
			item.Index,
		)
		_, found := keeper.GetSessionPayments(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestSessionPaymentsGetAll(t *testing.T) {
	keeper, ctx := keepertest.ServicerKeeper(t)
	items := createNSessionPayments(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllSessionPayments(ctx)),
	)
}
