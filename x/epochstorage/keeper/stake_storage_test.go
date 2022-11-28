package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/epochstorage/keeper"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

const advanceFewEpochs = 4
const stakeStorageSlots = 10

func createNStakeStorage(keeper *keeper.Keeper, ctx sdk.Context, n int) []epochstoragetypes.StakeStorage {
	items := make([]epochstoragetypes.StakeStorage, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.SetStakeStorage(ctx, items[i])
	}
	return items
}

func TestStakeStorageGet(t *testing.T) {
	keeper, ctx := testkeeper.EpochstorageKeeper(t)
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
	keeper, ctx := testkeeper.EpochstorageKeeper(t)
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
	// keeper, ctx := keepertest.EpochstorageKeeper(t)
	_, allkeepers, ctxx := testkeeper.InitAllKeepers(t)
	allkeepers.Epochstorage.SetEpochDetails(sdk.UnwrapSDKContext(ctxx), *epochstoragetypes.DefaultGenesis().EpochDetails)

	keeper := allkeepers.Epochstorage
	ctx := sdk.UnwrapSDKContext(ctxx)

	items := make([]epochstoragetypes.StakeStorage, stakeStorageSlots)
	chainID := "ETH1"
	storageType := epochstoragetypes.ProviderKey
	for i := 0; i < len(items); i++ {
		items[i].Index = keeper.StakeStorageKey(storageType, uint64(i), chainID)
		keeper.SetStakeStorage(ctx, items[i])
	}
	storageType = epochstoragetypes.ClientKey
	for i := 0; i < len(items); i++ {
		items[i].Index = keeper.StakeStorageKey(storageType, uint64(i), chainID)
		keeper.SetStakeStorage(ctx, items[i])
	}

	for i := 0; i < advanceFewEpochs; i++ {
		testkeeper.AdvanceEpoch(ctxx, allkeepers)
	}

	keeper.RemoveAllEntriesPriorToBlockNumber(ctx, 10, []string{"COS3ETH1LAV1COS4"})
	allStorage := keeper.GetAllStakeStorage(ctx)
	require.Equal(t, len(allStorage), stakeStorageSlots*2) // no entry was removed

	keeper.RemoveAllEntriesPriorToBlockNumber(ctx, 10, []string{"COS3"})
	allStorage = keeper.GetAllStakeStorage(ctx)
	require.Equal(t, len(allStorage), stakeStorageSlots*2) // no entry was removed

	keeper.RemoveAllEntriesPriorToBlockNumber(ctx, 0, []string{chainID})
	allStorage = keeper.GetAllStakeStorage(ctx)
	require.Equal(t, len(allStorage), stakeStorageSlots*2) // no entry was removed

	keeper.RemoveAllEntriesPriorToBlockNumber(ctx, 9, []string{chainID})
	allStorage = keeper.GetAllStakeStorage(ctx)
	require.Equal(t, len(allStorage), 2) // one provider one client

	keeper.RemoveAllEntriesPriorToBlockNumber(ctx, 10, []string{chainID})
	allStorage = keeper.GetAllStakeStorage(ctx)
	require.Equal(t, len(allStorage), 0) // zero entries left

	items[0].Index = epochstoragetypes.ProviderKey + strconv.FormatUint(uint64(10), 10) + ""
	keeper.SetStakeStorage(ctx, items[0])
	keeper.RemoveAllEntriesPriorToBlockNumber(ctx, 11, []string{""})
	allStorage = keeper.GetAllStakeStorage(ctx)
	require.Equal(t, len(allStorage), 0) // zero entries left
}

func TestStakeStorageGetAll(t *testing.T) {
	keeper, ctx := testkeeper.EpochstorageKeeper(t)
	items := createNStakeStorage(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllStakeStorage(ctx)),
	)
}
