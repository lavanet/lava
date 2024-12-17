package keeper

import (
	"testing"
	"time"

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
	"github.com/lavanet/lava/v4/x/dualstaking/keeper"
	"github.com/lavanet/lava/v4/x/dualstaking/types"
	epochstoragekeeper "github.com/lavanet/lava/v4/x/epochstorage/keeper"
	fixationkeeper "github.com/lavanet/lava/v4/x/fixationstore/keeper"
	speckeeper "github.com/lavanet/lava/v4/x/spec/keeper"
	timerstorekeeper "github.com/lavanet/lava/v4/x/timerstore/keeper"
	"github.com/stretchr/testify/require"
)

func DualstakingKeeper(t testing.TB) (*keeper.Keeper, sdk.Context) {
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
		"DualstakingParams",
	)

	paramsSubspaceEpochstorage := typesparams.NewSubspace(cdc,
		types.Amino,
		storeKey,
		memStoreKey,
		"EpochStorageParams",
	)

	paramsSubspaceSpec := typesparams.NewSubspace(cdc,
		types.Amino,
		storeKey,
		memStoreKey,
		"SpecParams",
	)

	tsKeeper := timerstorekeeper.NewKeeper(cdc)
	epochstorageKeeper := epochstoragekeeper.NewKeeper(cdc, nil, nil, paramsSubspaceEpochstorage, nil, nil, nil, nil)

	k := keeper.NewKeeper(
		cdc,
		storeKey,
		memStoreKey,
		paramsSubspace,
		&mockBankKeeper{},
		&mockStakingKeeperEmpty{},
		&mockAccountKeeper{},
		epochstorageKeeper,
		speckeeper.NewKeeper(cdc, nil, nil, paramsSubspaceSpec, nil),
		fixationkeeper.NewKeeper(cdc, tsKeeper, epochstorageKeeper.BlocksToSaveRaw),
	)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())
	ctx = ctx.WithBlockTime(time.Now().UTC())
	// Initialize params
	k.SetParams(ctx, types.DefaultParams())

	return k, ctx
}
