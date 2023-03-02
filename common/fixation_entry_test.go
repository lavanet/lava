package common_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	"github.com/cosmos/cosmos-sdk/store/prefix"
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
func initCtxAndFixationStore(t *testing.T) (*common.FixationStore, sdk.Context) {
	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	mockStoreKey := sdk.NewKVStoreKey("storeKey")
	mockMemStoreKey := storetypes.NewMemoryStoreKey("storeMemKey")
	stateStore.MountStoreWithDB(mockStoreKey, sdk.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(mockMemStoreKey, sdk.StoreTypeMemory, nil)

	require.NoError(t, stateStore.LoadLatestVersion())

	fixationKey := "mock_fix"

	vs := common.NewFixationStore(mockStoreKey, cdc, fixationKey)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.TestingLogger())

	return vs, ctx
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

// Test adding entry versions with different fixation keys
func TestDifferentFixationKeys(t *testing.T) {
	// create dummy data for two dummy entries
	dummyIndex := "index"
	dummyObj := sdk.Coin{Denom: "utest", Amount: sdk.ZeroInt()}
	dummyObj2 := sdk.Coin{Denom: "utest", Amount: sdk.OneInt()}

	// init FixationStore + context
	vs, ctx := initCtxAndFixationStore(t)
	vs2 := common.NewFixationStore(vs.GetStoreKey(), vs.GetCdc(), "fix2")

	// add the first dummy entry
	blockToAddFirstEntry := int64(10)
	ctx = ctx.WithBlockHeight(blockToAddFirstEntry)
	err := vs.AppendEntry(ctx, dummyIndex, uint64(blockToAddFirstEntry), &dummyObj)
	require.Nil(t, err)

	// add the second dummy entry
	blockToAddSecondEntry := int64(20)
	ctx = ctx.WithBlockHeight(blockToAddSecondEntry)
	err = vs2.AppendEntry(ctx, dummyIndex, uint64(blockToAddSecondEntry), &dummyObj2)
	require.Nil(t, err)

	// get all indices with original fixation key and dummyIndex. make sure there is one entry
	indexList := vs.GetAllEntryIndices(ctx)
	require.Equal(t, 1, len(indexList))

	// verify the data matches the entry from original fixation key storage
	var dummyCoin sdk.Coin
	err, found := vs.GetEntry(ctx, dummyIndex, uint64(blockToAddFirstEntry), &dummyCoin)
	require.Nil(t, err)
	require.True(t, found)
	require.True(t, dummyCoin.IsEqual(dummyObj))
	require.False(t, dummyCoin.Equal(dummyObj2))

	// get all indices with fix2 and dummyIndex. make sure there is one entry
	indexList = vs2.GetAllEntryIndices(ctx)
	require.Equal(t, 1, len(indexList))

	// verify the data matches the entry from original fixation key storage
	err, found = vs2.FindEntry(ctx, dummyIndex, uint64(blockToAddSecondEntry), &dummyCoin)
	require.Nil(t, err)
	require.True(t, found)
	require.True(t, dummyCoin.IsEqual(dummyObj2))
	require.False(t, dummyCoin.Equal(dummyObj))

	// advance enough blocks so the entry with the regular fixation key will be deleted with a new append, but the second entry (with "fix2" key) won't be deleted
	ctx = ctx.WithBlockHeight(int64(blockToAddFirstEntry) + types.STALE_ENTRY_TIME + 1)

	// append to trigger delete function and verify it's not deleted yet since RefCount = 1 (see L198)
	dummyObj3 := sdk.Coin{Denom: "utest", Amount: sdk.OneInt().Add(sdk.OneInt())}
	err = vs.AppendEntry(ctx, dummyIndex, uint64(ctx.BlockHeight()), &dummyObj3)
	require.Nil(t, err)

	// make sure the old entry was not deleted (check block)
	err, found = vs.FindEntry(ctx, dummyIndex, uint64(blockToAddFirstEntry), &dummyCoin)
	require.Nil(t, err)
	require.True(t, found)
	require.True(t, dummyCoin.IsEqual(dummyObj))

	// zero the refcount and advance one block
	err, found = vs.PutEntry(ctx, dummyIndex, uint64(blockToAddFirstEntry), &dummyCoin)
	require.Nil(t, err)
	require.True(t, found)
	require.True(t, dummyCoin.IsEqual(dummyObj))
	ctx = ctx.WithBlockHeight(int64(blockToAddFirstEntry) + types.STALE_ENTRY_TIME + 2)

	// append object to remove the first entry
	dummyObj4 := sdk.Coin{Denom: "utest", Amount: sdk.OneInt().Add(sdk.OneInt().Add(sdk.OneInt()))}
	blockToAddEntryForRemovalZeroRefCount := uint64(ctx.BlockHeight())
	err = vs.AppendEntry(ctx, dummyIndex, blockToAddEntryForRemovalZeroRefCount, &dummyObj4)
	require.Nil(t, err)

	// make sure the old entry was deleted (check block)
	err, found = vs.FindEntry(ctx, dummyIndex, uint64(blockToAddFirstEntry), &dummyCoin)
	require.NotNil(t, err)
	require.False(t, found)

	// make sure you cant subtract refs from entry with 0 refCount
	err, found = vs.PutEntry(ctx, dummyIndex, blockToAddEntryForRemovalZeroRefCount, &dummyCoin)
	require.NotNil(t, err)
	require.False(t, found)
}

// Test that the appended entries are sorted (first element is oldest)
func TestEntriesSort(t *testing.T) {
	// create dummy data for two dummy entries
	dummyIndex := "index"
	dummyObj := sdk.Coin{Denom: "utest", Amount: sdk.ZeroInt()}
	dummyObj2 := sdk.Coin{Denom: "utest", Amount: sdk.OneInt()}
	dummyObj3 := sdk.Coin{Denom: "utest", Amount: sdk.OneInt().Mul(sdk.NewIntFromUint64(2))}

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

	// get the relevant store and init an iterator and verify the entries are organized from oldest to latest (first element is oldest)
	store := prefix.NewStore(ctx.KVStore(vs.GetStoreKey()), types.KeyPrefix(types.EntryKey+vs.GetPrefix()+dummyIndex))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	// iterate over entries
	proposedBlock := uint64(0)
	sorted := false
	var oldEntry types.Entry
	for ; iterator.Valid(); iterator.Next() {
		// umarshal the old entry version
		vs.GetCdc().MustUnmarshal(iterator.Value(), &oldEntry)

		// proposedBlock should always be smaller than the next entry's block if the elements in the store are sorted
		if proposedBlock < oldEntry.Block {
			sorted = true
		} else {
			sorted = false
		}
		proposedBlock = oldEntry.Block
	}
	require.True(t, sorted)

	// verify the last element is the latest entry
	require.Equal(t, oldEntry.Block, proposedBlock)
}
