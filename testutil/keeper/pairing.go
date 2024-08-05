package keeper

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
	typesparams "github.com/cosmos/cosmos-sdk/x/params/types"
	epochstoragemodule "github.com/lavanet/lava/v2/x/epochstorage"
	epochstoragekeeper "github.com/lavanet/lava/v2/x/epochstorage/keeper"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	fixationkeeper "github.com/lavanet/lava/v2/x/fixationstore/keeper"
	"github.com/lavanet/lava/v2/x/pairing/keeper"
	"github.com/lavanet/lava/v2/x/pairing/types"
	timerstorekeeper "github.com/lavanet/lava/v2/x/timerstore/keeper"
	"github.com/stretchr/testify/require"
)

func PairingKeeper(t testing.TB) (*keeper.Keeper, sdk.Context) {
	storeKey := sdk.NewKVStoreKey(types.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	epochStoreKey := sdk.NewKVStoreKey(epochstoragetypes.StoreKey)
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
