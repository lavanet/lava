package keeper

import (
	"context"
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	conflictkeeper "github.com/lavanet/lava/x/conflict/keeper"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	epochstoragekeeper "github.com/lavanet/lava/x/epochstorage/keeper"
	epochtypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingkeeper "github.com/lavanet/lava/x/pairing/keeper"
	"github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"github.com/tendermint/tendermint/rpc/core"
	tmdb "github.com/tendermint/tm-db"

	speckeeper "github.com/lavanet/lava/x/spec/keeper"
)

type Keepers struct {
	Epochstorage  epochstoragekeeper.Keeper
	Spec          speckeeper.Keeper
	Pairing       pairingkeeper.Keeper
	Conflict      conflictkeeper.Keeper
	BankKeeper    mockBankKeeper
	AccountKeeper mockAccountKeeper
}

type Servers struct {
	EpochServer    epochtypes.MsgServer
	SpecServer     spectypes.MsgServer
	PairingServer  types.MsgServer
	ConflictServer conflicttypes.MsgServer
}

func InitAllKeepers(t testing.TB) (*Servers, *Keepers, context.Context) {
	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	pairingStoreKey := sdk.NewKVStoreKey(types.StoreKey)
	pairingMemStoreKey := storetypes.NewMemoryStoreKey(types.MemStoreKey)
	stateStore.MountStoreWithDB(pairingStoreKey, sdk.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(pairingMemStoreKey, sdk.StoreTypeMemory, nil)

	specStoreKey := sdk.NewKVStoreKey(spectypes.StoreKey)
	specMemStoreKey := storetypes.NewMemoryStoreKey(spectypes.MemStoreKey)
	stateStore.MountStoreWithDB(specStoreKey, sdk.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(specMemStoreKey, sdk.StoreTypeMemory, nil)

	epochStoreKey := sdk.NewKVStoreKey(epochtypes.StoreKey)
	epochMemStoreKey := storetypes.NewMemoryStoreKey(epochtypes.MemStoreKey)
	stateStore.MountStoreWithDB(epochStoreKey, sdk.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(epochMemStoreKey, sdk.StoreTypeMemory, nil)

	conflictStoreKey := sdk.NewKVStoreKey(conflicttypes.StoreKey)
	conflictMemStoreKey := storetypes.NewMemoryStoreKey(conflicttypes.MemStoreKey)
	stateStore.MountStoreWithDB(conflictStoreKey, sdk.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(conflictMemStoreKey, sdk.StoreTypeMemory, nil)

	require.NoError(t, stateStore.LoadLatestVersion())

	epochparamsSubspace := paramtypes.NewSubspace(cdc,
		types.Amino,
		epochStoreKey,
		epochMemStoreKey,
		"EpochstorageParams",
	)

	pairingparamsSubspace := paramtypes.NewSubspace(cdc,
		types.Amino,
		pairingStoreKey,
		pairingMemStoreKey,
		"PairingParams",
	)

	specparamsSubspace := paramtypes.NewSubspace(cdc,
		types.Amino,
		specStoreKey,
		specMemStoreKey,
		"SpecParams",
	)

	conflictparamsSubspace := paramtypes.NewSubspace(cdc,
		types.Amino,
		conflictStoreKey,
		conflictMemStoreKey,
		"ConflictParams",
	)

	ks := Keepers{}
	ks.AccountKeeper = mockAccountKeeper{}
	ks.BankKeeper = mockBankKeeper{balance: make(map[string]sdk.Coins), moduleBank: make(map[string]map[string]sdk.Coins)}
	ks.Spec = *speckeeper.NewKeeper(cdc, specStoreKey, specMemStoreKey, specparamsSubspace)
	ks.Epochstorage = *epochstoragekeeper.NewKeeper(cdc, epochStoreKey, epochMemStoreKey, epochparamsSubspace, &ks.BankKeeper, &ks.AccountKeeper, ks.Spec)
	ks.Pairing = *pairingkeeper.NewKeeper(cdc, pairingStoreKey, pairingMemStoreKey, pairingparamsSubspace, &ks.BankKeeper, &ks.AccountKeeper, ks.Spec, ks.Epochstorage)
	ks.Conflict = *conflictkeeper.NewKeeper(cdc, conflictStoreKey, conflictMemStoreKey, conflictparamsSubspace, &ks.BankKeeper, &ks.AccountKeeper, ks.Pairing, ks.Epochstorage, ks.Spec)

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	ks.Pairing.SetParams(ctx, types.DefaultParams())
	ks.Spec.SetParams(ctx, spectypes.DefaultParams())
	ks.Epochstorage.SetParams(ctx, epochtypes.DefaultParams())
	ks.Conflict.SetParams(ctx, conflicttypes.DefaultParams())

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

func AdvanceEpoch(ctx context.Context, ks *Keepers) context.Context {
	unwrapedCtx := sdk.UnwrapSDKContext(ctx)

	nextEpochBlockNum := ks.Epochstorage.GetNextEpoch(unwrapedCtx, ks.Epochstorage.GetEpochStart(unwrapedCtx))
	unwrapedCtx = unwrapedCtx.WithBlockHeight(int64(nextEpochBlockNum))

	NewBlock(sdk.WrapSDKContext(unwrapedCtx), ks)
	return sdk.WrapSDKContext(unwrapedCtx)
}

func NewBlock(ctx context.Context, ks *Keepers) {
	unwrapedCtx := sdk.UnwrapSDKContext(ctx)
	if ks.Epochstorage.IsEpochStart(sdk.UnwrapSDKContext(ctx)) {

		block := uint64(unwrapedCtx.BlockHeight())

		//begin block
		ks.Epochstorage.SetEpochDetailsStart(unwrapedCtx, block)
		ks.Epochstorage.StoreEpochStakeStorage(unwrapedCtx, block, epochtypes.ProviderKey)
		ks.Epochstorage.StoreEpochStakeStorage(unwrapedCtx, block, epochtypes.ClientKey)

		ks.Pairing.RemoveOldEpochPayment(unwrapedCtx)
		ks.Pairing.CheckUnstakingForCommit(unwrapedCtx)

		//end block
		ks.Epochstorage.RemoveOldEpochData(unwrapedCtx, epochtypes.ProviderKey)
		ks.Epochstorage.RemoveOldEpochData(unwrapedCtx, epochtypes.ClientKey)
		ks.Epochstorage.UpdateEarliestEpochstart(unwrapedCtx)
	}

	ks.Conflict.CheckAndHandleAllVotes(unwrapedCtx)

	blockstore := MockBlockStore{}
	blockstore.SetHeight(sdk.UnwrapSDKContext(ctx).BlockHeight())
	core.SetEnvironment(&core.Environment{BlockStore: &blockstore})
}
