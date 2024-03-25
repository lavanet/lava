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
		keeper.SetUniqueEpochSession(ctx, uint64(i), items[i], items[i], items[i], uint64(i))
	}
	return items
}

func TestUniqueEpochSessionGet(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNUniqueEpochSession(keeper, ctx, 10)
	for i := range items {
		item := strconv.Itoa(i)
		found := keeper.GetUniqueEpochSession(ctx, uint64(i), item, item, item, uint64(i))
		require.True(t, found)
	}
}

func TestUniqueEpochSessionRemove(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNUniqueEpochSession(keeper, ctx, 10)
	for i := range items {
		keeper.RemoveUniqueEpochSessions(ctx, uint64(i))
		item := strconv.Itoa(i)
		found := keeper.GetUniqueEpochSession(ctx, uint64(i), item, item, item, uint64(i))
		require.False(t, found)
	}
}

func TestUniqueEpochSessionGetAll(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNUniqueEpochSession(keeper, ctx, 10)
	expectedKeys := []string{}
	keys := []string{}
	for i, item := range items {
		key := string(types.UniqueEpochSessionKey(item, item, item, uint64(i)))
		expectedKeys = append(expectedKeys, key)
		keys = append(keys, keeper.GetAllUniqueEpochSession(ctx, uint64(i))...)
	}
	require.ElementsMatch(t, nullify.Fill(expectedKeys), nullify.Fill(keys))
}

func TestUniqueEpochSessionGetAllStore(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNUniqueEpochSession(keeper, ctx, 10)
	expectedKeys := []string{}
	for i, item := range items {
		key := string(types.UniqueEpochSessionKey(item, item, item, uint64(i)))
		expectedKeys = append(expectedKeys, key)
	}
	_, keys := keeper.GetAllUniqueEpochSessionStore(ctx)
	require.ElementsMatch(t, nullify.Fill(expectedKeys), nullify.Fill(keys))
}

/* ########## ProviderEpochCu ############ */

func createNProviderEpochCu(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.ProviderEpochCu {
	items := make([]types.ProviderEpochCu, n)
	for i := range items {
		provider := strconv.Itoa(i)
		items[i] = types.ProviderEpochCu{ServicedCu: uint64(i)}
		keeper.SetProviderEpochCu(ctx, uint64(i), provider, items[i])
	}
	return items
}

func TestProviderEpochCuGet(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNProviderEpochCu(keeper, ctx, 10)
	for i, item := range items {
		provider := strconv.Itoa(i)
		pec, found := keeper.GetProviderEpochCu(ctx, uint64(i), provider)
		require.True(t, found)
		require.Equal(t, item.ServicedCu, pec.ServicedCu)
	}
}

func TestProviderEpochCuRemove(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNProviderEpochCu(keeper, ctx, 10)
	for i := range items {
		provider := strconv.Itoa(i)
		keeper.RemoveProviderEpochCu(ctx, uint64(i), provider)
		_, found := keeper.GetProviderEpochCu(ctx, uint64(i), provider)
		require.False(t, found)
	}
}

func TestProviderEpochCuGetAllStore(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNProviderEpochCu(keeper, ctx, 10)
	expectedKeys := []string{}
	for i := range items {
		provider := strconv.Itoa(i)
		key := string(types.ProviderEpochCuKey(provider))
		expectedKeys = append(expectedKeys, key)
	}
	_, keys, pecs := keeper.GetAllProviderEpochCuStore(ctx)
	require.ElementsMatch(t, nullify.Fill(expectedKeys), nullify.Fill(keys))
	require.ElementsMatch(t, items, pecs)
}

/* ########## ProviderConsumerEpochCu ############ */

func createNProviderConsumerEpochCu(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.ProviderConsumerEpochCu {
	items := make([]types.ProviderConsumerEpochCu, n)
	for i := range items {
		name := strconv.Itoa(i)
		items[i] = types.ProviderConsumerEpochCu{Cu: uint64(i)}
		keeper.SetProviderConsumerEpochCu(ctx, uint64(i), name, name, items[i])
	}
	return items
}

func TestProviderConsumerEpochCuGet(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNProviderConsumerEpochCu(keeper, ctx, 10)
	for i, item := range items {
		name := strconv.Itoa(i)
		pecc, found := keeper.GetProviderConsumerEpochCu(ctx, uint64(i), name, name)
		require.True(t, found)
		require.Equal(t, item.Cu, pecc.Cu)
	}
}

func TestProviderConsumerEpochCuRemove(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNProviderConsumerEpochCu(keeper, ctx, 10)
	for i := range items {
		name := strconv.Itoa(i)
		keeper.RemoveProviderConsumerEpochCu(ctx, uint64(i), name, name)
		_, found := keeper.GetProviderConsumerEpochCu(ctx, uint64(i), name, name)
		require.False(t, found)
	}
}

func TestProviderConsumerEpochCuGetAllStore(t *testing.T) {
	keeper, ctx := keepertest.PairingKeeper(t)
	items := createNProviderConsumerEpochCu(keeper, ctx, 10)
	expectedKeys := []string{}
	for i := range items {
		name := strconv.Itoa(i)
		key := string(types.ProviderConsumerEpochCuKey(name, name))
		expectedKeys = append(expectedKeys, key)
	}
	_, keys, pecs := keeper.GetAllProviderConsumerEpochCuStore(ctx)
	require.ElementsMatch(t, nullify.Fill(expectedKeys), nullify.Fill(keys))
	require.ElementsMatch(t, items, pecs)
}
