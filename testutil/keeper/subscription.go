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
	dualstakingkeeper "github.com/lavanet/lava/v4/x/dualstaking/keeper"
	epochstoragekeeper "github.com/lavanet/lava/v4/x/epochstorage/keeper"
	fixationkeeper "github.com/lavanet/lava/v4/x/fixationstore/keeper"
	planskeeper "github.com/lavanet/lava/v4/x/plans/keeper"
	projectskeeper "github.com/lavanet/lava/v4/x/projects/keeper"
	"github.com/lavanet/lava/v4/x/subscription/keeper"
	"github.com/lavanet/lava/v4/x/subscription/types"
	timerstorekeeper "github.com/lavanet/lava/v4/x/timerstore/keeper"
	"github.com/stretchr/testify/require"
)

func SubscriptionKeeper(t testing.TB) (*keeper.Keeper, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(memStoreKey, storetypes.StoreTypeMemory, nil)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	paramsSubspace := typesparams.NewSubspace(cdc,
		types.Amino,
		storeKey,
		memStoreKey,
		"SubscriptionParams",
	)

	paramsSubspaceEpochstorage := typesparams.NewSubspace(cdc,
		types.Amino,
		storeKey,
		memStoreKey,
		"EpochStorageParams",
	)

	paramsSubspaceProjects := typesparams.NewSubspace(cdc,
		types.Amino,
		storeKey,
		memStoreKey,
		"ProjectsParams",
	)

	paramsSubspacePlans := typesparams.NewSubspace(cdc,
		types.Amino,
		storeKey,
		memStoreKey,
		"PlansParams",
	)
	epochstorageKeeper := epochstoragekeeper.NewKeeper(cdc, nil, nil, paramsSubspaceEpochstorage, nil, nil, nil, nil)
	tsKeeper := timerstorekeeper.NewKeeper(cdc)
	fsKeeper := fixationkeeper.NewKeeper(cdc, tsKeeper, epochstorageKeeper.BlocksToSaveRaw)

	k := keeper.NewKeeper(
		cdc,
		storeKey,
		memStoreKey,
		paramsSubspace,
		nil,
		nil,
		epochstorageKeeper,
		projectskeeper.NewKeeper(cdc, nil, nil, paramsSubspaceProjects, nil, fsKeeper),
		planskeeper.NewKeeper(cdc, nil, nil, paramsSubspacePlans, nil, nil, fsKeeper, nil),
		dualstakingkeeper.NewKeeper(cdc, nil, nil, paramsSubspace, nil, nil, mockAccountKeeper{}, nil, nil, fsKeeper),
		nil,
		nil,
		fsKeeper,
		tsKeeper,
		nil,
	)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	k.SetParams(ctx, types.DefaultParams())

	return k, ctx
}
