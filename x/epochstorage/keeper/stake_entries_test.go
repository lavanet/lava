package keeper_test

import (
	"strconv"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/x/epochstorage/keeper"
	"github.com/lavanet/lava/x/epochstorage/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

/* ########## StakeEntry ############ */

func createNStakeEntries(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.StakeEntry {
	items := make([]types.StakeEntry, n)
	for i := range items {
		items[i] = types.StakeEntry{
			Stake:   sdk.NewCoin("token", math.NewInt(int64(i))),
			Address: strconv.Itoa(i),
			Chain:   strconv.Itoa(i),
		}
		keeper.SetStakeEntry(ctx, uint64(i), items[i])
	}
	return items
}

func TestGetStakeEntry(t *testing.T) {
	keeper, ctx := keepertest.EpochstorageKeeper(t)
	items := createNStakeEntries(keeper, ctx, 10)
	for i, item := range items {
		entry, found := keeper.GetStakeEntry(ctx, uint64(i), strconv.Itoa(i), strconv.Itoa(i))
		require.True(t, found)
		require.True(t, item.Stake.IsEqual(entry.Stake))
	}
}

func TestRemoveAllStakeEntriesForEpoch(t *testing.T) {
	keeper, ctx := keepertest.EpochstorageKeeper(t)
	items := createNStakeEntries(keeper, ctx, 10)
	epoch := uint64(1)
	keeper.SetStakeEntry(ctx, epoch, types.StakeEntry{}) // set another stake entry on epoch=1
	keeper.RemoveAllStakeEntriesForEpoch(ctx, epoch)
	for i := range items {
		entries := keeper.GetAllStakeEntriesForEpoch(ctx, uint64(i))
		if i == 1 {
			require.Len(t, entries, 0) // on epoch 1 there should be no entries
		} else {
			require.Len(t, entries, 1) // every other epoch is unaffected - there should be one entry
		}
	}
}

func TestGetAllStakeEntriesForEpoch(t *testing.T) {
	keeper, ctx := keepertest.EpochstorageKeeper(t)
	items := createNStakeEntries(keeper, ctx, 10)
	epoch := uint64(1)
	keeper.SetStakeEntry(ctx, epoch, types.StakeEntry{}) // set another stake entry on epoch=1
	for i := range items {
		entries := keeper.GetAllStakeEntriesForEpoch(ctx, uint64(i))
		if i == 1 {
			require.Len(t, entries, 2) // on epoch 1 there should be two entries
		} else {
			require.Len(t, entries, 1) // every other epoch is unaffected - there should be one entry
		}
	}
}

func TestGetAllStakeEntriesForEpochChainId(t *testing.T) {
	keeper, ctx := keepertest.EpochstorageKeeper(t)
	items := createNStakeEntries(keeper, ctx, 10)
	epoch := uint64(1)

	// set two entries on epoch=1: one with the same chain as items, and another with a different chain
	keeper.SetStakeEntry(ctx, epoch, types.StakeEntry{Chain: items[1].Chain})
	keeper.SetStakeEntry(ctx, epoch, types.StakeEntry{Chain: "otherChain"})

	// there should be 3 entries on epoch=1: two of items[1].Chain and one of "otherChain"
	entries := keeper.GetAllStakeEntriesForEpochChainId(ctx, epoch, items[1].Chain)
	require.Len(t, entries, 2)
	entries = keeper.GetAllStakeEntriesForEpochChainId(ctx, epoch, "otherChain")
	require.Len(t, entries, 1)
}

/* ########## Current StakeEntry ############ */

func createNStakeEntriesCurrent(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.StakeEntry {
	items := make([]types.StakeEntry, n)
	for i := range items {
		items[i] = types.StakeEntry{
			Stake:   sdk.NewCoin("token", math.NewInt(int64(i))),
			Address: strconv.Itoa(i),
			Vault:   strconv.Itoa(i),
			Chain:   strconv.Itoa(i),
		}
		keeper.SetStakeEntryCurrent(ctx, items[i])
	}
	return items
}

func TestGetStakeEntryCurrent(t *testing.T) {
	keeper, ctx := keepertest.EpochstorageKeeper(t)
	items := createNStakeEntriesCurrent(keeper, ctx, 10)
	for i, item := range items {
		entry, found := keeper.GetStakeEntryCurrent(ctx, strconv.Itoa(i), strconv.Itoa(i))
		require.True(t, found)
		require.True(t, item.Stake.IsEqual(entry.Stake))
	}
}

func TestRemoveStakeEntryCurrent(t *testing.T) {
	keeper, ctx := keepertest.EpochstorageKeeper(t)
	items := createNStakeEntriesCurrent(keeper, ctx, 10)
	for i := range items {
		keeper.RemoveStakeEntryCurrent(ctx, strconv.Itoa(i), strconv.Itoa(i))
		_, found := keeper.GetStakeEntryCurrent(ctx, strconv.Itoa(i), strconv.Itoa(i))
		require.False(t, found)
	}
}
