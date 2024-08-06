package keeper_test

import (
	"strconv"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/lavanet/lava/v2/testutil/keeper"
	commontypes "github.com/lavanet/lava/v2/utils/common/types"
	"github.com/lavanet/lava/v2/x/pairing/keeper"
	"github.com/lavanet/lava/v2/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func createNReputations(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Reputation {
	items := make([]types.Reputation, n)
	for i := range items {
		decIndex := math.LegacyNewDec(int64(i + 1))
		strIndex := strconv.Itoa(i)
		items[i] = types.Reputation{
			Score: types.QosScore{
				Score:    types.Frac{Num: decIndex, Denom: decIndex},
				Variance: types.Frac{Num: decIndex, Denom: decIndex},
			},
			EpochScore: types.QosScore{
				Score:    types.Frac{Num: decIndex, Denom: decIndex},
				Variance: types.Frac{Num: decIndex, Denom: decIndex},
			},
			TimeLastUpdated: int64(i),
			CreationTime:    int64(i),
			Stake:           sdk.NewCoin(commontypes.TokenDenom, sdk.NewInt(int64(i))),
		}
		keeper.SetReputation(ctx, strIndex, strIndex, strIndex, items[i])
	}
	return items
}

func TestGetReputation(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNReputations(keeper, ctx, 10)
	for i, item := range items {
		strIndex := strconv.Itoa(i)
		entry, found := keeper.GetReputation(ctx, strIndex, strIndex, strIndex)
		require.True(t, found)
		require.True(t, item.Equal(entry))
	}

	_, found := keeper.GetReputation(ctx, "dummy", "dummy", "dummy")
	require.False(t, found)
}

func TestRemoveReputation(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNReputations(keeper, ctx, 10)
	for i := range items {
		strIndex := strconv.Itoa(i)
		keeper.RemoveReputation(ctx, strIndex, strIndex, strIndex)
		_, found := keeper.GetReputation(ctx, strIndex, strIndex, strIndex)
		require.False(t, found)
	}

	require.Panics(t, func() { keeper.RemoveReputation(ctx, "dummy", "dummy", "dummy") })
}

func TestGetAllReputations(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNReputations(keeper, ctx, 10)
	genEntries := keeper.GetAllReputation(ctx)
	for i := range genEntries {
		require.True(t, items[i].Equal(genEntries[i].Reputation))
	}
}

func createNReputationsScores(keeper *keeper.Keeper, ctx sdk.Context, n int) ([]math.LegacyDec, sdk.Context) {
	items := make([]math.LegacyDec, n)
	height := ctx.BlockHeight()
	for i := range items {
		decIndex := math.LegacyNewDec(int64(i + 1))
		strIndex := strconv.Itoa(i)

		err := keeper.SetReputationScore(ctx, strIndex, strIndex, strIndex, decIndex)
		if err != nil {
			panic(err)
		}

		items[i] = decIndex
		height++
		ctx = ctx.WithBlockHeight(height)
	}
	return items, ctx
}

func TestGetReputationScore(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items, ctx := createNReputationsScores(keeper, ctx, 10)
	for i, item := range items {
		strIndex := strconv.Itoa(i)
		entry, found := keeper.GetReputationScore(ctx, strIndex, strIndex, strIndex)
		require.True(t, found)
		require.True(t, item.Equal(entry))
	}

	_, found := keeper.GetReputationScore(ctx, "dummy", "dummy", "dummy")
	require.False(t, found)
}

func TestGetReputationScoreForBlock(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items, ctx := createNReputationsScores(keeper, ctx, 10)
	for i, item := range items {
		strIndex := strconv.Itoa(i)
		entry, entryBlock, found := keeper.GetReputationScoreForBlock(ctx, strIndex, strIndex, strIndex, uint64(ctx.BlockHeight()))
		require.True(t, found)
		require.True(t, item.Equal(entry))
		require.Equal(t, uint64(i), entryBlock)
	}

	_, _, found := keeper.GetReputationScoreForBlock(ctx, "dummy", "dummy", "dummy", uint64(ctx.BlockHeight()))
	require.False(t, found)
	_, _, found = keeper.GetReputationScoreForBlock(ctx, "2", "2", "2", 1)
	require.False(t, found)
}

func TestRemoveReputationScore(t *testing.T) {
	ts := newTester(t)
	keeper, ctx := ts.Keepers.Pairing, ts.Ctx
	items, ctx := createNReputationsScores(&keeper, ctx, 10)
	for i := range items {
		strIndex := strconv.Itoa(i)
		err := keeper.RemoveReputationScore(ctx, strIndex, strIndex, strIndex)
		require.NoError(t, err)
		_, found := keeper.GetReputationScore(ctx, strIndex, strIndex, strIndex)
		require.True(t, found)
	}

	ts.AdvanceEpoch()           // removal applied
	ts.AdvanceEpochUntilStale() // deletion happens

	for i := range items {
		strIndex := strconv.Itoa(i)
		_, found := keeper.GetReputationScore(ctx, strIndex, strIndex, strIndex)
		require.False(t, found)
	}
}
