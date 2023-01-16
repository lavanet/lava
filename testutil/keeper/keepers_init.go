package keeper

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	conflictkeeper "github.com/lavanet/lava/x/conflict/keeper"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	epochstoragekeeper "github.com/lavanet/lava/x/epochstorage/keeper"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingkeeper "github.com/lavanet/lava/x/pairing/keeper"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/lavanet/lava/x/spec"
	speckeeper "github.com/lavanet/lava/x/spec/keeper"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"github.com/tendermint/tendermint/rpc/core"
	tenderminttypes "github.com/tendermint/tendermint/types"
	tmdb "github.com/tendermint/tm-db"
)

const (
	BLOCK_TIME                                       = 30 * time.Second
	EPOCHS_NUM_TO_CHECK_CU_FOR_UNRESPONSIVE_PROVIDER = 4 // number of epochs to sum CU that the provider serviced
	EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS              = 2 // number of epochs to sum CU of complainers against the provider
)

type Keepers struct {
	Epochstorage  epochstoragekeeper.Keeper
	Spec          speckeeper.Keeper
	Pairing       pairingkeeper.Keeper
	Conflict      conflictkeeper.Keeper
	BankKeeper    mockBankKeeper
	AccountKeeper mockAccountKeeper
	ParamsKeeper  paramskeeper.Keeper
	BlockStore    MockBlockStore
}

type Servers struct {
	EpochServer    epochstoragetypes.MsgServer
	SpecServer     spectypes.MsgServer
	PairingServer  pairingtypes.MsgServer
	ConflictServer conflicttypes.MsgServer
}

func SimulateParamChange(ctx sdk.Context, paramKeeper paramskeeper.Keeper, subspace string, key string, value string) (err error) {
	proposal := &paramproposal.ParameterChangeProposal{Changes: []paramproposal.ParamChange{{Subspace: subspace, Key: key, Value: value}}}
	err = spec.HandleParameterChangeProposal(ctx, paramKeeper, proposal)
	return
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
	ks.BankKeeper = mockBankKeeper{balance: make(map[string]sdk.Coins)}
	ks.Spec = *speckeeper.NewKeeper(cdc, specStoreKey, specMemStoreKey, specparamsSubspace)
	ks.Epochstorage = *epochstoragekeeper.NewKeeper(cdc, epochStoreKey, epochMemStoreKey, epochparamsSubspace, &ks.BankKeeper, &ks.AccountKeeper, ks.Spec)
	ks.Pairing = *pairingkeeper.NewKeeper(cdc, pairingStoreKey, pairingMemStoreKey, pairingparamsSubspace, &ks.BankKeeper, &ks.AccountKeeper, ks.Spec, &ks.Epochstorage)
	ks.ParamsKeeper = paramsKeeper
	ks.Conflict = *conflictkeeper.NewKeeper(cdc, conflictStoreKey, conflictMemStoreKey, conflictparamsSubspace, &ks.BankKeeper, &ks.AccountKeeper, ks.Pairing, ks.Epochstorage, ks.Spec)
	ks.BlockStore = MockBlockStore{height: 0, blockHistory: make(map[int64]*tenderminttypes.Block)}

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.TestingLogger())

	// Initialize params
	ks.Pairing.SetParams(ctx, pairingtypes.DefaultParams())
	ks.Spec.SetParams(ctx, spectypes.DefaultParams())
	ks.Epochstorage.SetParams(ctx, epochstoragetypes.DefaultParams())
	ks.Conflict.SetParams(ctx, conflicttypes.DefaultParams())

	ks.Epochstorage.PushFixatedParams(ctx, 0, 0)

	ss := Servers{}
	ss.EpochServer = epochstoragekeeper.NewMsgServerImpl(ks.Epochstorage)
	ss.SpecServer = speckeeper.NewMsgServerImpl(ks.Spec)
	ss.PairingServer = pairingkeeper.NewMsgServerImpl(ks.Pairing)
	ss.ConflictServer = conflictkeeper.NewMsgServerImpl(ks.Conflict)

	core.SetEnvironment(&core.Environment{BlockStore: &ks.BlockStore})

	ks.Epochstorage.SetEpochDetails(ctx, *epochstoragetypes.DefaultGenesis().EpochDetails)
	NewBlock(sdk.WrapSDKContext(ctx), &ks)
	return &ss, &ks, sdk.WrapSDKContext(ctx)
}

func AdvanceBlock(ctx context.Context, ks *Keepers, customBlockTime ...time.Duration) context.Context {
	unwrapedCtx := sdk.UnwrapSDKContext(ctx)

	block := uint64(unwrapedCtx.BlockHeight() + 1)
	unwrapedCtx = unwrapedCtx.WithBlockHeight(int64(block))
	if len(customBlockTime) > 0 {
		NewBlock(sdk.WrapSDKContext(unwrapedCtx), ks, customBlockTime...)
	} else {
		NewBlock(sdk.WrapSDKContext(unwrapedCtx), ks)
	}
	return sdk.WrapSDKContext(unwrapedCtx)
}

func AdvanceBlocks(ctx context.Context, ks *Keepers, blocks int, customBlockTime ...time.Duration) context.Context {
	if len(customBlockTime) > 0 {
		for i := 0; i < blocks; i++ {
			ctx = AdvanceBlock(ctx, ks, customBlockTime...)
		}
	} else {
		for i := 0; i < blocks; i++ {
			ctx = AdvanceBlock(ctx, ks)
		}
	}

	return ctx
}

func AdvanceToBlock(ctx context.Context, ks *Keepers, block uint64, customBlockTime ...time.Duration) context.Context {
	unwrapedCtx := sdk.UnwrapSDKContext(ctx)
	if uint64(unwrapedCtx.BlockHeight()) == block {
		return ctx
	}

	if len(customBlockTime) > 0 {
		for uint64(unwrapedCtx.BlockHeight()) < block {
			ctx = AdvanceBlock(ctx, ks, customBlockTime...)
			unwrapedCtx = sdk.UnwrapSDKContext(ctx)
		}
	} else {
		for uint64(unwrapedCtx.BlockHeight()) < block {
			ctx = AdvanceBlock(ctx, ks)
			unwrapedCtx = sdk.UnwrapSDKContext(ctx)
		}
	}

	return ctx
}

// Make sure you save the new context
func AdvanceEpoch(ctx context.Context, ks *Keepers, customBlockTime ...time.Duration) context.Context {
	unwrapedCtx := sdk.UnwrapSDKContext(ctx)

	nextEpochBlockNum, err := ks.Epochstorage.GetNextEpoch(unwrapedCtx, ks.Epochstorage.GetEpochStart(unwrapedCtx))
	if err != nil {
		panic(err)
	}
	if len(customBlockTime) > 0 {
		return AdvanceToBlock(ctx, ks, nextEpochBlockNum, customBlockTime...)
	}
	return AdvanceToBlock(ctx, ks, nextEpochBlockNum)
}

// Make sure you save the new context
func NewBlock(ctx context.Context, ks *Keepers, customTime ...time.Duration) {
	unwrapedCtx := sdk.UnwrapSDKContext(ctx)
	block := uint64(unwrapedCtx.BlockHeight())
	if ks.Epochstorage.IsEpochStart(sdk.UnwrapSDKContext(ctx)) {
		ks.Epochstorage.FixateParams(unwrapedCtx, block)
		// begin block
		ks.Epochstorage.SetEpochDetailsStart(unwrapedCtx, block)
		ks.Epochstorage.StoreCurrentEpochStakeStorage(unwrapedCtx, block, epochstoragetypes.ProviderKey)
		ks.Epochstorage.StoreCurrentEpochStakeStorage(unwrapedCtx, block, epochstoragetypes.ClientKey)

		ks.Epochstorage.UpdateEarliestEpochstart(unwrapedCtx)
		ks.Epochstorage.RemoveOldEpochData(unwrapedCtx, epochstoragetypes.ProviderKey)
		ks.Epochstorage.RemoveOldEpochData(unwrapedCtx, epochstoragetypes.ClientKey)

		ks.Pairing.RemoveOldEpochPayment(unwrapedCtx)
		ks.Pairing.CheckUnstakingForCommit(unwrapedCtx)

		start := time.Now()
		ks.Pairing.UnstakeUnresponsiveProviders(unwrapedCtx, EPOCHS_NUM_TO_CHECK_CU_FOR_UNRESPONSIVE_PROVIDER, EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS)
		elapsed_time := time.Since(start)
		fmt.Printf("%v, ", elapsed_time)
	}

	ks.Conflict.CheckAndHandleAllVotes(unwrapedCtx)

	if len(customTime) > 0 {
		ks.BlockStore.AdvanceBlock(customTime[0])
	} else {
		ks.BlockStore.AdvanceBlock(BLOCK_TIME)
	}
}
