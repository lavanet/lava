package keeper

import (
	"context"
	"crypto/rand"
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
	"github.com/lavanet/lava/x/pairing"
	pairingkeeper "github.com/lavanet/lava/x/pairing/keeper"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/lavanet/lava/x/plans"
	planskeeper "github.com/lavanet/lava/x/plans/keeper"
	planstypes "github.com/lavanet/lava/x/plans/types"
	projectskeeper "github.com/lavanet/lava/x/projects/keeper"
	projectstypes "github.com/lavanet/lava/x/projects/types"
	"github.com/lavanet/lava/x/spec"
	speckeeper "github.com/lavanet/lava/x/spec/keeper"
	spectypes "github.com/lavanet/lava/x/spec/types"
	subscriptionkeeper "github.com/lavanet/lava/x/subscription/keeper"
	subscriptiontypes "github.com/lavanet/lava/x/subscription/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"github.com/tendermint/tendermint/rpc/core"
	tenderminttypes "github.com/tendermint/tendermint/types"
	tmdb "github.com/tendermint/tm-db"
)

const (
	BLOCK_TIME = 30 * time.Second
)

const BLOCK_HEADER_LEN = 32

type Keepers struct {
	Epochstorage  epochstoragekeeper.Keeper
	Spec          speckeeper.Keeper
	Plans         planskeeper.Keeper
	Projects      projectskeeper.Keeper
	Subscription  subscriptionkeeper.Keeper
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
	ProjectServer  projectstypes.MsgServer
	PlansServer    planstypes.MsgServer
}

func SimulateParamChange(ctx sdk.Context, paramKeeper paramskeeper.Keeper, subspace string, key string, value string) (err error) {
	proposal := &paramproposal.ParameterChangeProposal{Changes: []paramproposal.ParamChange{{Subspace: subspace, Key: key, Value: value}}}
	err = spec.HandleParameterChangeProposal(ctx, paramKeeper, proposal)
	return
}

func SimulatePlansProposal(ctx sdk.Context, plansKeeper planskeeper.Keeper, plansToPropose []planstypes.Plan) error {
	proposal := planstypes.NewPlansAddProposal("mockProposal", "mockProposal for testing", plansToPropose)
	err := proposal.ValidateBasic()
	if err != nil {
		return err
	}
	proposalHandler := plans.NewPlansProposalsHandler(plansKeeper)
	err = proposalHandler(ctx, proposal)
	return err
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

	plansStoreKey := sdk.NewKVStoreKey(planstypes.StoreKey)
	plansMemStoreKey := storetypes.NewMemoryStoreKey(planstypes.MemStoreKey)
	stateStore.MountStoreWithDB(plansStoreKey, sdk.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(plansMemStoreKey, sdk.StoreTypeMemory, nil)

	projectsStoreKey := sdk.NewKVStoreKey(projectstypes.StoreKey)
	projectsMemStoreKey := storetypes.NewMemoryStoreKey(projectstypes.MemStoreKey)
	stateStore.MountStoreWithDB(projectsStoreKey, sdk.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(projectsMemStoreKey, sdk.StoreTypeMemory, nil)

	subscriptionStoreKey := sdk.NewKVStoreKey(subscriptiontypes.StoreKey)
	subscriptionMemStoreKey := storetypes.NewMemoryStoreKey(subscriptiontypes.MemStoreKey)
	stateStore.MountStoreWithDB(subscriptionStoreKey, sdk.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(subscriptionMemStoreKey, sdk.StoreTypeMemory, nil)

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

	projectsparamsSubspace, _ := paramsKeeper.GetSubspace(projectstypes.ModuleName)

	plansparamsSubspace, _ := paramsKeeper.GetSubspace(planstypes.ModuleName)

	subscriptionparamsSubspace, _ := paramsKeeper.GetSubspace(subscriptiontypes.ModuleName)

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
	ks.Projects = *projectskeeper.NewKeeper(cdc, projectsStoreKey, projectsMemStoreKey, projectsparamsSubspace)
	ks.Epochstorage = *epochstoragekeeper.NewKeeper(cdc, epochStoreKey, epochMemStoreKey, epochparamsSubspace, &ks.BankKeeper, &ks.AccountKeeper, ks.Spec)
	ks.Plans = *planskeeper.NewKeeper(cdc, plansStoreKey, plansMemStoreKey, plansparamsSubspace)
	ks.Projects = *projectskeeper.NewKeeper(cdc, projectsStoreKey, projectsMemStoreKey, projectsparamsSubspace)
	ks.Subscription = *subscriptionkeeper.NewKeeper(cdc, subscriptionStoreKey, subscriptionMemStoreKey, subscriptionparamsSubspace, &ks.BankKeeper, &ks.AccountKeeper, &ks.Epochstorage, ks.Projects, ks.Plans)
	ks.Pairing = *pairingkeeper.NewKeeper(cdc, pairingStoreKey, pairingMemStoreKey, pairingparamsSubspace, &ks.BankKeeper, &ks.AccountKeeper, ks.Spec, &ks.Epochstorage)
	ks.ParamsKeeper = paramsKeeper
	ks.Conflict = *conflictkeeper.NewKeeper(cdc, conflictStoreKey, conflictMemStoreKey, conflictparamsSubspace, &ks.BankKeeper, &ks.AccountKeeper, ks.Pairing, ks.Epochstorage, ks.Spec)
	ks.BlockStore = MockBlockStore{height: 0, blockHistory: make(map[int64]*tenderminttypes.Block)}

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.TestingLogger())

	// Initialize params
	ks.Pairing.SetParams(ctx, pairingtypes.DefaultParams())
	ks.Spec.SetParams(ctx, spectypes.DefaultParams())
	ks.Subscription.SetParams(ctx, subscriptiontypes.DefaultParams())
	ks.Epochstorage.SetParams(ctx, epochstoragetypes.DefaultParams())
	ks.Conflict.SetParams(ctx, conflicttypes.DefaultParams())
	ks.Projects.SetParams(ctx, projectstypes.DefaultParams())
	ks.Plans.SetParams(ctx, planstypes.DefaultParams())

	ks.Epochstorage.PushFixatedParams(ctx, 0, 0)

	ss := Servers{}
	ss.EpochServer = epochstoragekeeper.NewMsgServerImpl(ks.Epochstorage)
	ss.SpecServer = speckeeper.NewMsgServerImpl(ks.Spec)
	ss.PlansServer = planskeeper.NewMsgServerImpl(ks.Plans)
	ss.PairingServer = pairingkeeper.NewMsgServerImpl(ks.Pairing)
	ss.ConflictServer = conflictkeeper.NewMsgServerImpl(ks.Conflict)
	ss.ProjectServer = projectskeeper.NewMsgServerImpl(ks.Projects)

	core.SetEnvironment(&core.Environment{BlockStore: &ks.BlockStore})

	ks.Epochstorage.SetEpochDetails(ctx, *epochstoragetypes.DefaultGenesis().EpochDetails)
	NewBlock(sdk.WrapSDKContext(ctx), &ks)
	return &ss, &ks, sdk.WrapSDKContext(ctx)
}

func AdvanceBlock(ctx context.Context, ks *Keepers, customBlockTime ...time.Duration) context.Context {
	unwrapedCtx := sdk.UnwrapSDKContext(ctx)

	block := uint64(unwrapedCtx.BlockHeight() + 1)
	unwrapedCtx = unwrapedCtx.WithBlockHeight(int64(block))

	headerHash := make([]byte, BLOCK_HEADER_LEN)
	rand.Read(headerHash)
	unwrapedCtx = unwrapedCtx.WithHeaderHash(headerHash)

	NewBlock(sdk.WrapSDKContext(unwrapedCtx), ks)

	if len(customBlockTime) > 0 {
		ks.BlockStore.AdvanceBlock(customBlockTime[0])
	} else {
		ks.BlockStore.AdvanceBlock(BLOCK_TIME)
	}

	b := ks.BlockStore.LoadBlock(int64(block))
	unwrapedCtx = unwrapedCtx.WithBlockTime(b.Header.Time)

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
func NewBlock(ctx context.Context, ks *Keepers) {
	unwrapedCtx := sdk.UnwrapSDKContext(ctx)
	if ks.Epochstorage.IsEpochStart(sdk.UnwrapSDKContext(ctx)) {
		ks.Epochstorage.EpochStart(unwrapedCtx)
		ks.Pairing.EpochStart(unwrapedCtx, pairing.EPOCHS_NUM_TO_CHECK_CU_FOR_UNRESPONSIVE_PROVIDER, pairing.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS)
	}

	ks.Conflict.CheckAndHandleAllVotes(unwrapedCtx)
}
