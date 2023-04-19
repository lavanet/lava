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

type template struct {
	op    string
	name  string
	store int
	index string
	coin  int
	fail  bool
	count int64
}

// helper to automate testing operations
func testWithTemplate(t *testing.T, playbook []template, countObj int, countVS int) {
	vs, ctx := initCtxAndFixationStores(t, countVS)

	var coins []sdk.Coin
	var dummy sdk.Coin

	for i := 0; i < countObj; i++ {
		coins = append(coins, sdk.Coin{Denom: "utest", Amount: sdk.NewInt(int64(i+1))})
	}

	for _, play := range playbook {
		block := uint64(ctx.BlockHeight())
		if play.count != 0 {
			block = uint64(play.count)
		}
		index := "myindex"
		if play.index != "" {
			index = play.index
		}
		what := play.op + " " + play.name +
			" index: " + index +
			" block: " + strconv.Itoa(int(block))
		switch play.op {
		case "append":
			if block > uint64(ctx.BlockHeight()) {
				ctx = ctx.WithBlockHeight(int64(block))
			}
			err := vs[play.store].AppendEntry(ctx, play.index, block, &coins[play.coin])
			if !play.fail {
				require.Nil(t, err, what)
			} else {
				require.NotNil(t, err, what)
			}
		case "modify":
			err := vs[play.store].ModifyEntry(ctx, play.index, block, &coins[play.coin])
			if !play.fail {
				require.Nil(t, err, what)
			} else {
				require.NotNil(t, err, what)
			}
		case "find":
			found := vs[play.store].FindEntry(ctx, play.index, block, &dummy)
			if !play.fail {
				require.True(t, found, what)
				require.Equal(t, dummy, coins[play.coin], what)
			} else {
				require.False(t, found, what)
			}
		case "get":
			found := vs[play.store].GetEntry(ctx, play.index, &dummy)
			if !play.fail {
				require.True(t, found, what)
				require.Equal(t, dummy, coins[play.coin], what)
			} else {
				require.False(t, found, what)
			}
		case "put":
			vs[play.store].PutEntry(ctx, play.index, block)
		case "block":
			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + play.count)
		case "getall":
			indexList := vs[play.store].GetAllEntryIndices(ctx)
			require.Equal(t, int(play.count), len(indexList), what)
		case "getallprefix":
			indexList := vs[play.store].GetAllEntryIndicesWithPrefix(ctx, index)
			require.Equal(t, int(play.count), len(indexList), what)
		}
	}
}

// Test API calls with invalid entry index
func TestEntryInvalidIndex(t *testing.T) {
	invalid := "index" + string('\001')

	playbook := []template{
		{ op: "append", name: "with invalid index (fail)", index: invalid, fail: true },
		{ op: "modify", name: "with invalid index (fail)", index: invalid, fail: true },
		{ op: "find", name: "with invalid index (fail)", index: invalid, fail: true },
		{ op: "get", name: "with invalid index (fail)", index: invalid, fail: true },
	}

	testWithTemplate(t, playbook, 3, 1)
}

// Test addition and auto-removal of a fixation entry
func TestFixationEntryAdditionAndRemoval(t *testing.T) {
	block0 := int64(10)
	block1 := block0 + types.STALE_ENTRY_TIME + 1

	playbook := []template{
		{ op: "append", name: "entry #1", count: block0, coin: 0 },
		{ op: "find", name: "entry #1", count: block0, coin: 0 },
		{ op: "getall", name: "to check exactly one index", count: 1 },
		{ op: "append", name: "entry #2", count: block1, coin: 1 },
		// entry #1 not deleted because not enough time with refcount = zero
		{ op: "find", name: "entry #1 (not stale yet)", count: block0 },
		{ op: "block", name: "add STAEL_ENTRY_TIME+1", count: types.STALE_ENTRY_TIME+1 },
		// entry #1 now deleted because blocks advanced by STALE_ENTRY_TIME+1
		{ op: "find", name: "entry #1 (now stale/gone)", count: block0, fail: true },
		{ op: "find", name: "latest entry", coin: 1 },
		{ op: "getall", name: "to check again exactly one index", count: 1 },
	}

	testWithTemplate(t, playbook, 2, 1)
}

// Test addition of same entry twice within the same block
func TestAdditionOfTwoEntriesWithSameIndexInSameBlock(t *testing.T) {
	block0 := int64(10)

	playbook := []template{
		{ op: "append", name: "entry #1", count: block0, coin: 0 },
		{ op: "append", name: "entry #2", count: block0, coin: 1 },
		{ op: "getall", name: "to check exactly one index", count: 1 },
		{ op: "find", name: "entry #2", count: block0, coin: 1 },
	}

	testWithTemplate(t, playbook, 2, 1)
}

// Test adding entry versions and getting an older version
func TestEntryVersions(t *testing.T) {
	block0 := int64(10)
	block1 := block0 + int64(10)

	playbook := []template{
		{ op: "append", name: "entry #1", count: block0, coin: 0 },
		{ op: "append", name: "entry #2", count: block1, coin: 1 },
		{ op: "find", name: "entry #1", count: block0, coin: 0 },
		{ op: "getall", name: "to check exactly one index", count: 1 },
	}

	testWithTemplate(t, playbook, 2, 1)
}

// Test non-visibility of a stale entry
func TestEntryStale(t *testing.T) {
	block0 := int64(10)
	block1 := block0 + int64(10)
	block2 := block1 + int64(10) + types.STALE_ENTRY_TIME+1

	playbook := []template{
		{ op: "append", name: "entry #1", count: block0, coin: 0 },
		{ op: "get", name: "refcount entry #1", count: block0, coin: 0 },
		{ op: "append", name: "entry #2", count: block1, coin: 1 },
		{ op: "append", name: "entry #3", count: block2, coin: 2 },
		// entry #1 should not be deleted because it has refcount != zero);
		// entry #2 (refcount = zero) also not deleted because it is not oldest
		{ op: "find", name: "entry #1", count: block0+1, coin: 0 },
		{ op: "find", name: "entry #2", count: block1+1, coin: 1 },
		{ op: "block", name: "add STAEL_ENTRY_TIME+1", count: types.STALE_ENTRY_TIME+1 },
		// entry #2 now stale and therefore should not be visible
		{ op: "find", name: "entry #2", count: block1+1, fail: true },
		// entry #3 (refcount = zero) is old, but being the latest it always
		// remains visible (despite of refcount and age).
		{ op: "find", name: "entry #3", count: block2+1, coin: 2 },
	}

	testWithTemplate(t, playbook, 3, 1)
}

// Test adding entry versions with different fixation keys
func TestDifferentFixationKeys(t *testing.T) {
	block0 := int64(10)
	block1 := block0 + int64(10)
	block2 := block1 + types.STALE_ENTRY_TIME+1

	playbook := []template{
		{ op: "append", name: "entry #1 (store #1)", store: 0, count: block0, coin: 0 },
		{ op: "append", name: "entry #1 (store #2)", store: 1, count: block1, coin: 1 },
		{ op: "getall", name: "for exactly one index (store #1)", store: 0, count: 1 },
		{ op: "getall", name: "for exactly one index (store #2)", store: 1, count: 1 },
		{ op: "find", name: "entry #1 (store #1)", store: 0, count: block0, coin: 0 },
		{ op: "find", name: "entry #2 (store #2)", store: 1, count: block1, coin: 1 },
		{ op: "append", name: "entry #3 (store #1)", store: 0, count: block2, coin: 2 },
		// entry #1 not deleted because not enough time with refcount = zero
		{ op: "find", name: "entry #1 (store #1)", store: 0, count: block0, coin: 0 },
		{ op: "block", name: "add STAEL_ENTRY_TIME+1", count: types.STALE_ENTRY_TIME+1 },
		// entry #1 now deleted because blocks advanced by STALE_ENTRY_TIME+1
		// entry #2 in store#2 remains unaffected
		{ op: "find", name: "entry #1 (store #1)", store: 0, count: block0, fail: true },
		{ op: "find", name: "entry #2 (store #2)", store: 1, count: block1, coin: 1 },
	}

	testWithTemplate(t, playbook, 3, 2)
}

func TestGetAndPutEntry(t *testing.T) {
	block0 := int64(10)
	block1 := block0 + types.STALE_ENTRY_TIME+1
	block2 := block1 + types.STALE_ENTRY_TIME+1

	playbook := []template{
		{ op: "append", name: "entry #1", count: block0, coin: 0 },
		{ op: "get", name: "refcount entry #1", coin: 0 },
		{ op: "append", name: "entry #2", count: block1, coin: 1 },
		// entry #1 should not be deleted because it has refcount != zero);
		{ op: "find", name: "entry #1", count: block0, coin: 0 },
		{ op: "put", name: "refcount entry #1", count: block0 },
		// entry #1 not deleted because not enough time with refcount = zero
		{ op: "find", name: "entry #1", count: block0, coin: 0 },
		{ op: "append", name: "entry #3", count: block2, coin: 2 },
		// entry #1 now deleted because blocks advanced by STALE_ENTRY_TIME+1
		{ op: "find", name: "entry #1", count: block0, fail: true },
	}

	testWithTemplate(t, playbook, 3, 1)
}

func TestDeleteTwoEntries(t *testing.T) {
	block0 := int64(10)
	block1 := block0 + int64(10)
	block2 := block1 + int64(10)
	block3 := block2 + types.STALE_ENTRY_TIME+1

	playbook := []template{
		{ op: "append", name: "entry #1", count: block0, coin: 0 },
		{ op: "append", name: "entry #2", count: block1, coin: 1 },
		{ op: "append", name: "entry #3", count: block2, coin: 2 },
		// this update triggers deletion of entry #1, #2
		{ op: "append", name: "entry #4", count: block3, coin: 3 },
		{ op: "find", name: "entry #1", count: block0, fail: true },
		{ op: "find", name: "entry #2", count: block1, fail: true },
	}

	testWithTemplate(t, playbook, 4, 1)
}

// Test that the appended entries are sorted (first element is oldest)
func TestEntriesSort(t *testing.T) {
	block0 := int64(10)
	block1 := block0 + int64(10)
	block2 := block1 + int64(10)

	playbook := []template{
		{ op: "append", name: "entry #1", count: block0, coin: 0 },
		{ op: "append", name: "entry #2", count: block1, coin: 1 },
		{ op: "append", name: "entry #3", count: block2, coin: 2 },
		{ op: "find", name: "entry #3", count: block2+int64(5), coin: 2 },
		{ op: "find", name: "entry #2", count: block1+int64(5), coin: 1 },
		{ op: "find", name: "entry #1", count: block0+int64(5), coin: 0 },
		{ op: "find", name: "no entry", count: block0-int64(5), fail: true },
	}

	testWithTemplate(t, playbook, 3, 1)
}

func TestGetAllEntries(t *testing.T) {
	block0 := int64(10)

	playbook := []template{
		{ op: "append", name: "entry #1", index: "prefix1_a", count: block0, coin: 0 },
		{ op: "append", name: "entry #1", index: "prefix1_b", count: block0, coin: 1 },
		{ op: "append", name: "entry #1", index: "prefix1_c", count: block0, coin: 2 },
		{ op: "append", name: "entry #1", index: "prefix2_a", count: block0, coin: 3 },
		{ op: "append", name: "entry #1", index: "prefix2_b", count: block0, coin: 4 },
		{ op: "append", name: "entry #1", index: "prefix3_a", count: block0, coin: 5 },
		{ op: "getall", name: "to check all indices", count: 6 },
		{ op: "getallprefix", name: "to check all indices with prefix", index: "prefix", count: 6 },
		{ op: "getallprefix", name: "to check indices with prefix1", index: "prefix1", count: 3 },
		{ op: "getallprefix", name: "to check indices with prefix2", index: "prefix2", count: 2 },
		{ op: "getallprefix", name: "to check indices with prefix3", index: "prefix3", count: 1 },
	}

	testWithTemplate(t, playbook, 6, 1)
}
