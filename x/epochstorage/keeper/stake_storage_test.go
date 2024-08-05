package keeper_test

import (
	"strconv"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	testkeeper "github.com/lavanet/lava/v2/testutil/keeper"
	"github.com/lavanet/lava/v2/testutil/nullify"
	"github.com/lavanet/lava/v2/x/epochstorage/keeper"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

const (
	advanceFewEpochs  = 4
	stakeStorageSlots = 10
)

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

func removeAllEntriesBeforeBlock(keeper keeper.Keeper, ctx sdk.Context, block uint64, allChainID []string) {
	allStorage := keeper.GetAllStakeStorage(ctx)
	for _, chainId := range allChainID {
		for _, entry := range allStorage {
			if strings.Contains(entry.Index, chainId) {
				storageBlock := entry.Index[:(len(entry.Index) - len(chainId))]
				blockHeight, err := strconv.ParseUint(storageBlock, 10, 64)
				if err != nil {
					if storageBlock == "" {
						// empty storageBlock means stake entry current, so skip it
						continue
					}
					panic("failed to decode storage block: " + strconv.Itoa(int(block)) +
						"chainID: " + chainId + "index: " + entry.Index)
				}
				if blockHeight < block {
					keeper.RemoveStakeStorage(ctx, entry.Index)
				}
			}
		}
	}
}

func TestStakeStorageRemoveAllPriorToBlock(t *testing.T) {
	// keeper, ctx := keepertest.EpochstorageKeeper(t)
	_, allkeepers, ctxx := testkeeper.InitAllKeepers(t)

	keeper := allkeepers.Epochstorage
	ctx := sdk.UnwrapSDKContext(ctxx)

	items := make([]epochstoragetypes.StakeStorage, stakeStorageSlots)
	chainID := "ETH1"
	for i := 0; i < len(items); i++ {
		items[i].Index = keeper.StakeStorageKey(uint64(i), chainID)
		keeper.SetStakeStorage(ctx, items[i])
	}

	for i := 0; i < advanceFewEpochs; i++ {
		testkeeper.AdvanceEpoch(ctxx, allkeepers)
	}

	removeAllEntriesBeforeBlock(keeper, ctx, 10, []string{"OSMOSISETH1LAV1OSMOSIST"})
	allStorage := keeper.GetAllStakeStorage(ctx)
	require.Equal(t, len(allStorage), stakeStorageSlots) // no entry was removed

	removeAllEntriesBeforeBlock(keeper, ctx, 10, []string{"OSMOSIS"})
	allStorage = keeper.GetAllStakeStorage(ctx)
	require.Equal(t, len(allStorage), stakeStorageSlots) // no entry was removed

	removeAllEntriesBeforeBlock(keeper, ctx, 0, []string{chainID})
	allStorage = keeper.GetAllStakeStorage(ctx)
	require.Equal(t, len(allStorage), stakeStorageSlots) // no entry was removed

	removeAllEntriesBeforeBlock(keeper, ctx, 9, []string{chainID})
	allStorage = keeper.GetAllStakeStorage(ctx)
	require.Equal(t, len(allStorage), 1) // one provider

	removeAllEntriesBeforeBlock(keeper, ctx, 10, []string{chainID})
	allStorage = keeper.GetAllStakeStorage(ctx)
	require.Equal(t, len(allStorage), 0) // zero entries left

	items[0].Index = strconv.FormatUint(uint64(10), 10) + ""
	keeper.SetStakeStorage(ctx, items[0])
	removeAllEntriesBeforeBlock(keeper, ctx, 11, []string{""})
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
