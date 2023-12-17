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
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	typesparams "github.com/cosmos/cosmos-sdk/x/params/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	downtimekeeper "github.com/lavanet/lava/x/downtime/keeper"
	downtimemoduletypes "github.com/lavanet/lava/x/downtime/types"
	v1 "github.com/lavanet/lava/x/downtime/v1"
	epochstoragekeeper "github.com/lavanet/lava/x/epochstorage/keeper"
	"github.com/lavanet/lava/x/rewards/keeper"
	"github.com/lavanet/lava/x/rewards/types"
	timerstorekeeper "github.com/lavanet/lava/x/timerstore/keeper"
	"github.com/stretchr/testify/require"
)

func RewardsKeeper(t testing.TB) (*keeper.Keeper, sdk.Context) {
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

	downtimeKey := sdk.NewKVStoreKey(downtimemoduletypes.StoreKey)
	stateStore.MountStoreWithDB(downtimeKey, storetypes.StoreTypeIAVL, db)

	epochstorageKeeper := epochstoragekeeper.NewKeeper(cdc, nil, nil, paramsSubspaceEpochstorage, nil, nil, nil)
	downtimeKeeper := downtimekeeper.NewKeeper(cdc, downtimeKey, paramsSubspaceDowntime, epochstorageKeeper)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	downtimeKeeper.SetParams(ctx, v1.DefaultParams())

	stakingStoreKey := sdk.NewKVStoreKey(stakingtypes.StoreKey)
	stakingKeeper := *stakingkeeper.NewKeeper(cdc, stakingStoreKey, mockAccountKeeper{}, mockBankKeeper{}, authtypes.NewModuleAddress(govtypes.ModuleName).String())

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
		authtypes.FeeCollectorName,
		timerstorekeeper.NewKeeper(cdc),
	)

	// Initialize params
	k.SetParams(ctx, types.DefaultParams())

	return k, ctx
}
