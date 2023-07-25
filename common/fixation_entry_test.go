package common

import (
	"fmt"
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
	op    string // op-code
	name  string // description
	index string // entry index (name)
	store int    // which fixation store
	coin  int    // which data item
	fail  bool   // should fail?
	// count = 0 sets target block for ctx
	// count > 0 is target block for ctx, or actual count
	// count < 0 is target block for "append" only (not ctx)
	count int64 // target block or count
}

// helper to automate testing operations
func testWithFixationTemplate(t *testing.T, playbook []fixationTemplate, countObj int, countVS int) {
	ctx, fs := initCtxAndFixationStores(t, countVS)

	var coins []sdk.Coin
	var dummy sdk.Coin

	for i := 0; i < countObj; i++ {
		coins = append(coins, sdk.NewCoin("utest", sdk.NewInt(int64(i+1))))
	}

	for _, play := range playbook {
		block := uint64(ctx.BlockHeight())
		if play.count > 0 {
			block = uint64(play.count)
		}
		index := "myindex"
		if play.index != "" {
			index = play.index
		}
		what := play.op + " " + play.name +
			" index: " + index +
			" block: " + strconv.Itoa(int(block))
		// uncomment for debugging:
		// fmt.Printf("step: cmd %s, ctx %d, count %d, block %d\n",
		//     play.op, ctx.BlockHeight(), play.count, block)
		switch play.op {
		case "append":
			if play.count > 0 && block > uint64(ctx.BlockHeight()) {
				ctx = ctx.WithBlockHeight(int64(block))
			} else if play.count < 0 {
				block = uint64(-play.count) // allow future block with advancing ctx
			}
			err := fs[play.store].AppendEntry(ctx, index, block, &coins[play.coin])
			if !play.fail {
				require.Nil(t, err, what+fmt.Sprintf(" %v", err))
			} else {
				require.NotNil(t, err, what)
			}
		case "modify":
			fs[play.store].ModifyEntry(ctx, index, block, &coins[play.coin])
		case "read":
			fs[play.store].ReadEntry(ctx, index, block, &dummy)
		case "find":
			found := fs[play.store].FindEntry(ctx, index, block, &dummy)
			if !play.fail {
				require.True(t, found, what)
				require.Equal(t, dummy, coins[play.coin], what)
			} else {
				require.False(t, found, what)
			}
		case "has": //nolint:goconst
			has := fs[play.store].HasEntry(ctx, index, block)
			require.Equal(t, !play.fail, has, what)
		case "stale":
			stale := fs[play.store].IsEntryStale(ctx, index, block)
			require.Equal(t, play.fail, stale, what)
		case "get":
			found := fs[play.store].GetEntry(ctx, index, &dummy)
			if !play.fail {
				require.True(t, found, what)
				require.Equal(t, dummy, coins[play.coin], what)
			} else {
				require.False(t, found, what)
			}
		case "del":
			err := fs[play.store].DelEntry(ctx, index, block)
			if !play.fail {
				require.Nil(t, err, what+fmt.Sprintf(" %v", err))
			} else {
				require.NotNil(t, err, what)
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
			indexList := fs[play.store].GetAllEntryVersions(ctx, index)
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
	unknown := "unknown"

	playbook := []fixationTemplate{
		{op: "append", name: "with invalid index (fail)", index: invalid, fail: true},
		{op: "find", name: "with invalid index (fail)", index: invalid, fail: true},
		{op: "get", name: "with invalid index (fail)", index: invalid, fail: true},
		{op: "find", name: "with unknown index (fail)", index: unknown, fail: true},
		{op: "get", name: "with unknown index (fail)", index: unknown, fail: true},
	}

	testWithFixationTemplate(t, playbook, 1, 1)
}

// Test addition and auto-removal of a fixation entry
func TestFixationEntryAdditionAndRemoval(t *testing.T) {
	block0 := int64(10)
	block1 := block0 + types.STALE_ENTRY_TIME + 1

	playbook := []fixationTemplate{
		{op: "append", name: "entry #1", count: block0, coin: 0},
		{op: "find", name: "entry #1", count: block0, coin: 0},
		{op: "has", name: "entry #1", count: block0, coin: 0},
		{op: "stale", name: "entry #1", count: block0, coin: 0},
		{op: "getall", name: "to check exactly one index", count: 1},
		{op: "append", name: "entry #2", count: block1, coin: 1},
		{op: "has", name: "entry #1 (again)", count: block0, coin: 0},
		{op: "has", name: "entry #2", count: block1, coin: 0},
		// entry #1 not deleted because not enough time with refcount = zero
		{op: "has", name: "entry #1 (not stale yet)", count: block0},
		{op: "find", name: "entry #1 (not stale yet)", count: block0},
		{op: "block", name: "add STALE_ENTRY_TIME+1", count: types.STALE_ENTRY_TIME + 1},
		// entry #1 now deleted because blocks advanced by STALE_ENTRY_TIME+1
		{op: "has", name: "entry #1 (now stale/gone)", count: block0, fail: true},
		{op: "find", name: "entry #1 (now stale/gone)", count: block0, fail: true},
		{op: "find", name: "latest entry", coin: 1},
		{op: "getall", name: "to check again exactly one index", count: 1},
	}

	testWithFixationTemplate(t, playbook, 2, 1)
}

// Test append of future entries
func TestFixationEntryAppendFuture(t *testing.T) {
	block0 := int64(100)
	block1 := int64(200)
	block2 := int64(300)

	playbook := []fixationTemplate{
		{op: "append", name: "entry #1", count: block0, coin: 0},
		{op: "append", name: "entry #3", count: -block2, coin: 2},
		{op: "append", name: "entry #2", count: -block1, coin: 1},
		{op: "getvers", name: "to check 3 versions", count: 3},
		{op: "get", name: "get entry #1", coin: 0},
		{op: "block", name: "advance to block1", count: block1 - block0},
		{op: "get", name: "get entry #2", coin: 1},
		{op: "block", name: "advance to block2", count: block2 - block1},
		{op: "get", name: "get entry #3", coin: 2},
		{op: "getvers", name: "to check 3 versions", count: 3},
	}

	testWithFixationTemplate(t, playbook, 3, 1)
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
		{op: "has", name: "entry #1", count: block0, coin: 0},
		{op: "has", name: "entry #2", count: block1, coin: 1},
		{op: "find", name: "entry #1", count: block0 + 1, coin: 0},
		{op: "find", name: "entry #2", count: block1 + 1, coin: 1},
		{op: "block", name: "add STALE_ENTRY_TIME+1", count: types.STALE_ENTRY_TIME + 1},
		// entry #2 now stale and therefore should not be visible
		{op: "find", name: "entry #2", count: block1 + 1, fail: true},
		// but should still be positive for HasEntry()
		{op: "has", name: "entry #2 (still positive)", count: block1},
		{op: "stale", name: "entry #2 (still positive)", count: block1, fail: true},
		// entry #3 (refcount = zero) is old, but being the latest it always
		// remains visible (despite of refcount and age).
		{op: "has", name: "entry #3", count: block2, coin: 2},
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
		{op: "block", name: "add STALE_ENTRY_TIME+1", count: types.STALE_ENTRY_TIME + 1},
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
		{op: "put", name: "negative refcount entry #1 version 0", count: block0},
	}

	require.Panics(t, func() { testWithFixationTemplate(t, playbook, 1, 1) })
}

func TestPutFutureEntry(t *testing.T) {
	block0 := int64(10)
	block1 := block0 + int64(10)

	playbook := []fixationTemplate{
		{op: "append", name: "entry #1 version 0", count: block0, coin: 0},
		{op: "append", name: "entry #1 version 1 (future)", count: -block1, coin: 0},
		{op: "getvers", name: "to check 2 versions", count: 2},
		{op: "block", name: "advance 1 block", count: 1},
		{op: "put", name: "cancel entry #1 version 1", count: block1},
		{op: "getvers", name: "to check 1 versions", count: 1},
	}

	testWithFixationTemplate(t, playbook, 1, 1)
}

func TestDelEntry(t *testing.T) {
	block0 := int64(10)
	block1 := block0 + 10
	block2 := block1 + 10
	block3 := block2 + 10

	playbook := []fixationTemplate{
		{op: "append", name: "entry #1 version 0", count: block0, coin: 0},
		{op: "get", name: "entry #1", coin: 0},
		{op: "append", name: "entry #1 version 1", count: block1, coin: 1},
		// entry #1 get-able
		{op: "getall", name: "count indices before del", count: 1},
		{op: "get", name: "entry #1", coin: 1},
		{op: "block", name: "advance to block3", count: block2 - block1},
		{op: "del", name: "entry #1"},
		// second del should fail
		{op: "del", name: "entry #1", fail: true},
		// entry #1 now gone from: get, append should fail
		{op: "get", name: "entry #1 gone", coin: 1, fail: true},
		{op: "append", name: "entry #1 version 2", count: block3, fail: true},
		{op: "getall", name: "count indices after del", count: 0},
		// entry #1 find(s) should still work
		{op: "find", name: "entry #1 version 0", coin: 0, count: block0},
		{op: "find", name: "entry #1 version 1", coin: 1, count: block1},
		// entry #1 find beyond the delete should fail
		{op: "has", name: "entry #1 version 1 (deleted)", count: block2, fail: true},
		{op: "find", name: "entry #1 version 1 (deleted)", count: block2, fail: true},
	}

	testWithFixationTemplate(t, playbook, 3, 1)
}

func TestDelEntryWithFuture(t *testing.T) {
	block0 := int64(100)
	block1 := int64(200)
	block2 := int64(300)
	block3 := int64(2000)

	playbook := []fixationTemplate{
		{op: "append", name: "entry #1", count: block0, coin: 0},
		{op: "append", name: "entry #3", count: -block2, coin: 2},
		{op: "append", name: "entry #2", count: -block1, coin: 1},
		{op: "getvers", name: "to check 3 versions", count: 3},
		{op: "del", name: "between entry #2 and entry #3 ", count: block1 + 50},
		{op: "block", name: "advance to block1", count: block1 - block0},
		// now entry #2 is latest, and should have inherited DeleteAt
		{op: "get", name: "entry #2", coin: 1},
		{op: "put", name: "entry #2"},
		{op: "block", name: "advance to block1+50", count: 50},
		// now entry #2 is deleted, and entry #3 discarded
		{op: "get", name: "entry #2", fail: true},
		{op: "getvers", name: "to check 3 versions", count: 2},
		{op: "block", name: "advance until entry #1 stale", count: types.STALE_ENTRY_TIME - 50},
		{op: "block", name: "+1", count: 1},
		{op: "getvers", name: "to check 2 version", count: 1},
		{op: "block", name: "advance until entry #2 stale", count: block2 - block1},
		{op: "getvers", name: "to check 0 version", count: 0},
		// should be possible to re-append the entry now
		{op: "append", name: "entry #1", count: block3, coin: 0},
	}

	testWithFixationTemplate(t, playbook, 3, 1)
}

// TestDelEntrySameFuture tests the case when a future entry is deleted
// on the same block it matures.
func TestDelEntrySameFuture(t *testing.T) {
	block0 := int64(100)
	block1 := int64(200)

	playbook := []fixationTemplate{
		{op: "append", name: "entry #1", count: block0, coin: 0},
		{op: "append", name: "entry #2", count: -block1, coin: 1},
		{op: "getvers", name: "to check 2 versions", count: 2},
		{op: "del", name: "exactly on (future) entry #2 ", count: block1},
		{op: "block", name: "advance to block1", count: block1 - block0},
		// now entry #2 is latest, and deleted
		{op: "get", name: "entry #1 should fail", fail: true},
		{op: "block", name: "advance until entry #1 stale", count: types.STALE_ENTRY_TIME},
		{op: "getvers", name: "to check 0 version", count: 0},
	}

	testWithFixationTemplate(t, playbook, 2, 1)
}

func TestDeleletdStaleStays(t *testing.T) {
	block0 := int64(10)
	block1 := block0 + 10
	block2 := block1 + 10
	block3 := block2 + 10
	block4 := block3 + 10

	playbook := []fixationTemplate{
		{op: "append", name: "entry #1 version 0", count: block0, coin: 0},
		{op: "get", name: "entry #1", coin: 0},
		{op: "append", name: "entry #1 version 1", count: block1, coin: 1},
		{op: "append", name: "entry #1 version 2", count: block2, coin: 2},
		{op: "append", name: "entry #1 version 3", count: block3, coin: 2},
		{op: "del", name: "entry #1", count: block4},
		// entry #1 version 1,2,3 are in stale-period now
		{op: "block", name: "add block4-block3", count: block4 - block3},
		{op: "getvers", name: "to check 4 versions left", count: 4},
		// entry #1 version 1,2,3 become stale, version 2 will be gone
		{op: "block", name: "add STALE_ENTRY_TIME", count: types.STALE_ENTRY_TIME},
		{op: "getvers", name: "to check 3 versions left", count: 3},
	}

	testWithFixationTemplate(t, playbook, 4, 1)
}

func TestExactEntryMethods(t *testing.T) {
	invalid := "index" + string('\001')
	unknown := "unknown"

	block0 := int64(10)
	block1 := block0 + types.STALE_ENTRY_TIME + 1

	playbook := []fixationTemplate{
		{op: "append", name: "entry #1 version 0", count: block0, coin: 0},
		{op: "append", name: "entry #1 version 1", count: block1, coin: 0},
	}

	testWithFixationTemplate(t, playbook, 1, 1)

	playbooks := [][]fixationTemplate{
		{{op: "read", name: "with invalid index (fail)", index: invalid}},
		{{op: "modify", name: "with invalid index (fail)", index: invalid}},
		{{op: "put", name: "with invalid index (fail)", index: invalid}},
		{{op: "read", name: "with unknown index (fail)", index: unknown}},
		{{op: "modify", name: "with unknown index (fail)", index: unknown}},
		{{op: "put", name: "with unknown index (fail)", index: unknown}},
		{{op: "read", name: "entry #1 version 0", count: block0 + 1, coin: 0}},
		{{op: "modify", name: "entry #1 version 0", count: block0 + 1, coin: 0}},
		{{op: "put", name: "entry #1 version 0", count: block0 + 1, coin: 0}},
	}

	for _, p := range playbooks {
		what := p[0].op + " " + p[0].name
		require.Panics(t, func() { testWithFixationTemplate(t, p, 1, 1) }, what)
	}
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
		{op: "getvers", name: "to check 4 versions left", count: 4},
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
