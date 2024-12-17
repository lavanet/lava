package keeper

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	typesparams "github.com/cosmos/cosmos-sdk/x/params/types"
	epochstoragemodule "github.com/lavanet/lava/v4/x/epochstorage"
	epochstoragekeeper "github.com/lavanet/lava/v4/x/epochstorage/keeper"
	epochstoragetypes "github.com/lavanet/lava/v4/x/epochstorage/types"
	fixationkeeper "github.com/lavanet/lava/v4/x/fixationstore/keeper"
	"github.com/lavanet/lava/v4/x/pairing/keeper"
	"github.com/lavanet/lava/v4/x/pairing/types"
	timerstorekeeper "github.com/lavanet/lava/v4/x/timerstore/keeper"
	"github.com/stretchr/testify/require"
)

func PairingKeeper(t testing.TB) (*keeper.Keeper, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	epochStoreKey := storetypes.NewKVStoreKey(epochstoragetypes.StoreKey)
	epochMemStoreKey := storetypes.NewMemoryStoreKey(epochstoragetypes.MemStoreKey)
	stateStore.MountStoreWithDB(epochStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(epochMemStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	paramsSubspace := typesparams.NewSubspace(cdc,
		types.Amino,
		storeKey,
		memStoreKey,
		"PairingParams",
	)

	paramsSubspaceEpochstorage := typesparams.NewSubspace(cdc,
		types.Amino,
		storeKey,
		memStoreKey,
		"EpochStorageParams",
	)

	tsKeeper := timerstorekeeper.NewKeeper(cdc)
	epochstorageKeeper := epochstoragekeeper.NewKeeper(cdc, epochStoreKey, epochMemStoreKey, paramsSubspaceEpochstorage, nil, nil, nil, nil)
	k := keeper.NewKeeper(
		cdc,
		storeKey,
		memStoreKey,
		paramsSubspace,
		nil,
		nil,
		nil,
		epochstorageKeeper,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		fixationkeeper.NewKeeper(cdc, tsKeeper, epochstorageKeeper.BlocksToSaveRaw),
		tsKeeper,
	)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	k.SetParams(ctx, types.DefaultParams())
	epochstoragemodule.InitGenesis(ctx, *epochstorageKeeper, *epochstoragetypes.DefaultGenesis())
	return k, ctx
}
