package keeper

import (
	"context"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/lavanet/lava/x/conflict"
	conflictkeeper "github.com/lavanet/lava/x/conflict/keeper"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	"github.com/lavanet/lava/x/epochstorage"
	epochstoragekeeper "github.com/lavanet/lava/x/epochstorage/keeper"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing"
	pairingkeeper "github.com/lavanet/lava/x/pairing/keeper"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/lavanet/lava/x/spec"
	speckeeper "github.com/lavanet/lava/x/spec/keeper"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"github.com/tendermint/tendermint/rpc/core"
	tmdb "github.com/tendermint/tm-db"
)

type Keepers struct {
	Epochstorage  epochstoragekeeper.Keeper
	Spec          speckeeper.Keeper
	Pairing       pairingkeeper.Keeper
	Conflict      conflictkeeper.Keeper
	BankKeeper    mockBankKeeper
	AccountKeeper mockAccountKeeper
	ParamsKeeper  paramskeeper.Keeper
}

type Servers struct {
	EpochServer    epochstoragetypes.MsgServer
	SpecServer     spectypes.MsgServer
	PairingServer  pairingtypes.MsgServer
	ConflictServer conflicttypes.MsgServer
}

func InitAllKeepers(t testing.TB) (*Servers, *Keepers, context.Context) {
	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	pairingStoreKey := sdk.NewKVStoreKey(pairingtypes.StoreKey)
	pairingMemStoreKey := storetypes.NewMemoryStoreKey(pairingtypes.MemStoreKey)
	stateStore.MountStoreWithDB(pairingStoreKey, sdk.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(pairingMemStoreKey, sdk.StoreTypeMemory, nil)

	specStoreKey := sdk.NewKVStoreKey(spectypes.StoreKey)
	specMemStoreKey := storetypes.NewMemoryStoreKey(spectypes.MemStoreKey)
	stateStore.MountStoreWithDB(specStoreKey, sdk.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(specMemStoreKey, sdk.StoreTypeMemory, nil)

	epochStoreKey := sdk.NewKVStoreKey(epochstoragetypes.StoreKey)
	epochMemStoreKey := storetypes.NewMemoryStoreKey(epochstoragetypes.MemStoreKey)
	stateStore.MountStoreWithDB(epochStoreKey, sdk.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(epochMemStoreKey, sdk.StoreTypeMemory, nil)

	paramsStoreKey := sdk.NewKVStoreKey(paramstypes.StoreKey)
	stateStore.MountStoreWithDB(paramsStoreKey, sdk.StoreTypeIAVL, db)
	tkey := sdk.NewTransientStoreKey(paramstypes.TStoreKey)
	stateStore.MountStoreWithDB(tkey, sdk.StoreTypeIAVL, db)
	conflictStoreKey := sdk.NewKVStoreKey(conflicttypes.StoreKey)
	conflictMemStoreKey := storetypes.NewMemoryStoreKey(conflicttypes.MemStoreKey)
	stateStore.MountStoreWithDB(conflictStoreKey, sdk.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(conflictMemStoreKey, sdk.StoreTypeMemory, nil)

	require.NoError(t, stateStore.LoadLatestVersion())

	paramsKeeper := paramskeeper.NewKeeper(cdc, pairingtypes.Amino, paramsStoreKey, tkey)
	paramsKeeper.Subspace(spectypes.ModuleName)
	paramsKeeper.Subspace(epochstoragetypes.ModuleName)
	paramsKeeper.Subspace(pairingtypes.ModuleName)
	// paramsKeeper.Subspace(conflicttypes.ModuleName) //TODO...

	epochparamsSubspace, _ := paramsKeeper.GetSubspace(epochstoragetypes.ModuleName)

	pairingparamsSubspace, _ := paramsKeeper.GetSubspace(pairingtypes.ModuleName)

	specparamsSubspace, _ := paramsKeeper.GetSubspace(spectypes.ModuleName)

	conflictparamsSubspace := paramstypes.NewSubspace(cdc,
		conflicttypes.Amino,
		conflictStoreKey,
		conflictMemStoreKey,
		"ConflictParams",
	)

	ks := Keepers{}
	ks.AccountKeeper = mockAccountKeeper{}
	ks.BankKeeper = mockBankKeeper{balance: make(map[string]sdk.Coins), moduleBank: make(map[string]map[string]sdk.Coins)}
	ks.Spec = *speckeeper.NewKeeper(cdc, specStoreKey, specMemStoreKey, specparamsSubspace)
	ks.Epochstorage = *epochstoragekeeper.NewKeeper(cdc, epochStoreKey, epochMemStoreKey, epochparamsSubspace, &ks.BankKeeper, &ks.AccountKeeper, ks.Spec)
	ks.Pairing = *pairingkeeper.NewKeeper(cdc, pairingStoreKey, pairingMemStoreKey, pairingparamsSubspace, &ks.BankKeeper, &ks.AccountKeeper, ks.Spec, &ks.Epochstorage)
	ks.ParamsKeeper = paramsKeeper
	ks.Conflict = *conflictkeeper.NewKeeper(cdc, conflictStoreKey, conflictMemStoreKey, conflictparamsSubspace, &ks.BankKeeper, &ks.AccountKeeper, ks.Pairing, ks.Epochstorage, ks.Spec)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	// Initialize Genesis
	spec.InitGenesis(ctx, ks.Spec, *spectypes.DefaultGenesis())
	pairing.InitGenesis(ctx, ks.Pairing, *pairingtypes.DefaultGenesis())
	epochstorage.InitGenesis(ctx, ks.Epochstorage, *epochstoragetypes.DefaultGenesis()) //epoch storage is always last
	conflict.InitGenesis(ctx, ks.Conflict, *conflicttypes.DefaultGenesis())

	ss := Servers{}
	ss.EpochServer = epochstoragekeeper.NewMsgServerImpl(ks.Epochstorage)
	ss.SpecServer = speckeeper.NewMsgServerImpl(ks.Spec)
	ss.PairingServer = pairingkeeper.NewMsgServerImpl(ks.Pairing)
	ss.ConflictServer = conflictkeeper.NewMsgServerImpl(ks.Conflict)

	return &ss, &ks, sdk.WrapSDKContext(ctx)
}

func AdvanceBlock(ctx context.Context, ks *Keepers) context.Context {
	unwrapedCtx := sdk.UnwrapSDKContext(ctx)

	block := uint64(unwrapedCtx.BlockHeight() + 1)
	unwrapedCtx = unwrapedCtx.WithBlockHeight(int64(block))

	NewBlock(sdk.WrapSDKContext(unwrapedCtx), ks)

	return sdk.WrapSDKContext(unwrapedCtx)
}

func AdvanceBlocks(ctx context.Context, ks *Keepers, blocks int) context.Context {
	for i := 0; i < blocks; i++ {
		ctx = AdvanceBlock(ctx, ks)
	}

	return ctx
}

func AdvanceToBlock(ctx context.Context, ks *Keepers, block uint64) context.Context {

	unwrapedCtx := sdk.UnwrapSDKContext(ctx)
	if uint64(unwrapedCtx.BlockHeight()) == block {
		return ctx
	}

	for uint64(unwrapedCtx.BlockHeight()) < block {
		ctx = AdvanceBlock(ctx, ks)
		unwrapedCtx = sdk.UnwrapSDKContext(ctx)
	}

	return ctx
}

//Make sure you save the new context
func AdvanceEpoch(ctx context.Context, ks *Keepers) context.Context {
	unwrapedCtx := sdk.UnwrapSDKContext(ctx)

	nextEpochBlockNum, err := ks.Epochstorage.GetNextEpoch(unwrapedCtx, ks.Epochstorage.GetEpochStart(unwrapedCtx))
	if err != nil {
		panic(err)
	}

	return AdvanceToBlock(ctx, ks, nextEpochBlockNum)
}

//Make sure you save the new context
func NewBlock(ctx context.Context, ks *Keepers) {
	unwrapedCtx := sdk.UnwrapSDKContext(ctx)
	if ks.Epochstorage.IsEpochStart(sdk.UnwrapSDKContext(ctx)) {

		block := uint64(unwrapedCtx.BlockHeight())

		ks.Epochstorage.FixateParams(unwrapedCtx, block)
		//begin block
		ks.Epochstorage.SetEpochDetailsStart(unwrapedCtx, block)
		ks.Epochstorage.StoreEpochStakeStorage(unwrapedCtx, block, epochstoragetypes.ProviderKey)
		ks.Epochstorage.StoreEpochStakeStorage(unwrapedCtx, block, epochstoragetypes.ClientKey)

		ks.Pairing.RemoveOldEpochPayment(unwrapedCtx)
		ks.Pairing.CheckUnstakingForCommit(unwrapedCtx)

		//end block
		ks.Epochstorage.RemoveOldEpochData(unwrapedCtx, epochstoragetypes.ProviderKey)
		ks.Epochstorage.RemoveOldEpochData(unwrapedCtx, epochstoragetypes.ClientKey)
		ks.Epochstorage.UpdateEarliestEpochstart(unwrapedCtx)
	}

	ks.Conflict.CheckAndHandleAllVotes(unwrapedCtx)

	blockstore := MockBlockStore{}
	blockstore.SetHeight(sdk.UnwrapSDKContext(ctx).BlockHeight())
	core.SetEnvironment(&core.Environment{BlockStore: &blockstore})
}
