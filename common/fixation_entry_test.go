package common_test

import (
	"strconv"
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
func initCtxAndFixationStores(t *testing.T, count int) ([]*common.FixationStore, sdk.Context) {
	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	mockStoreKey := sdk.NewKVStoreKey("storeKey")
	mockMemStoreKey := storetypes.NewMemoryStoreKey("storeMemKey")
	stateStore.MountStoreWithDB(mockStoreKey, sdk.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(mockMemStoreKey, sdk.StoreTypeMemory, nil)

	require.NoError(t, stateStore.LoadLatestVersion())

	fs := make([]*common.FixationStore, count)
	for i := 0; i < count; i++ {
		fixationKey := "mock_fix_" + strconv.Itoa(i)
		fs[i] = common.NewFixationStore(mockStoreKey, cdc, fixationKey)
	}

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.TestingLogger())

	return fs, ctx
}

func initCtxAndFixationStore(t *testing.T) (*common.FixationStore, sdk.Context) {
	fs, ctx := initCtxAndFixationStores(t, 1)
	return fs[0], ctx
}

// Test API calls with invalid entry index
func TestEntryInvalidIndex(t *testing.T) {
	dummyIndex := "index" + string('\001')
	dummyObj := sdk.Coin{Denom: "utest", Amount: sdk.ZeroInt()}

	vs, ctx := initCtxAndFixationStore(t)

	err := vs.AppendEntry(ctx, dummyIndex, uint64(ctx.BlockHeight()), &dummyObj)
	require.NotNil(t, err)

	err = vs.ModifyEntry(ctx, dummyIndex, uint64(ctx.BlockHeight()), &dummyObj)
	require.NotNil(t, err)

	err, found := vs.FindEntry(ctx, dummyIndex, uint64(ctx.BlockHeight()), &dummyObj)
	require.NotNil(t, err)
	require.False(t, found)

	err, found = vs.GetEntry(ctx, dummyIndex, &dummyObj)
	require.NotNil(t, err)
	require.False(t, found)
}

// Test addition and removal of a fixation entry
func TestFixationEntryAdditionAndRemoval(t *testing.T) {
	// create dummy data for dummy entry
	dummyIndex := "index"
	dummyObj := sdk.Coin{Denom: "utest", Amount: sdk.ZeroInt()}

	// init FixationStore + context
	vs, ctx := initCtxAndFixationStore(t)

	// add dummy entry
	firstEntryBlock := uint64(ctx.BlockHeight())
	err := vs.AppendEntry(ctx, dummyIndex, firstEntryBlock, &dummyObj)
	require.Nil(t, err)

	// get all entry indices and make sure there is only one index
	indexList := vs.GetAllEntryIndices(ctx)
	require.Equal(t, 1, len(indexList))

	// get the entry from the storage
	var dummyCoin sdk.Coin
	err, found := vs.FindEntry(ctx, dummyIndex, firstEntryBlock, &dummyCoin)
	require.Nil(t, err)
	require.True(t, found)

	// make sure that one entry's data is the same data that was used to create it
	require.True(t, dummyCoin.IsEqual(dummyObj))

	// advance the block height to +STALE_ENTRY_TIME (the entry should not be deleted yet)
	ctx = ctx.WithBlockHeight(types.STALE_ENTRY_TIME + ctx.BlockHeight())
	dummyObj2 := sdk.Coin{Denom: "utest", Amount: sdk.OneInt()}
	err = vs.AppendEntry(ctx, dummyIndex, uint64(ctx.BlockHeight()), &dummyObj2)
	require.Nil(t, err)

	// make sure the old entry was not deleted (check block)
	err, found = vs.FindEntry(ctx, dummyIndex, firstEntryBlock, &dummyCoin)
	require.True(t, found)
	require.Nil(t, err)

	// remove the entry by advancing over the STALE_ENTRY_TIME and appending a new one (append triggers the removal func)
	ctx = ctx.WithBlockHeight(types.STALE_ENTRY_TIME + ctx.BlockHeight() + 1)
	dummyObj3 := sdk.Coin{Denom: "utest", Amount: sdk.OneInt()}
	err = vs.AppendEntry(ctx, dummyIndex, uint64(ctx.BlockHeight()), &dummyObj3)
	require.Nil(t, err)

	// make sure the old entry was deleted (check block)
	err, found = vs.FindEntry(ctx, dummyIndex, firstEntryBlock, &dummyCoin)
	require.False(t, found)
	require.NotNil(t, err)

	// get the latest version and make sure it's equal to dummyObj3
	err, found = vs.FindEntry(ctx, dummyIndex, uint64(ctx.BlockHeight()), &dummyCoin)
	require.True(t, found)
	require.Nil(t, err)
	require.True(t, dummyCoin.IsEqual(dummyObj3))

	// make sure dummy index is still in the entry index list
	indexList = vs.GetAllEntryIndices(ctx)
	require.Equal(t, 1, len(indexList))
}

// Test that when adds two entries with the same block and index and makes sure that only the latest one is kept
func TestAdditionOfTwoEntriesWithSameIndexInSameBlock(t *testing.T) {
	// create dummy data for two dummy entries
	dummyIndex := "index"
	dummyObj := sdk.Coin{Denom: "utest", Amount: sdk.ZeroInt()}
	dummyObj2 := sdk.Coin{Denom: "utest", Amount: sdk.OneInt()}

	// init FixationStore + context
	vs, ctx := initCtxAndFixationStore(t)

	// add the first dummy entry
	blockToAddEntry := uint64(0)
	err := vs.AppendEntry(ctx, dummyIndex, blockToAddEntry, &dummyObj)
	require.Nil(t, err)

	// add the second dummy entry
	err = vs.AppendEntry(ctx, dummyIndex, blockToAddEntry, &dummyObj2)
	require.Nil(t, err)

	// get all entry indices and make sure there is only one index
	indexList := vs.GetAllEntryIndices(ctx)
	require.Equal(t, 1, len(indexList))

	// get the entry from the storage
	var dummyCoin sdk.Coin
	err, found := vs.FindEntry(ctx, dummyIndex, blockToAddEntry, &dummyCoin)
	require.True(t, found)
	require.Nil(t, err)

	// make sure that one entry's data is the same data of the second dummy entry
	require.True(t, dummyCoin.IsEqual(dummyObj2))

	// make sure dummy index is still in the entry index list
	indexList = vs.GetAllEntryIndices(ctx)
	require.Equal(t, 1, len(indexList))
}

// Test adding entry versions and getting an older version
func TestEntryVersions(t *testing.T) {
	// create dummy data for two dummy entries
	dummyIndex := "index"
	dummyObj := sdk.Coin{Denom: "utest", Amount: sdk.ZeroInt()}
	dummyObj2 := sdk.Coin{Denom: "utest", Amount: sdk.OneInt()}

	// init FixationStore + context
	vs, ctx := initCtxAndFixationStore(t)

	// add the first dummy entry
	blockToAddFirstEntry := int64(10)
	ctx = ctx.WithBlockHeight(blockToAddFirstEntry)
	err := vs.AppendEntry(ctx, dummyIndex, uint64(blockToAddFirstEntry), &dummyObj)
	require.Nil(t, err)

	// add the second dummy entry
	blockToAddSecondEntry := int64(20)
	ctx = ctx.WithBlockHeight(blockToAddSecondEntry)
	err = vs.AppendEntry(ctx, dummyIndex, uint64(blockToAddSecondEntry), &dummyObj2)
	require.Nil(t, err)

	// get the older version from block blockToAddFirstEntry
	var dummyCoin sdk.Coin
	err, found := vs.FindEntry(ctx, dummyIndex, uint64(blockToAddFirstEntry+1), &dummyCoin)
	require.Nil(t, err)
	require.True(t, found)

	// verify the data matches the old entry from storage
	require.True(t, dummyCoin.IsEqual(dummyObj))

	// make sure dummy index is still in the entry index list
	indexList := vs.GetAllEntryIndices(ctx)
	require.Equal(t, 1, len(indexList))
}

// Test adding non-visibility of a stale entry
func TestEntryStale(t *testing.T) {
	dummyIndex := "index"
	dummyObj1 := sdk.Coin{Denom: "utest", Amount: sdk.NewInt(0)}
	dummyObj2 := sdk.Coin{Denom: "utest", Amount: sdk.NewInt(1)}
	dummyObj3 := sdk.Coin{Denom: "utest", Amount: sdk.NewInt(2)}

	vs, ctx := initCtxAndFixationStore(t)

	block1 := int64(10)
	ctx = ctx.WithBlockHeight(block1)
	err := vs.AppendEntry(ctx, dummyIndex, uint64(block1), &dummyObj1)
	require.Nil(t, err)

	// bump refcount to avoid deletion later
	var dummyCoin sdk.Coin
	err, found := vs.GetEntry(ctx, dummyIndex, &dummyCoin)
	require.Nil(t, err)
	require.True(t, found)

	block2 := int64(20)
	ctx = ctx.WithBlockHeight(block2)
	err = vs.AppendEntry(ctx, dummyIndex, uint64(block2), &dummyObj2)
	require.Nil(t, err)

	block3 := int64(30 + types.STALE_ENTRY_TIME + 1)
	ctx = ctx.WithBlockHeight(block3)
	err = vs.AppendEntry(ctx, dummyIndex, uint64(block3), &dummyObj3)
	require.Nil(t, err)

	// the last AppendEntry should not have deleted the first entry (because
	// it has refcount != zero), and hence also not the second entry (though
	// it has refcount zero).

	err, found = vs.FindEntry(ctx, dummyIndex, uint64(block1+1), &dummyCoin)
	require.Nil(t, err)
	require.True(t, found)
	require.Equal(t, dummyCoin, dummyObj1)

	// But the second entry is now stale and therefore should not be visible

	err, found = vs.FindEntry(ctx, dummyIndex, uint64(block2+1), &dummyCoin)
	require.NotNil(t, err)
	require.False(t, found)

	// the third entry also has refcount = 0 and is old, but being the latest
	// it should also be visible always despite of refcount and age.

	err, found = vs.FindEntry(ctx, dummyIndex, uint64(block3+1), &dummyCoin)
	require.Nil(t, err)
	require.True(t, found)
}

// Test adding entry versions with different fixation keys
func TestDifferentFixationKeys(t *testing.T) {
	// create dummy data for two dummy entries
	dummyIndex := "index"
	dummyObj := sdk.Coin{Denom: "utest", Amount: sdk.ZeroInt()}
	dummyObj2 := sdk.Coin{Denom: "utest", Amount: sdk.OneInt()}

	// init FixationStore + context
	vs, ctx := initCtxAndFixationStores(t, 2)

	// add the first dummy entry
	blockToAddFirstEntry := int64(10)
	ctx = ctx.WithBlockHeight(blockToAddFirstEntry)
	err := vs[0].AppendEntry(ctx, dummyIndex, uint64(blockToAddFirstEntry), &dummyObj)
	require.Nil(t, err)

	// add the second dummy entry
	blockToAddSecondEntry := int64(20)
	ctx = ctx.WithBlockHeight(blockToAddSecondEntry)
	err = vs[1].AppendEntry(ctx, dummyIndex, uint64(blockToAddSecondEntry), &dummyObj2)
	require.Nil(t, err)

	// make sure there is one entry in the original fixation key storage
	indexList := vs[0].GetAllEntryIndices(ctx)
	require.Equal(t, 1, len(indexList))

	// verify the data matches the entry from original fixation key storage
	var dummyCoin sdk.Coin
	err, found := vs[0].GetEntry(ctx, dummyIndex, &dummyCoin)
	require.Nil(t, err)
	require.True(t, found)
	require.True(t, dummyCoin.IsEqual(dummyObj))
	require.False(t, dummyCoin.Equal(dummyObj2))

	// make sure there is one entry in the second fixation key storage
	indexList = vs[1].GetAllEntryIndices(ctx)
	require.Equal(t, 1, len(indexList))

	// verify the data matches the entry from original fixation key storage
	err, found = vs[1].FindEntry(ctx, dummyIndex, uint64(blockToAddSecondEntry), &dummyCoin)
	require.Nil(t, err)
	require.True(t, found)
	require.True(t, dummyCoin.IsEqual(dummyObj2))
	require.False(t, dummyCoin.Equal(dummyObj))

	// advance enough blocks so entries with refcount zero would be deleted
	ctx = ctx.WithBlockHeight(int64(blockToAddFirstEntry) + types.STALE_ENTRY_TIME + 1)

	// append to trigger delete function: expect first entry to remain because of
	// its refcount, and second entry because it is not old enough
	dummyObj3 := sdk.Coin{Denom: "utest", Amount: sdk.OneInt().Add(sdk.OneInt())}
	err = vs[0].AppendEntry(ctx, dummyIndex, uint64(ctx.BlockHeight()), &dummyObj3)
	require.Nil(t, err)

	// make sure the old entry was not deleted (check block)
	err, found = vs[0].FindEntry(ctx, dummyIndex, uint64(blockToAddFirstEntry), &dummyCoin)
	require.Nil(t, err)
	require.True(t, found)
	require.True(t, dummyCoin.IsEqual(dummyObj))

	// zero the refcount and advance one block
	err, found = vs[0].PutEntry(ctx, dummyIndex, uint64(blockToAddFirstEntry), &dummyCoin)
	require.Nil(t, err)
	require.True(t, found)
	require.True(t, dummyCoin.IsEqual(dummyObj))
	ctx = ctx.WithBlockHeight(int64(blockToAddFirstEntry) + types.STALE_ENTRY_TIME + 2)

	// append object to remove the first entry
	dummyObj4 := sdk.Coin{Denom: "utest", Amount: sdk.OneInt().Add(sdk.OneInt().Add(sdk.OneInt()))}
	blockToAddEntryForRemovalZeroRefCount := uint64(ctx.BlockHeight())
	err = vs[0].AppendEntry(ctx, dummyIndex, blockToAddEntryForRemovalZeroRefCount, &dummyObj4)
	require.Nil(t, err)

	// make sure the old entry was deleted (check block)
	err, found = vs[0].FindEntry(ctx, dummyIndex, uint64(blockToAddFirstEntry), &dummyCoin)
	require.NotNil(t, err)
	require.False(t, found)

	// make sure you cant subtract refs from entry with 0 refCount
	err, found = vs[0].PutEntry(ctx, dummyIndex, blockToAddEntryForRemovalZeroRefCount, &dummyCoin)
	require.NotNil(t, err)
	require.False(t, found)
}

// Test that the appended entries are sorted (first element is oldest)
func TestEntriesSort(t *testing.T) {
	// create dummy data for two dummy entries
	dummyIndex := "index"
	dummyObj := sdk.Coin{Denom: "utest", Amount: sdk.ZeroInt()}
	dummyObj2 := sdk.Coin{Denom: "utest", Amount: sdk.OneInt()}
	dummyObj3 := sdk.Coin{Denom: "utest", Amount: sdk.NewInt(2)}

	// init FixationStore + context
	vs, ctx := initCtxAndFixationStore(t)

	// add the first dummy entry
	blockToAddEntry := int64(10)
	ctx = ctx.WithBlockHeight(blockToAddEntry)
	err := vs.AppendEntry(ctx, dummyIndex, uint64(blockToAddEntry), &dummyObj)
	require.Nil(t, err)

	// add the second dummy entry
	blockToAddEntry += 10
	ctx = ctx.WithBlockHeight(blockToAddEntry)
	err = vs.AppendEntry(ctx, dummyIndex, uint64(blockToAddEntry), &dummyObj2)
	require.Nil(t, err)

	// add the third dummy entry
	blockToAddEntry += 10
	ctx = ctx.WithBlockHeight(blockToAddEntry)
	err = vs.AppendEntry(ctx, dummyIndex, uint64(blockToAddEntry), &dummyObj3)
	require.Nil(t, err)

	var resCoin sdk.Coin

	// find entry with >= highest block number
	err, found := vs.FindEntry(ctx, dummyIndex, 35, &resCoin)
	require.Nil(t, err)
	require.True(t, found)
	require.True(t, resCoin.IsEqual(dummyObj3))

	// find entry with >= middle block number
	err, found = vs.FindEntry(ctx, dummyIndex, 25, &resCoin)
	require.Nil(t, err)
	require.True(t, found)
	require.True(t, resCoin.IsEqual(dummyObj2))

	// find entry with >= lowest block number
	err, found = vs.FindEntry(ctx, dummyIndex, 15, &resCoin)
	require.Nil(t, err)
	require.True(t, found)
	require.True(t, resCoin.IsEqual(dummyObj))

	// fail to find entry with < lowest block number
	err, found = vs.FindEntry(ctx, dummyIndex, 5, &resCoin)
	require.NotNil(t, err)
	require.False(t, found)
}
