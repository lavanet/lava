package types

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	ctx, tsArr := initCtxAndTimerStores(t, 1)
	ts := tsArr[0]

	ts.AddTimerByBlockHeight(ctx, uint64(ctx.BlockHeight())+1, []byte("bla"), []byte("abc"))
	ts.AddTimerByBlockTime(ctx, uint64(ctx.BlockTime().Unix())+1, []byte("def"), []byte("test"))

	gs := ts.Export(ctx)
	emptyCtx, _ := initCtx(t)
	ts.Init(emptyCtx, gs)

	store1 := prefix.NewStore(
		ctx.KVStore(ts.storeKey),
		KeyPrefix(ts.prefix))
	iterator1 := sdk.KVStorePrefixIterator(store1, []byte{})
	defer iterator1.Close()

	store2 := prefix.NewStore(
		emptyCtx.KVStore(ts.storeKey),
		KeyPrefix(ts.prefix))
	iterator2 := sdk.KVStorePrefixIterator(store2, []byte{})
	defer iterator2.Close()

	for ; iterator1.Valid(); iterator1.Next() {
		require.Equal(t, iterator1.Key(), iterator2.Key())
		require.Equal(t, iterator1.Value(), iterator2.Value())
		iterator2.Next()
	}
}
