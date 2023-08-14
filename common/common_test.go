package common

import (
	"testing"

	tmdb "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

var (
	mockStoreKey    = sdk.NewKVStoreKey("storeKey")
	mockMemStoreKey = storetypes.NewMemoryStoreKey("storeMemKey")
)

// Helper function to init a mock keeper and context
func initCtx(t *testing.T) (sdk.Context, *codec.ProtoCodec) {
	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	stateStore.MountStoreWithDB(mockStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(mockMemStoreKey, storetypes.StoreTypeMemory, nil)

	require.NoError(t, stateStore.LoadLatestVersion())

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.TestingLogger())

	return ctx, cdc
}
