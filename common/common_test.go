package common_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmdb "github.com/tendermint/tm-db"
)

var (
	mockStoreKey    *sdk.KVStoreKey = sdk.NewKVStoreKey("storeKey")
	mockMemStoreKey *sdk.MemoryStoreKey = storetypes.NewMemoryStoreKey("storeMemKey")
)

// Helper function to init a mock keeper and context
func initCtx(t *testing.T) (sdk.Context, *codec.ProtoCodec) {
	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	stateStore.MountStoreWithDB(mockStoreKey, sdk.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(mockMemStoreKey, sdk.StoreTypeMemory, nil)

	require.NoError(t, stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.TestingLogger())

	return ctx, cdc
}
