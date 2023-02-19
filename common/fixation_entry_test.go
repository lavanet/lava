package common_test

import (
	"encoding/binary"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/lavanet/lava/common"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmdb "github.com/tendermint/tm-db"
)

type MockKeeper struct {
	Cdc        codec.BinaryCodec
	StoreKey   sdk.StoreKey
	MemKey     sdk.StoreKey
	Paramstore paramstypes.Subspace
}

func initMockKeeper(t *testing.T) (MockKeeper, sdk.Context) {
	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	mockStoreKey := sdk.NewKVStoreKey("mockkeeper")
	mockMemStoreKey := storetypes.NewMemoryStoreKey("mem_mockkeeper")
	stateStore.MountStoreWithDB(mockStoreKey, sdk.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(mockMemStoreKey, sdk.StoreTypeMemory, nil)

	require.NoError(t, stateStore.LoadLatestVersion())

	mockparamsSubspace := paramstypes.NewSubspace(cdc,
		codec.NewLegacyAmino(),
		mockStoreKey,
		mockMemStoreKey,
		"MockParams",
	)

	mockKeeper := MockKeeper{Cdc: cdc, StoreKey: mockStoreKey, MemKey: mockMemStoreKey, Paramstore: mockparamsSubspace}

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.TestingLogger())

	return mockKeeper, ctx
}

// Test addition and removal of a fixation entry
func TestFixationEntryAdditionAndRemoval(t *testing.T) {
	// create dummy data for dummy entry
	marshaledData := make([]byte, 8)
	binary.LittleEndian.PutUint64(marshaledData, 0)
	dummyIndex := "index"

	// init mockkeeper + context
	mockkeeper, ctx := initMockKeeper(t)

	// add dummy entry
	err := common.AddEntry(ctx, mockkeeper.StoreKey, "mock", "mockkeeper", mockkeeper.Cdc, dummyIndex, marshaledData)
	require.Nil(t, err)

	// get all entries with dummyIndex and make sure there is only one entry
	entryListFromStorage := common.GetAllEntriesForIndex(ctx, mockkeeper.StoreKey, "mock", mockkeeper.Cdc, dummyIndex)
	require.Equal(t, 1, len(entryListFromStorage))

	// make sure that one entry's data is the same data that was used to create it
	marshaledDataFromStorage := entryListFromStorage[0].MarshaledData
	require.Equal(t, marshaledData, marshaledDataFromStorage)

	// remove the entry
	common.RemoveEntry(ctx, mockkeeper.StoreKey, "mock", dummyIndex)

	// make sure there are no more entries with that index
	entryListFromStorage = common.GetAllEntriesForIndex(ctx, mockkeeper.StoreKey, "mock", mockkeeper.Cdc, dummyIndex)
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

	// init mockkeeper + context
	mockkeeper, ctx := initMockKeeper(t)

	// add the first dummy entry
	err := common.AddEntry(ctx, mockkeeper.StoreKey, "mock", "mockkeeper", mockkeeper.Cdc, dummyIndex, marshaledData)
	require.Nil(t, err)

	// add the second dummy entry
	err = common.AddEntry(ctx, mockkeeper.StoreKey, "mock", "mockkeeper", mockkeeper.Cdc, dummyIndex, marshaledData2)
	require.Nil(t, err)

	// get all entries with dummyIndex and make sure there is only one entry
	entryListFromStorage := common.GetAllEntriesForIndex(ctx, mockkeeper.StoreKey, "mock", mockkeeper.Cdc, dummyIndex)
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

	// init mockkeeper + context
	mockkeeper, ctx := initMockKeeper(t)

	// add the first dummy entry
	err := common.AddEntry(ctx, mockkeeper.StoreKey, "mock", "mockkeeper", mockkeeper.Cdc, dummyIndex, marshaledData)
	require.Nil(t, err)

	// advance a block
	ctx = ctx.WithBlockHeight(1)

	// add the second dummy entry
	err = common.AddEntry(ctx, mockkeeper.StoreKey, "mock", "mockkeeper", mockkeeper.Cdc, dummyIndex, marshaledData2)
	require.Nil(t, err)

	// get all entries with dummyIndex and make sure there are two entry
	entryListFromStorage := common.GetAllEntriesForIndex(ctx, mockkeeper.StoreKey, "mock", mockkeeper.Cdc, dummyIndex)
	require.Equal(t, 2, len(entryListFromStorage))

	// get the older version from block 0
	oldEntry, found := common.GetEntryOlderVersionByBlock(ctx, mockkeeper.StoreKey, "mock", mockkeeper.Cdc, dummyIndex, uint64(0))
	require.True(t, found)

	// verify the index and data matches the old entry from storage
	oldEntryIndex := common.CreateOldVersionIndex(dummyIndex, uint64(0))
	require.Equal(t, oldEntryIndex, oldEntry.Index)
	require.Equal(t, marshaledData, oldEntry.MarshaledData)
}
