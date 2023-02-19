package common_test

import (
	"encoding/binary"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/stretchr/testify/require"
)

// Test addition and removal of a fixation entry
func TestFixationEntryAdditionAndRemoval(t *testing.T) {
	// create dummy data for dummy entry
	marshaledData := make([]byte, 8)
	binary.LittleEndian.PutUint64(marshaledData, 0)
	dummyIndex := "index"

	// init all keepers + context
	_, keepers, ctx := keepertest.InitAllKeepers(t)

	// add dummy entry
	err := common.AddEntry(sdk.UnwrapSDKContext(ctx), keepers.MockKeeper.StoreKey, "mock", "mockkeeper", keepers.MockKeeper.Cdc, dummyIndex, marshaledData)
	require.Nil(t, err)

	// get all entries with dummyIndex and make sure there is only one entry
	entryListFromStorage := common.GetAllEntriesForIndex(sdk.UnwrapSDKContext(ctx), keepers.MockKeeper.StoreKey, "mock", keepers.MockKeeper.Cdc, dummyIndex)
	require.Equal(t, 1, len(entryListFromStorage))

	// make sure that one entry's data is the same data that was used to create it
	marshaledDataFromStorage := entryListFromStorage[0].MarshaledData
	require.Equal(t, marshaledData, marshaledDataFromStorage)

	// remove the entry
	common.RemoveEntry(sdk.UnwrapSDKContext(ctx), keepers.MockKeeper.StoreKey, "mock", dummyIndex)

	// make sure there are no more entries with that index
	entryListFromStorage = common.GetAllEntriesForIndex(sdk.UnwrapSDKContext(ctx), keepers.MockKeeper.StoreKey, "mock", keepers.MockKeeper.Cdc, dummyIndex)
	require.Equal(t, 0, len(entryListFromStorage))
}

// Test that adds two entries in the same block and makes sure that only the latest one is kept
func TestAdditionOfTwoEntriesWithSameIndexInSameBlock(t *testing.T) {
	// create dummy data for two dummy entries
	marshaledData := make([]byte, 8)
	binary.LittleEndian.PutUint64(marshaledData, 0)
	marshaledData2 := make([]byte, 8)
	binary.LittleEndian.PutUint64(marshaledData2, 1)
	dummyIndex := "index"

	// init all keepers + context
	_, keepers, ctx := keepertest.InitAllKeepers(t)

	// add the first dummy entry
	err := common.AddEntry(sdk.UnwrapSDKContext(ctx), keepers.MockKeeper.StoreKey, "mock", "mockkeeper", keepers.MockKeeper.Cdc, dummyIndex, marshaledData)
	require.Nil(t, err)

	// add the second dummy entry
	err = common.AddEntry(sdk.UnwrapSDKContext(ctx), keepers.MockKeeper.StoreKey, "mock", "mockkeeper", keepers.MockKeeper.Cdc, dummyIndex, marshaledData2)
	require.Nil(t, err)

	// get all entries with dummyIndex and make sure there is only one entry
	entryListFromStorage := common.GetAllEntriesForIndex(sdk.UnwrapSDKContext(ctx), keepers.MockKeeper.StoreKey, "mock", keepers.MockKeeper.Cdc, dummyIndex)
	require.Equal(t, 1, len(entryListFromStorage))

	// make sure that one entry's data is the same data of the second dummy entry
	marshaledDataFromStorage := entryListFromStorage[0].MarshaledData
	require.Equal(t, marshaledData2, marshaledDataFromStorage)
}

// Test adding entry versions and getting an older version
func TestEntryVersions(t *testing.T) {
	// create dummy data for two dummy entries
	marshaledData := make([]byte, 8)
	binary.LittleEndian.PutUint64(marshaledData, 0)
	marshaledData2 := make([]byte, 8)
	binary.LittleEndian.PutUint64(marshaledData2, 1)
	dummyIndex := "index"

	// init all keepers + context
	_, keepers, ctx := keepertest.InitAllKeepers(t)

	// add the first dummy entry
	err := common.AddEntry(sdk.UnwrapSDKContext(ctx), keepers.MockKeeper.StoreKey, "mock", "mockkeeper", keepers.MockKeeper.Cdc, dummyIndex, marshaledData)
	require.Nil(t, err)

	// advance a block
	ctx = keepertest.AdvanceBlock(ctx, keepers)

	// add the second dummy entry
	err = common.AddEntry(sdk.UnwrapSDKContext(ctx), keepers.MockKeeper.StoreKey, "mock", "mockkeeper", keepers.MockKeeper.Cdc, dummyIndex, marshaledData2)
	require.Nil(t, err)

	// get all entries with dummyIndex and make sure there are two entry
	entryListFromStorage := common.GetAllEntriesForIndex(sdk.UnwrapSDKContext(ctx), keepers.MockKeeper.StoreKey, "mock", keepers.MockKeeper.Cdc, dummyIndex)
	require.Equal(t, 2, len(entryListFromStorage))

	// get the older version from block 0
	oldEntry, found := common.GetEntryOlderVersionByBlock(sdk.UnwrapSDKContext(ctx), keepers.MockKeeper.StoreKey, "mock", keepers.MockKeeper.Cdc, dummyIndex, uint64(0))
	require.True(t, found)

	// verify the index and data matches the old entry from storage
	oldEntryIndex := common.CreateOldVersionIndex(dummyIndex, uint64(0))
	require.Equal(t, oldEntryIndex, oldEntry.Index)
	require.Equal(t, marshaledData, oldEntry.MarshaledData)
}
