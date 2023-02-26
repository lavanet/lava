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
	blockToAddEntry := uint64(ctx.BlockHeight())
	err := vs.AppendEntry(ctx, dummyIndex, blockToAddEntry, &dummyObj)
	require.Nil(t, err)

	// get all entry indices and make sure there is only one index
	indexList := vs.GetAllEntryIndices(ctx)
	require.Equal(t, 1, len(indexList))

	// get the entry from the storage
	var dummyCoin sdk.Coin
	err = vs.GetEntry(ctx, dummyIndex, blockToAddEntry, &dummyCoin, types.DO_NOTHING)
	require.Nil(t, err)

	// make sure that one entry's data is the same data that was used to create it
	require.True(t, dummyCoin.IsEqual(dummyObj))

	// advance the block height to +STALE_ENTRY_TIME (the entry should not be deleted yet)
	ctx = ctx.WithBlockHeight(types.STALE_ENTRY_TIME + int64(blockToAddEntry))
	dummyObj2 := sdk.Coin{Denom: "utest", Amount: sdk.OneInt()}
	blockToAddEntryBeforeStale := uint64(ctx.BlockHeight())
	err = vs.AppendEntry(ctx, dummyIndex, blockToAddEntryBeforeStale, &dummyObj2)
	require.Nil(t, err)

	// make sure the old entry was not deleted (check block)
	err = vs.GetEntry(ctx, dummyIndex, blockToAddEntry, &dummyCoin, types.DO_NOTHING)
	require.Nil(t, err)

	// remove the entry by advancing over the STALE_ENTRY_TIME and appending a new one (append triggers the removal func)
	ctx = ctx.WithBlockHeight(types.STALE_ENTRY_TIME + int64(blockToAddEntry) + 1)
	dummyObj3 := sdk.Coin{Denom: "utest", Amount: sdk.OneInt()}
	blockToAddEntryAfterStale := uint64(ctx.BlockHeight())
	err = vs.AppendEntry(ctx, dummyIndex, blockToAddEntryAfterStale, &dummyObj3)
	require.Nil(t, err)

	// make sure the old entry was deleted (check block)
	err = vs.GetEntry(ctx, dummyIndex, blockToAddEntry, &dummyCoin, types.DO_NOTHING)
	require.NotNil(t, err)

	// get the latest version and make sure it's equal to dummyObj3
	err = vs.GetEntry(ctx, dummyIndex, blockToAddEntryAfterStale, &dummyCoin, types.DO_NOTHING)
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
	err = vs.GetEntry(ctx, dummyIndex, blockToAddEntry, &dummyCoin, types.DO_NOTHING)
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
	blockToAddFirstEntry := uint64(10)
	err := vs.AppendEntry(ctx, dummyIndex, blockToAddFirstEntry, &dummyObj)
	require.Nil(t, err)

	// add the second dummy entry
	blockToAddSecondEntry := uint64(20)
	err = vs.AppendEntry(ctx, dummyIndex, blockToAddSecondEntry, &dummyObj2)
	require.Nil(t, err)

	// get the older version from block blockToAddFirstEntry
	var dummyCoin sdk.Coin
	err = vs.GetEntry(ctx, dummyIndex, blockToAddFirstEntry+1, &dummyCoin, types.DO_NOTHING)
	require.Nil(t, err)

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
	vs2 := vs.WithPrefix("fix2")

	// add the first dummy entry
	blockToAddFirstEntry := uint64(10)
	err := vs.AppendEntry(ctx, dummyIndex, blockToAddFirstEntry, &dummyObj)
	require.Nil(t, err)

	// add the second dummy entry
	blockToAddSecondEntry := uint64(10)
	err = vs2.AppendEntry(ctx, dummyIndex, blockToAddSecondEntry, &dummyObj2)
	require.Nil(t, err)

	// get all indices with original fixation key and dummyIndex. make sure there is one entry
	indexList := vs.GetAllEntryIndices(ctx)
	require.Equal(t, 1, len(indexList))

	// verify the data matches the entry from original fixation key storage
	var dummyCoin sdk.Coin
	err = vs.GetEntry(ctx, dummyIndex, blockToAddFirstEntry, &dummyCoin, types.DO_NOTHING)
	require.Nil(t, err)
	require.True(t, dummyCoin.IsEqual(dummyObj))
	require.False(t, dummyCoin.Equal(dummyObj2))

	// get all indices with fix2 and dummyIndex. make sure there is one entry
	indexList = vs2.GetAllEntryIndices(ctx)
	require.Equal(t, 1, len(indexList))

	// verify the data matches the entry from original fixation key storage
	err = vs2.GetEntry(ctx, dummyIndex, blockToAddSecondEntry, &dummyCoin, types.DO_NOTHING)
	require.Nil(t, err)
	require.True(t, dummyCoin.IsEqual(dummyObj2))
	require.False(t, dummyCoin.Equal(dummyObj))

	// advance enough blocks so the entry with the regular fixation key will be deleted with a new append, but the second entry (with "fix2" key) won't be deleted
	ctx = ctx.WithBlockHeight(int64(blockToAddFirstEntry) + types.STALE_ENTRY_TIME + 1)

	// append object to remove the first entry
	dummyObj3 := sdk.Coin{Denom: "utest", Amount: sdk.OneInt().Add(sdk.OneInt())}
	blockToAddEntryForRemoval := uint64(ctx.BlockHeight())
	err = vs.AppendEntry(ctx, dummyIndex, blockToAddEntryForRemoval, &dummyObj3)
	require.Nil(t, err)

	// make sure the old entry was deleted (check block)
	err = vs.GetEntry(ctx, dummyIndex, blockToAddEntryForRemoval, &dummyCoin, types.DO_NOTHING)
	require.True(t, dummyCoin.IsEqual(dummyObj3))
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
	blockToAddEntry := uint64(10)
	err := vs.AppendEntry(ctx, dummyIndex, blockToAddEntry, &dummyObj)
	require.Nil(t, err)

	// add the second dummy entry
	blockToAddEntry += 10
	err = vs.AppendEntry(ctx, dummyIndex, blockToAddEntry, &dummyObj2)
	require.Nil(t, err)

	// add the third dummy entry
	blockToAddEntry += 10
	err = vs.AppendEntry(ctx, dummyIndex, blockToAddEntry, &dummyObj3)
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
