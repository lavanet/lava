package keeper_test

import (
	"strconv"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/x/pairing/keeper"
	"github.com/lavanet/lava/x/pairing/types"
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
			TimeLastUpdated: uint64(i),
			CreationTime:    uint64(i),
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
