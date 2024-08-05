package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/lavanet/lava/v2/testutil/keeper"
	"github.com/lavanet/lava/v2/testutil/nullify"
	"github.com/lavanet/lava/v2/x/conflict/keeper"
	"github.com/lavanet/lava/v2/x/conflict/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

func createNConflictVote(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.ConflictVote {
	items := make([]types.ConflictVote, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetConflictVote(ctx, items[i])
	}
	return items
}

func TestConflictVoteGet(t *testing.T) {
	keeper, ctx := keepertest.ConflictKeeper(t)
	items := createNConflictVote(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetConflictVote(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}

func TestConflictVoteRemove(t *testing.T) {
	keeper, ctx := keepertest.ConflictKeeper(t)
	items := createNConflictVote(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemoveConflictVote(ctx,
			item.Index,
		)
		_, found := keeper.GetConflictVote(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestConflictVoteGetAll(t *testing.T) {
	keeper, ctx := keepertest.ConflictKeeper(t)
	items := createNConflictVote(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllConflictVote(ctx)),
	)
}
