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
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	distributionkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	typesparams "github.com/cosmos/cosmos-sdk/x/params/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	downtimekeeper "github.com/lavanet/lava/v4/x/downtime/keeper"
	downtimemoduletypes "github.com/lavanet/lava/v4/x/downtime/types"
	v1 "github.com/lavanet/lava/v4/x/downtime/v1"
	epochstoragekeeper "github.com/lavanet/lava/v4/x/epochstorage/keeper"
	"github.com/lavanet/lava/v4/x/rewards/keeper"
	"github.com/lavanet/lava/v4/x/rewards/types"
	timerstorekeeper "github.com/lavanet/lava/v4/x/timerstore/keeper"
	"github.com/stretchr/testify/require"
)

func RewardsKeeper(t testing.TB) (*keeper.Keeper, sdk.Context) {
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
		"RewardsParams",
	)

	paramsSubspaceDowntime := typesparams.NewSubspace(cdc,
		types.Amino,
		storeKey,
		memStoreKey,
		"DowntimeParams",
	)

	paramsSubspaceEpochstorage := typesparams.NewSubspace(cdc,
		types.Amino,
		storeKey,
		memStoreKey,
		"EpochStorageParams",
	)

	downtimeKey := storetypes.NewKVStoreKey(downtimemoduletypes.StoreKey)
	stateStore.MountStoreWithDB(downtimeKey, storetypes.StoreTypeIAVL, db)

	stakingStoreKey := storetypes.NewKVStoreKey(stakingtypes.StoreKey)
	stakingKeeper := *stakingkeeper.NewKeeper(cdc, stakingStoreKey, mockAccountKeeper{}, mockBankKeeper{}, authtypes.NewModuleAddress(govtypes.ModuleName).String())

	distributionStoreKey := storetypes.NewKVStoreKey(distributiontypes.StoreKey)
	distributionKeeper := distributionkeeper.NewKeeper(cdc, distributionStoreKey, mockAccountKeeper{}, mockBankKeeper{}, stakingKeeper, authtypes.FeeCollectorName, authtypes.NewModuleAddress(govtypes.ModuleName).String())

	epochstorageKeeper := epochstoragekeeper.NewKeeper(cdc, nil, nil, paramsSubspaceEpochstorage, nil, nil, nil, stakingKeeper)
	downtimeKeeper := downtimekeeper.NewKeeper(cdc, downtimeKey, paramsSubspaceDowntime, epochstorageKeeper)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	downtimeKeeper.SetParams(ctx, v1.DefaultParams())

	k := keeper.NewKeeper(
		cdc,
		storeKey,
		memStoreKey,
		paramsSubspace,
		mockBankKeeper{},
		mockAccountKeeper{},
		nil,
		nil,
		downtimeKeeper,
		stakingKeeper,
		nil,
		distributionKeeper,
		authtypes.FeeCollectorName,
		timerstorekeeper.NewKeeper(cdc),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// Initialize params
	k.SetParams(ctx, types.DefaultParams())

	return k, ctx
}
