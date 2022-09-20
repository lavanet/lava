package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/epochstorage/keeper"
	"github.com/lavanet/lava/x/epochstorage/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

const stakeStorageSlots = 10

func createNStakeStorage(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.StakeStorage {
	items := make([]types.StakeStorage, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetStakeStorage(ctx, items[i])
	}
	return items
}

func TestStakeStorageGet(t *testing.T) {
	keeper, ctx := keepertest.EpochstorageKeeper(t)
	items := createNStakeStorage(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetStakeStorage(ctx,
			item.Index,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}
func TestStakeStorageRemove(t *testing.T) {
	keeper, ctx := keepertest.EpochstorageKeeper(t)
	items := createNStakeStorage(keeper, ctx, stakeStorageSlots)
	for _, item := range items {
		keeper.RemoveStakeStorage(ctx,
			item.Index,
		)
		_, found := keeper.GetStakeStorage(ctx,
			item.Index,
		)
		require.False(t, found)
	}
}

func TestStakeStorageRemoveAllPriorToBlock(t *testing.T) {
	keeper, ctx := keepertest.EpochstorageKeeper(t)
	items := make([]types.StakeStorage, stakeStorageSlots)
	chainID := "ETH1"
	storageType := types.ProviderKey
	for i := 0; i < len(items); i++ {
		items[i].Index = storageType + strconv.FormatUint(uint64(i), 10) + chainID
		keeper.SetStakeStorage(ctx, items[i])
	}
	storageType = types.ClientKey
	for i := 0; i < len(items); i++ {
		items[i].Index = storageType + strconv.FormatUint(uint64(i), 10) + chainID
		keeper.SetStakeStorage(ctx, items[i])
	}
	keeper.RemoveAllEntriesPriorToBlockNumber(ctx, 10, []string{"COS3ETH1LAV1COS4"})
	allStorage := keeper.GetAllStakeStorage(ctx)
	require.Equal(t, len(allStorage), stakeStorageSlots*2) // no entry was removed

	keeper.RemoveAllEntriesPriorToBlockNumber(ctx, 10, []string{"COS3"})
	allStorage = keeper.GetAllStakeStorage(ctx)
	require.Equal(t, len(allStorage), stakeStorageSlots*2) // no entry was removed

	keeper.RemoveAllEntriesPriorToBlockNumber(ctx, 0, []string{"ETH1"})
	allStorage = keeper.GetAllStakeStorage(ctx)
	require.Equal(t, len(allStorage), stakeStorageSlots*2) // no entry was removed

	keeper.RemoveAllEntriesPriorToBlockNumber(ctx, 9, []string{"ETH1"})
	allStorage = keeper.GetAllStakeStorage(ctx)
	require.Equal(t, len(allStorage), 2) // one provider one client

	keeper.RemoveAllEntriesPriorToBlockNumber(ctx, 10, []string{"ETH1"})
	allStorage = keeper.GetAllStakeStorage(ctx)
	require.Equal(t, len(allStorage), 0) // zero entries left
}

func TestStakeStorageGetAll(t *testing.T) {
	keeper, ctx := keepertest.EpochstorageKeeper(t)
	items := createNStakeStorage(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllStakeStorage(ctx)),
	)
}
