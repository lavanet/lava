package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/lavanet/lava/v2/testutil/keeper"
	"github.com/lavanet/lava/v2/testutil/nullify"
	"github.com/lavanet/lava/v2/x/pairing/keeper"
	"github.com/lavanet/lava/v2/x/pairing/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNBadgeUsedCu(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.BadgeUsedCu {
	items := make([]types.BadgeUsedCu, n)
	for i := range items {
		items[i].BadgeUsedCuKey = []byte{byte(i)}

		keeper.SetBadgeUsedCu(ctx, items[i])
	}
	return items
}

func TestBadgeUsedCuGet(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNBadgeUsedCu(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetBadgeUsedCu(ctx,
			item.BadgeUsedCuKey,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}

func TestBadgeUsedCuRemove(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNBadgeUsedCu(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveBadgeUsedCu(ctx,
			item.BadgeUsedCuKey,
		)
		_, found := keeper.GetBadgeUsedCu(ctx,
			item.BadgeUsedCuKey,
		)
		require.False(t, found)
	}
}

func TestBadgeUsedCuGetAll(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNBadgeUsedCu(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllBadgeUsedCu(ctx)),
	)
}
