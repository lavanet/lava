package common

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common/types"
	"github.com/stretchr/testify/require"
)

func initCtxAndFixationStores(t *testing.T, count int) (sdk.Context, []*FixationStore) {
	ctx, cdc := initCtx(t)

	fs := make([]*FixationStore, count)
	for i := 0; i < count; i++ {
		fixationKey := "mock_fix_" + strconv.Itoa(i)
		fs[i] = NewFixationStore(mockStoreKey, cdc, fixationKey)
	}

	return ctx, fs
}

func initCtxAndFixationStore(t *testing.T) (sdk.Context, *FixationStore) {
	ctx, fs := initCtxAndFixationStores(t, 1)
	return ctx, fs[0]
}

type fixationTemplate struct {
	op    string
	name  string
	store int
	index string
	coin  int
	fail  bool
	count int64
}

// helper to automate testing operations
func testWithFixationTemplate(t *testing.T, playbook []fixationTemplate, countObj int, countVS int) {
	ctx, fs := initCtxAndFixationStores(t, countVS)

	var coins []sdk.Coin
	var dummy sdk.Coin

	for i := 0; i < countObj; i++ {
		coins = append(coins, sdk.Coin{Denom: "utest", Amount: sdk.NewInt(int64(i + 1))})
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
			err := fs[play.store].AppendEntry(ctx, index, block, &coins[play.coin])
			if !play.fail {
				require.Nil(t, err, what)
			} else {
				require.NotNil(t, err, what)
			}
		case "modify":
			err := fs[play.store].ModifyEntry(ctx, index, block, &coins[play.coin])
			if !play.fail {
				require.Nil(t, err, what)
			} else {
				require.NotNil(t, err, what)
			}
		case "find":
			found := fs[play.store].FindEntry(ctx, index, block, &dummy)
			if !play.fail {
				require.True(t, found, what)
				require.Equal(t, dummy, coins[play.coin], what)
			} else {
				require.False(t, found, what)
			}
		case "get":
			found := fs[play.store].GetEntry(ctx, index, &dummy)
			if !play.fail {
				require.True(t, found, what)
				require.Equal(t, dummy, coins[play.coin], what)
			} else {
				require.False(t, found, what)
			}
		case "put":
			fs[play.store].PutEntry(ctx, index, block)
		case "block":
			ctx = ctx.WithBlockHeight(ctx.BlockHeight() + play.count)
			fs[play.store].AdvanceBlock(ctx)
		case "getall":
			indexList := fs[play.store].GetAllEntryIndices(ctx)
			require.Equal(t, int(play.count), len(indexList), what)
		case "getvers":
			indexList := fs[play.store].GetAllEntryVersions(ctx, index, true)
			require.Equal(t, int(play.count), len(indexList), what)
		case "getallprefix":
			indexList := fs[play.store].GetAllEntryIndicesWithPrefix(ctx, index)
			require.Equal(t, int(play.count), len(indexList), what)
		}
	}
}

// Test API calls with invalid entry index
func TestEntryInvalidIndex(t *testing.T) {
	invalid := "index" + string('\001')

	playbook := []fixationTemplate{
		{op: "append", name: "with invalid index (fail)", index: invalid, fail: true},
		{op: "modify", name: "with invalid index (fail)", index: invalid, fail: true},
		{op: "find", name: "with invalid index (fail)", index: invalid, fail: true},
		{op: "get", name: "with invalid index (fail)", index: invalid, fail: true},
	}

	testWithFixationTemplate(t, playbook, 3, 1)
}

// Test addition and auto-removal of a fixation entry
func TestFixationEntryAdditionAndRemoval(t *testing.T) {
	block0 := int64(10)
	block1 := block0 + types.STALE_ENTRY_TIME + 1

	playbook := []fixationTemplate{
		{op: "append", name: "entry #1", count: block0, coin: 0},
		{op: "find", name: "entry #1", count: block0, coin: 0},
		{op: "getall", name: "to check exactly one index", count: 1},
		{op: "append", name: "entry #2", count: block1, coin: 1},
		// entry #1 not deleted because not enough time with refcount = zero
		{op: "find", name: "entry #1 (not stale yet)", count: block0},
		{op: "block", name: "add STAEL_ENTRY_TIME+1", count: types.STALE_ENTRY_TIME + 1},
		// entry #1 now deleted because blocks advanced by STALE_ENTRY_TIME+1
		{op: "find", name: "entry #1 (now stale/gone)", count: block0, fail: true},
		{op: "find", name: "latest entry", coin: 1},
		{op: "getall", name: "to check again exactly one index", count: 1},
	}

	testWithFixationTemplate(t, playbook, 2, 1)
}

// Test addition of same entry twice within the same block
func TestAdditionOfTwoEntriesWithSameIndexInSameBlock(t *testing.T) {
	block0 := int64(10)

	playbook := []fixationTemplate{
		{op: "append", name: "entry #1", count: block0, coin: 0},
		{op: "append", name: "entry #2", count: block0, coin: 1},
		{op: "getall", name: "to check exactly one index", count: 1},
		{op: "find", name: "entry #2", count: block0, coin: 1},
	}

	testWithFixationTemplate(t, playbook, 2, 1)
}

// Test adding entry versions and getting an older version
func TestEntryVersions(t *testing.T) {
	block0 := int64(10)
	block1 := block0 + int64(10)

	playbook := []fixationTemplate{
		{op: "append", name: "entry #1", count: block0, coin: 0},
		{op: "append", name: "entry #2", count: block1, coin: 1},
		{op: "find", name: "entry #1", count: block0, coin: 0},
		{op: "getall", name: "to check exactly one index", count: 1},
	}

	testWithFixationTemplate(t, playbook, 2, 1)
}

// Test non-visibility of a stale entry
func TestEntryStale(t *testing.T) {
	block0 := int64(10)
	block1 := block0 + int64(10)
	block2 := block1 + int64(10) + types.STALE_ENTRY_TIME + 1

	playbook := []fixationTemplate{
		{op: "append", name: "entry #1", count: block0, coin: 0},
		{op: "get", name: "refcount entry #1", count: block0, coin: 0},
		{op: "append", name: "entry #2", count: block1, coin: 1},
		{op: "append", name: "entry #3", count: block2, coin: 2},
		// entry #1 should not be deleted because it has refcount != zero);
		// entry #2 (refcount = zero) also not deleted because it is not oldest
		{op: "find", name: "entry #1", count: block0 + 1, coin: 0},
		{op: "find", name: "entry #2", count: block1 + 1, coin: 1},
		{op: "block", name: "add STAEL_ENTRY_TIME+1", count: types.STALE_ENTRY_TIME + 1},
		// entry #2 now stale and therefore should not be visible
		{op: "find", name: "entry #2", count: block1 + 1, fail: true},
		// entry #3 (refcount = zero) is old, but being the latest it always
		// remains visible (despite of refcount and age).
		{op: "find", name: "entry #3", count: block2 + 1, coin: 2},
	}

	testWithFixationTemplate(t, playbook, 3, 1)
}

// Test adding entry versions with different fixation keys
func TestDifferentFixationKeys(t *testing.T) {
	block0 := int64(10)
	block1 := block0 + int64(10)
	block2 := block1 + types.STALE_ENTRY_TIME + 1

	playbook := []fixationTemplate{
		{op: "append", name: "entry #1 (store #1)", store: 0, count: block0, coin: 0},
		{op: "append", name: "entry #1 (store #2)", store: 1, count: block1, coin: 1},
		{op: "getall", name: "for exactly one index (store #1)", store: 0, count: 1},
		{op: "getall", name: "for exactly one index (store #2)", store: 1, count: 1},
		{op: "find", name: "entry #1 (store #1)", store: 0, count: block0, coin: 0},
		{op: "find", name: "entry #2 (store #2)", store: 1, count: block1, coin: 1},
		{op: "append", name: "entry #3 (store #1)", store: 0, count: block2, coin: 2},
		// entry #1 not deleted because not enough time with refcount = zero
		{op: "find", name: "entry #1 (store #1)", store: 0, count: block0, coin: 0},
		{op: "block", name: "add STAEL_ENTRY_TIME+1", count: types.STALE_ENTRY_TIME + 1},
		// entry #1 now deleted because blocks advanced by STALE_ENTRY_TIME+1
		// entry #2 in store#2 remains unaffected
		{op: "find", name: "entry #1 (store #1)", store: 0, count: block0, fail: true},
		{op: "find", name: "entry #2 (store #2)", store: 1, count: block1, coin: 1},
	}

	testWithFixationTemplate(t, playbook, 3, 2)
}

func TestGetAndPutEntry(t *testing.T) {
	block0 := int64(10)
	block1 := block0 + types.STALE_ENTRY_TIME + 1
	block2 := block1 + types.STALE_ENTRY_TIME + 1

	playbook := []fixationTemplate{
		{op: "append", name: "entry #1", count: block0, coin: 0},
		{op: "get", name: "refcount entry #1", coin: 0},
		{op: "append", name: "entry #2", count: block1, coin: 1},
		// entry #1 should not be deleted because it has refcount != zero);
		{op: "find", name: "entry #1", count: block0, coin: 0},
		{op: "put", name: "refcount entry #1", count: block0},
		// entry #1 not deleted because not enough time with refcount = zero
		{op: "find", name: "entry #1", count: block0, coin: 0},
		{op: "append", name: "entry #3", count: block2, coin: 2},
		// entry #1 now deleted because blocks advanced by STALE_ENTRY_TIME+1
		{op: "find", name: "entry #1", count: block0, fail: true},
	}

	testWithFixationTemplate(t, playbook, 3, 1)
}

func TestDoublePutEntry(t *testing.T) {
	block0 := int64(10)
	block1 := block0 + types.STALE_ENTRY_TIME + 1

	playbook := []fixationTemplate{
		{op: "append", name: "entry #1 version 0", count: block0, coin: 0},
		{op: "append", name: "entry #1 version 1", count: block1, coin: 0},
		// entry #1 with block zero now has refcount = zero
		{op: "put", name: "negative refcount entry #1 version 0", count: block0, fail: false},
	}

	require.Panics(t, func() { testWithFixationTemplate(t, playbook, 3, 1) })
}

func TestDeleteTwoEntries(t *testing.T) {
	block0 := int64(10)
	block1 := block0 + int64(10)
	block2 := block1 + int64(10)
	block3 := block2 + types.STALE_ENTRY_TIME + 1

	playbook := []fixationTemplate{
		{op: "append", name: "entry #1", count: block0, coin: 0},
		{op: "append", name: "entry #2", count: block1, coin: 1},
		{op: "append", name: "entry #3", count: block2, coin: 2},
		// this update triggers deletion of entry #1, #2
		{op: "append", name: "entry #4", count: block3, coin: 3},
		{op: "find", name: "entry #1", count: block0, fail: true},
		{op: "find", name: "entry #2", count: block1, fail: true},
	}

	testWithFixationTemplate(t, playbook, 4, 1)
}

func TestRemoveStaleEntries(t *testing.T) {
	block0 := int64(10)
	block1 := block0 + int64(10)
	block2 := block1 + int64(10)
	block3 := block2 + int64(10)
	block4 := block3 + int64(10)
	block5 := int64(100)
	block6 := block5 + types.STALE_ENTRY_TIME
	block7 := block6 + types.STALE_ENTRY_TIME/2
	block8 := block7 + types.STALE_ENTRY_TIME/2 + 1
	block9 := block8 + types.STALE_ENTRY_TIME/2 + 2

	playbook := []fixationTemplate{
		{op: "append", name: "entry #1", count: block0, coin: 0},
		{op: "get", name: "refcount entry #1", coin: 0},
		{op: "append", name: "entry #2", count: block1, coin: 1},
		{op: "get", name: "refcount entry #2", coin: 1},
		{op: "append", name: "entry #3", count: block2, coin: 2},
		{op: "get", name: "refcount entry #3", coin: 2},
		{op: "append", name: "entry #4", count: block3, coin: 3},
		{op: "get", name: "refcount entry #4", coin: 3},
		{op: "append", name: "entry #5", count: block4, coin: 4},
		{op: "get", name: "refcount entry #5", coin: 4},
		// release an entry
		{op: "block", name: "advance a bit", count: block5 - block4},
		{op: "put", name: "refcount entry #1", count: block0},
		{op: "block", name: "wait entry #1 staled", count: block6 - block5},
		// expect 5 entry versions left
		{op: "getvers", name: "to check 5 versions left", count: 4},
		// release more entries
		{op: "put", name: "refcount entry #4", count: block3},
		{op: "block", name: "wait entry #1 half staled", count: block7 - block6},
		{op: "put", name: "refcount entry #3", count: block2},
		{op: "block", name: "wait another #1 half staled", count: block8 - block7},
		// entry #4 is stale but un-removable because entry #3 sill alive
		{op: "getvers", name: "to check 4 versions remain", count: 4},
		{op: "block", name: "wait another #1 half staled", count: block9 - block8},
		// entry #3 became stale, so (stale) entry #4 was removed
		{op: "getvers", name: "to check 3 versions remain", count: 3},
	}

	testWithFixationTemplate(t, playbook, 5, 1)
}

func TestRemoveLastEntry(t *testing.T) {
	block0 := int64(10)
	block1 := block0 + int64(10)
	block2 := block1 + types.STALE_ENTRY_TIME

	playbook := []fixationTemplate{
		{op: "append", name: "entry #1", count: block0, coin: 0},
		{op: "block", name: "advance a bit", count: block1 - block0},
		{op: "put", name: "refcount entry #1", count: block0},
		// expect 1 (stale) entry versions left
		{op: "getvers", name: "to check 1 versions left", count: 1},
		{op: "block", name: "wait for entry #1 stale", count: block2 - block1},
		// expect entry #1 gone now
		{op: "find", name: "try to find entry #1", count: block0, coin: 0, fail: true},
	}

	testWithFixationTemplate(t, playbook, 1, 1)
}

// Test that the appended entries are sorted (first element is oldest)
func TestEntriesSort(t *testing.T) {
	block0 := int64(10)
	block1 := block0 + int64(10)
	block2 := block1 + int64(10)

	playbook := []fixationTemplate{
		{op: "append", name: "entry #1", count: block0, coin: 0},
		{op: "append", name: "entry #2", count: block1, coin: 1},
		{op: "append", name: "entry #3", count: block2, coin: 2},
		{op: "find", name: "entry #3", count: block2 + int64(5), coin: 2},
		{op: "find", name: "entry #2", count: block1 + int64(5), coin: 1},
		{op: "find", name: "entry #1", count: block0 + int64(5), coin: 0},
		{op: "find", name: "no entry", count: block0 - int64(5), fail: true},
	}

	testWithFixationTemplate(t, playbook, 3, 1)
}

func TestGetAllEntries(t *testing.T) {
	block0 := int64(10)

	playbook := []fixationTemplate{
		{op: "append", name: "entry #1", index: "prefix1_a", count: block0, coin: 0},
		{op: "append", name: "entry #1", index: "prefix1_b", count: block0, coin: 1},
		{op: "append", name: "entry #1", index: "prefix1_c", count: block0, coin: 2},
		{op: "append", name: "entry #1", index: "prefix2_a", count: block0, coin: 3},
		{op: "append", name: "entry #1", index: "prefix2_b", count: block0, coin: 4},
		{op: "append", name: "entry #1", index: "prefix3_a", count: block0, coin: 5},
		{op: "getall", name: "to check all indices", count: 6},
		{op: "getallprefix", name: "to check all indices with prefix", index: "prefix", count: 6},
		{op: "getallprefix", name: "to check indices with prefix1", index: "prefix1", count: 3},
		{op: "getallprefix", name: "to check indices with prefix2", index: "prefix2", count: 2},
		{op: "getallprefix", name: "to check indices with prefix3", index: "prefix3", count: 1},
	}

	testWithFixationTemplate(t, playbook, 6, 1)
}
