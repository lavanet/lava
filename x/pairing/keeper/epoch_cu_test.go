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

/* ########## UniqueEpochSession ############ */

func createNUniqueEpochSession(keeper *keeper.Keeper, ctx sdk.Context, n int) []string {
	items := make([]string, n)
	for i := range items {
		items[i] = strconv.Itoa(i)
		keeper.SetUniqueEpochSession(ctx, items[i], items[i], items[i], uint64(i))
	}
	return items
}

func TestUniqueEpochSessionGet(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNUniqueEpochSession(keeper, ctx, 10)
	for i := range items {
		item := strconv.Itoa(i)
		found := keeper.GetUniqueEpochSession(ctx, item, item, item, uint64(i))
		require.True(t, found)
	}
}

func TestUniqueEpochSessionRemove(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNUniqueEpochSession(keeper, ctx, 10)
	keeper.RemoveUniqueEpochSessions(ctx, 0)
	for i := range items {
		item := strconv.Itoa(i)
		found := keeper.GetUniqueEpochSession(ctx, item, item, item, uint64(i))
		require.False(t, found)
	}
}

func TestUniqueEpochSessionGetAll(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNUniqueEpochSession(keeper, ctx, 10)
	expectedKeys := []string{}
	for i, item := range items {
		key := string(types.UniqueEpochSessionKey(item, item, item, uint64(i)))
		expectedKeys = append(expectedKeys, key)
	}
	require.ElementsMatch(t, nullify.Fill(expectedKeys), nullify.Fill(keeper.GetAllUniqueEpochSession(ctx, 0)))
}

/* ########## ProviderEpochCu ############ */

func createNProviderEpochCu(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.ProviderEpochCu {
	items := make([]types.ProviderEpochCu, n)
	for i := range items {
		provider := strconv.Itoa(i)
		items[i] = types.ProviderEpochCu{ServicedCu: uint64(i)}
		keeper.SetProviderEpochCu(ctx, provider, items[i])
	}
	return items
}

func TestProviderEpochCuGet(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNProviderEpochCu(keeper, ctx, 10)
	for i, item := range items {
		provider := strconv.Itoa(i)
		pec, found := keeper.GetProviderEpochCu(ctx, 0, provider)
		require.True(t, found)
		require.Equal(t, item.ServicedCu, pec.ServicedCu)
	}
}

func TestProviderEpochCuRemove(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNProviderEpochCu(keeper, ctx, 10)
	for i := range items {
		provider := strconv.Itoa(i)
		keeper.RemoveProviderEpochCu(ctx, 0, provider)
		_, found := keeper.GetProviderEpochCu(ctx, 0, provider)
		require.False(t, found)
	}
}

/* ########## ProviderConsumerEpochCu ############ */

func createNProviderConsumerEpochCu(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.ProviderConsumerEpochCu {
	items := make([]types.ProviderConsumerEpochCu, n)
	for i := range items {
		name := strconv.Itoa(i)
		items[i] = types.ProviderConsumerEpochCu{Cu: uint64(i)}
		keeper.SetProviderConsumerEpochCu(ctx, name, name, items[i])
	}
	return items
}

func TestProviderConsumerEpochCuGet(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNProviderConsumerEpochCu(keeper, ctx, 10)
	for i, item := range items {
		name := strconv.Itoa(i)
		pecc, found := keeper.GetProviderConsumerEpochCu(ctx, 0, name, name)
		require.True(t, found)
		require.Equal(t, item.Cu, pecc.Cu)
	}
}

func TestProviderConsumerEpochCuRemove(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNProviderConsumerEpochCu(keeper, ctx, 10)
	for i := range items {
		name := strconv.Itoa(i)
		keeper.RemoveProviderConsumerEpochCu(ctx, 0, name, name)
		_, found := keeper.GetProviderConsumerEpochCu(ctx, 0, name, name)
		require.False(t, found)
	}
}
