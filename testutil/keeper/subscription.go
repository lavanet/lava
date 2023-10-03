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
	epochstoragekeeper "github.com/lavanet/lava/x/epochstorage/keeper"
	"github.com/lavanet/lava/x/fixationstore"
	planskeeper "github.com/lavanet/lava/x/plans/keeper"
	projectskeeper "github.com/lavanet/lava/x/projects/keeper"
	"github.com/lavanet/lava/x/subscription/keeper"
	"github.com/lavanet/lava/x/subscription/types"
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

	fsKeeper := fixationstore.NewKeeper(cdc)

	k := keeper.NewKeeper(
		cdc,
		storeKey,
		memStoreKey,
		paramsSubspace,
		nil,
		nil,
		epochstoragekeeper.NewKeeper(cdc, nil, nil, paramsSubspaceEpochstorage, nil, nil, nil),
		projectskeeper.NewKeeper(cdc, nil, nil, paramsSubspaceProjects, nil, fsKeeper),
		planskeeper.NewKeeper(cdc, nil, nil, paramsSubspacePlans, nil, nil, fsKeeper),
		fsKeeper,
	)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	k.SetParams(ctx, types.DefaultParams())

	return k, ctx
}
