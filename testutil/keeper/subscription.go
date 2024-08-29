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
	dualstakingkeeper "github.com/lavanet/lava/v2/x/dualstaking/keeper"
	epochstoragekeeper "github.com/lavanet/lava/v2/x/epochstorage/keeper"
	fixationkeeper "github.com/lavanet/lava/v2/x/fixationstore/keeper"
	planskeeper "github.com/lavanet/lava/v2/x/plans/keeper"
	projectskeeper "github.com/lavanet/lava/v2/x/projects/keeper"
	"github.com/lavanet/lava/v2/x/subscription/keeper"
	"github.com/lavanet/lava/v2/x/subscription/types"
	timerstorekeeper "github.com/lavanet/lava/v2/x/timerstore/keeper"
	"github.com/stretchr/testify/require"
)

func SubscriptionKeeper(t testing.TB) (*keeper.Keeper, sdk.Context) {
	storeKey := sdk.NewKVStoreKey(types.StoreKey)
	memStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)

	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)
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
