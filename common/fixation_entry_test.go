package common_test

import (
	"encoding/binary"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common"
	"github.com/lavanet/lava/common/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmdb "github.com/tendermint/tm-db"
)

// Helper function to init a mock keeper and context
func initCtxAndVersionedStore(t *testing.T) (*common.VersionedStore, sdk.Context) {
	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	mockStoreKey := sdk.NewKVStoreKey("storeKey")
	mockMemStoreKey := storetypes.NewMemoryStoreKey("storeMemKey")
	stateStore.MountStoreWithDB(mockStoreKey, sdk.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(mockMemStoreKey, sdk.StoreTypeMemory, nil)

	require.NoError(t, stateStore.LoadLatestVersion())

	vs := common.NewVersionedStore(mockStoreKey, cdc)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.TestingLogger())

	return vs, ctx
}

// Test addition and removal of a fixation entry
func TestFixationEntryAdditionAndRemoval(t *testing.T) {
	// create dummy data for dummy entry
	marshaledData := make([]byte, 8)
	binary.LittleEndian.PutUint64(marshaledData, 1)
	dummyFixationKey := "dummyFix_"
	dummyIndex := "index"
	dummyObj := types.Entry{Index: dummyIndex, Block: 10, Data: marshaledData, References: 0}

	// init VersionedStore + context
	vs, ctx := initCtxAndVersionedStore(t)

	// add dummy entry
	blockToAddEntry := uint64(ctx.BlockHeight())
	err := vs.AppendEntry(ctx, dummyFixationKey, dummyObj.GetIndex(), blockToAddEntry, &dummyObj)
	require.Nil(t, err)

	// get all entry indices and make sure there is only one index
	indexList := vs.GetAllEntryIndices(ctx, dummyFixationKey)
	require.Equal(t, 1, len(indexList))

	// get the entry from the storage
	var tempEntry types.Entry
	err = vs.GetEntryForBlock(ctx, dummyFixationKey, dummyIndex, uint64(ctx.BlockHeight()), &tempEntry, types.DO_NOTHING)
	require.Nil(t, err)

	// make sure that one entry's data is the same data that was used to create it
	require.Equal(t, marshaledData, tempEntry.GetData())

	// remove the entry by advancing over the STALE_ENTRY_TIME and appending a new one (append triggers the removal func)
	ctx.WithBlockHeight(types.STALE_ENTRY_TIME + int64(blockToAddEntry) + 1)
	dummyObj2 := types.Entry{Index: dummyIndex, Block: 10, Data: marshaledData, References: 1}
	blockToAddEntry = uint64(ctx.BlockHeight())
	err = vs.AppendEntry(ctx, dummyFixationKey, dummyObj2.GetIndex(), blockToAddEntry, &dummyObj)
	require.Nil(t, err)

	// make sure there the old entry was deleted (check block)
	err = vs.GetEntryForBlock(ctx, dummyFixationKey, dummyIndex, dummyObj.Block, &tempEntry, types.SUB_REFERENCE)
	require.Equal(t, dummyObj2.Block, tempEntry.Block)

	// make sure dummy index is still in the entry index list
	indexList = vs.GetAllEntryIndices(ctx, dummyFixationKey)
	require.Equal(t, 1, len(indexList))
}

// Test that when adds two entries with the same block and index and makes sure that only the latest one is kept
func TestAdditionOfTwoEntriesWithSameIndexInSameBlock(t *testing.T) {
	// create dummy data for two dummy entries
	dummyIndex := "index"
	dummyFixationKey := "dummyFix_"
	dummyObj := sdk.Coin{Denom: "utest", Amount: sdk.ZeroInt()}
	dummyObj2 := sdk.Coin{Denom: "utest", Amount: sdk.OneInt()}

	// init VersionedStore + context
	vs, ctx := initCtxAndVersionedStore(t)

	byteCoin := vs.GetCdc().MustMarshal(&dummyObj)
	var tempCoin sdk.Coin
	vs.GetCdc().MustUnmarshal(byteCoin, &tempCoin)

	// add the first dummy entry
	blockToAddEntry := uint64(0)
	err := vs.AppendEntry(ctx, dummyFixationKey, dummyIndex, blockToAddEntry, &dummyObj)
	require.Nil(t, err)

	// add the second dummy entry
	err = vs.AppendEntry(ctx, dummyFixationKey, dummyIndex, blockToAddEntry, &dummyObj2)
	require.Nil(t, err)

	// get all entry indices and make sure there is only one index
	indexList := vs.GetAllEntryIndices(ctx, dummyFixationKey)
	require.Equal(t, 1, len(indexList))

	// get the entry from the storage
	var tempEntry types.Entry
	err = vs.GetEntryForBlock(ctx, dummyFixationKey, dummyIndex, blockToAddEntry, &tempEntry, types.DO_NOTHING)
	require.Nil(t, err)

	// make sure that one entry's data is the same data of the second dummy entry
	require.Equal(t, []byte{0xb}, tempEntry.GetData())

	// make sure dummy index is still in the entry index list
	indexList = vs.GetAllEntryIndices(ctx, dummyFixationKey)
	require.Equal(t, 1, len(indexList))
}

// // Test adding entry versions and getting an older version
// func TestEntryVersions(t *testing.T) {
// 	// create dummy data for two dummy entries
// 	marshaledData := make([]byte, 8)
// 	binary.LittleEndian.PutUint64(marshaledData, 0)
// 	marshaledData2 := make([]byte, 8)
// 	binary.LittleEndian.PutUint64(marshaledData2, 1)
// 	dummyIndex := "index"
// 	dummyFixationKey := "dummyFix_"
// 	dummyEntry := types.Entry{Index: "test", Block: 10, Data: marshaledData, References: 1}
// 	dummyEntry2 := types.Entry{Index: "test", Block: 20, Data: marshaledData2, References: 1}

// 	// init VersionedStore + context
// 	vs, ctx := initCtxAndVersionedStore(t)

// 	// add the first dummy entry
// 	err := vs.AppendEntry(ctx, dummyFixationKey, dummyIndex, dummyEntry.Block, &dummyEntry)
// 	require.Nil(t, err)

// 	// add the second dummy entry
// 	err = vs.AppendEntry(ctx, dummyFixationKey, dummyIndex, dummyEntry2.Block, &dummyEntry2)
// 	require.Nil(t, err)

// 	// get all entries with dummyIndex and make sure there are two entry
// 	var tempEntry types.Entry
// 	entries := make([]*types.Entry, 0)
// 	marshalers, err := vs.GetAllEntryVersionsByIndex(ctx, dummyFixationKey, dummyIndex, &tempEntry)
// 	for _, marshaler := range marshalers {
// 		entry := marshaler.(*types.Entry) // Type assertion to convert ProtoMarshaler to Entry
// 		entries = append(entries, entry)
// 	}
// 	require.Nil(t, err)
// 	require.Equal(t, 2, len(entries))

// 	// get the older version from block 0
// 	var oldEntry types.Entry
// 	found, err := vs.GetEntryForBlock(ctx, dummyFixationKey, dummyIndex, 13, &oldEntry, types.DO_NOTHING)
// 	require.True(t, found)
// 	require.Nil(t, err)

// 	// verify the block and data matches the old entry from storage
// 	require.Equal(t, 10, oldEntry.Block)
// 	require.Equal(t, marshaledData, oldEntry.Data)

// 	// make sure there is only one uniqueIndex object with this index
// 	uniqueIndices := vs.GetAllFixationEntryUniqueIndex(ctx, dummyFixationKey)
// 	require.Equal(t, 1, len(uniqueIndices))
// }

// // TODO: test sorting of entries when getting all, test that latest is really the latest
// // Test adding entry versions with different fixation keys
// func TestDifferentFixationKeys(t *testing.T) {
// 	// create dummy data for two dummy entries
// 	dummyIndex := "index"
// 	dummyFixationKey := "dummyFix_"
// 	dummyFixationKey2 := "dummyFix2_"
// 	dummyEntry := types.Entry{Index: "test", Block: 10, Data: []byte{0xa}, References: 1}
// 	dummyEntry2 := types.Entry{Index: "test", Block: 20, Data: []byte{0xb}, References: 1}

// 	// init VersionedStore + context
// 	vs, ctx := initCtxAndVersionedStore(t)

// 	// add the first dummy entry
// 	err := vs.AppendEntry(ctx, dummyFixationKey, dummyIndex, dummyEntry.Block, &dummyEntry)
// 	require.Nil(t, err)

// 	// add the second dummy entry
// 	err = vs.AppendEntry(ctx, dummyFixationKey2, dummyIndex, dummyEntry2.Block, &dummyEntry2)
// 	require.Nil(t, err)

// 	// get all entries with dummyIndex and make sure there is one entry
// 	var tempEntry types.Entry
// 	entries := make([]*types.Entry, 0)
// 	marshalers, err := vs.GetAllEntryVersionsByIndex(ctx, dummyFixationKey, dummyIndex, &tempEntry)
// 	for _, marshaler := range marshalers {
// 		entry := marshaler.(*types.Entry) // Type assertion to convert ProtoMarshaler to Entry
// 		entries = append(entries, entry)
// 	}
// 	require.Nil(t, err)
// 	require.Equal(t, 1, len(entries))

// 	// verify the block and data matches the old entry from storage
// 	require.Equal(t, uint64(10), entries[0].Block)
// 	require.Equal(t, []byte{0xa}, entries[0].Data)

// 	// make sure there is only one uniqueIndex object with this index
// 	uniqueIndices := vs.GetAllFixationEntryUniqueIndex(ctx, dummyFixationKey)
// 	require.Equal(t, 1, len(uniqueIndices))
// }

// func TestEntriesSort(t *testing.T) {
// 	// create dummy data for two dummy entries
// 	dummyIndex := "index"
// 	dummyFixationKey := "dummyFix_"
// 	dummyEntry := types.Entry{Index: "test", Block: 10, Data: []byte{0xa}, References: 1}
// 	dummyEntry2 := types.Entry{Index: "test", Block: 20, Data: []byte{0xb}, References: 1}
// 	dummyEntry3 := types.Entry{Index: "test", Block: 30, Data: []byte{0xc}, References: 1}

// 	// init VersionedStore + context
// 	vs, ctx := initCtxAndVersionedStore(t)

// 	// add the first dummy entry
// 	err := vs.AppendEntry(ctx, dummyFixationKey, dummyIndex, dummyEntry.Block, &dummyEntry)
// 	require.Nil(t, err)

// 	// add the second dummy entry
// 	err = vs.AppendEntry(ctx, dummyFixationKey, dummyIndex, dummyEntry2.Block, &dummyEntry2)
// 	require.Nil(t, err)

// 	// add the third dummy entry
// 	err = vs.AppendEntry(ctx, dummyFixationKey, dummyIndex, dummyEntry3.Block, &dummyEntry3)
// 	require.Nil(t, err)

// 	// get all entries with dummyIndex and make sure there is one entry
// 	var tempEntry types.Entry
// 	entries := make([]*types.Entry, 0)
// 	marshalers, err := vs.GetAllEntryVersionsByIndex(ctx, dummyFixationKey, dummyIndex, &tempEntry)
// 	for _, marshaler := range marshalers {
// 		entry := marshaler.(*types.Entry) // Type assertion to convert ProtoMarshaler to Entry
// 		entries = append(entries, entry)
// 	}
// 	require.Nil(t, err)
// 	require.Equal(t, 3, len(entries))

// 	// verify the entries are organized from oldest to latest (first element is oldest)
// 	proposedBlock := uint64(0)
// 	sorted := false
// 	for _, entry := range entries {
// 		if proposedBlock < entry.Block {
// 			sorted = true
// 		} else {
// 			sorted = false
// 		}
// 		proposedBlock = entry.Block
// 	}
// 	require.True(t, sorted)

// 	// verify the last element is the latest entry
// 	require.Equal(t, dummyEntry3.Block, proposedBlock)
// }
