package fixationstore

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/fixationstore/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	ctx, fs := initCtxAndFixationStore(t)

	// run playbook this will fill up the fixation store
	block0 := int64(10)
	block1 := block0 + int64(fs.getStaleBlocks(ctx)) + 1
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
		{op: "block", name: "add STALE_ENTRY_TIME+1", count: int64(fs.getStaleBlocks(ctx)) + 1},
		// entry #1 now deleted because blocks advanced by STALE_ENTRY_TIME+1
		{op: "has", name: "entry #1 (now stale/gone)", count: block0, fail: true},
		{op: "find", name: "entry #1 (now stale/gone)", count: block0, fail: true},
		{op: "find", name: "latest entry", coin: 1},
		{op: "getall", name: "to check again exactly one index", count: 1},
	}

	runPlaybook(t, ctx, []*FixationStore{fs}, playbook, 2)

	gs := fs.Export(ctx)
	emptyCtx, _ := initCtx(t)
	fs.Init(emptyCtx, gs)

	store1 := prefix.NewStore(
		ctx.KVStore(fs.storeKey),
		types.KeyPrefix(fs.prefix))
	iterator1 := sdk.KVStorePrefixIterator(store1, []byte{})
	defer iterator1.Close()

	store2 := prefix.NewStore(
		emptyCtx.KVStore(fs.storeKey),
		types.KeyPrefix(fs.prefix))
	iterator2 := sdk.KVStorePrefixIterator(store2, []byte{})
	defer iterator2.Close()

	for ; iterator1.Valid(); iterator1.Next() {
		require.Equal(t, iterator1.Key(), iterator2.Key())
		require.Equal(t, iterator1.Value(), iterator2.Value())
		iterator2.Next()
	}
}
