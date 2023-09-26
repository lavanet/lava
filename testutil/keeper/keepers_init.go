package keeper

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	tmdb "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cometbft/cometbft/rpc/core"
	tenderminttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	"github.com/lavanet/lava/common"
	"github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/utils/sigs"
	conflictkeeper "github.com/lavanet/lava/x/conflict/keeper"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	downtimekeeper "github.com/lavanet/lava/x/downtime/keeper"
	downtimemoduletypes "github.com/lavanet/lava/x/downtime/types"
	downtimev1 "github.com/lavanet/lava/x/downtime/v1"
	dualstakingkeeper "github.com/lavanet/lava/x/dualstaking/keeper"
	dualstakingtypes "github.com/lavanet/lava/x/dualstaking/types"
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
	protocolkeeper "github.com/lavanet/lava/x/protocol/keeper"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
	"github.com/lavanet/lava/x/spec"
	speckeeper "github.com/lavanet/lava/x/spec/keeper"
	spectypes "github.com/lavanet/lava/x/spec/types"
	subscriptionkeeper "github.com/lavanet/lava/x/subscription/keeper"
	subscriptiontypes "github.com/lavanet/lava/x/subscription/types"
	"github.com/stretchr/testify/require"
)

const (
	BLOCK_TIME       = 30 * time.Second
	BLOCK_HEADER_LEN = 32
)

var Randomizer *sigs.ZeroReader

// NOTE: the order of the keeper fields must follow that of calling app.mm.SetOrderBeginBlockers() in app/app.go
type Keepers struct {
	AccountKeeper mockAccountKeeper
	BankKeeper    mockBankKeeper
	StakingKeeper mockStakingKeeper
	Spec          speckeeper.Keeper
	Epochstorage  epochstoragekeeper.Keeper
	Dualstaking   dualstakingkeeper.Keeper
	Subscription  subscriptionkeeper.Keeper
	Conflict      conflictkeeper.Keeper
	Pairing       pairingkeeper.Keeper
	Projects      projectskeeper.Keeper
	Plans         planskeeper.Keeper
	Protocol      protocolkeeper.Keeper
	ParamsKeeper  paramskeeper.Keeper
	BlockStore    MockBlockStore
	Downtime      downtimekeeper.Keeper
}

type Servers struct {
	EpochServer        epochstoragetypes.MsgServer
	SpecServer         spectypes.MsgServer
	PairingServer      pairingtypes.MsgServer
	ConflictServer     conflicttypes.MsgServer
	ProjectServer      projectstypes.MsgServer
	ProtocolServer     protocoltypes.MsgServer
	SubscriptionServer subscriptiontypes.MsgServer
	DualstakingServer  dualstakingtypes.MsgServer
	PlansServer        planstypes.MsgServer
}

type KeeperBeginBlocker interface {
	BeginBlock(ctx sdk.Context)
}

func InitAllKeepers(t testing.TB) (*Servers, *Keepers, context.Context) {
	seed := time.Now().Unix()
	// seed = 1695297312 // uncomment this to debug a specific scenario
	Randomizer = sigs.NewZeroReader(seed)
	fmt.Println("Reproduce With testing seed: ", seed)

	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	pairingStoreKey := sdk.NewKVStoreKey(pairingtypes.StoreKey)
	pairingMemStoreKey := storetypes.NewMemoryStoreKey(pairingtypes.MemStoreKey)
	stateStore.MountStoreWithDB(pairingStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(pairingMemStoreKey, storetypes.StoreTypeMemory, nil)

	specStoreKey := sdk.NewKVStoreKey(spectypes.StoreKey)
	specMemStoreKey := storetypes.NewMemoryStoreKey(spectypes.MemStoreKey)
	stateStore.MountStoreWithDB(specStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(specMemStoreKey, storetypes.StoreTypeMemory, nil)

	plansStoreKey := sdk.NewKVStoreKey(planstypes.StoreKey)
	plansMemStoreKey := storetypes.NewMemoryStoreKey(planstypes.MemStoreKey)
	stateStore.MountStoreWithDB(plansStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(plansMemStoreKey, storetypes.StoreTypeMemory, nil)

	projectsStoreKey := sdk.NewKVStoreKey(projectstypes.StoreKey)
	projectsMemStoreKey := storetypes.NewMemoryStoreKey(projectstypes.MemStoreKey)
	stateStore.MountStoreWithDB(projectsStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(projectsMemStoreKey, storetypes.StoreTypeMemory, nil)

	protocolStoreKey := sdk.NewKVStoreKey(protocoltypes.StoreKey)
	protocolMemStoreKey := storetypes.NewMemoryStoreKey(protocoltypes.MemStoreKey)
	stateStore.MountStoreWithDB(protocolStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(protocolMemStoreKey, storetypes.StoreTypeMemory, nil)

	subscriptionStoreKey := sdk.NewKVStoreKey(subscriptiontypes.StoreKey)
	subscriptionMemStoreKey := storetypes.NewMemoryStoreKey(subscriptiontypes.MemStoreKey)
	stateStore.MountStoreWithDB(subscriptionStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(subscriptionMemStoreKey, storetypes.StoreTypeMemory, nil)

	dualstakingStoreKey := sdk.NewKVStoreKey(dualstakingtypes.StoreKey)
	dualstakingMemStoreKey := storetypes.NewMemoryStoreKey(dualstakingtypes.MemStoreKey)
	stateStore.MountStoreWithDB(dualstakingStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(dualstakingMemStoreKey, storetypes.StoreTypeMemory, nil)

	epochStoreKey := sdk.NewKVStoreKey(epochstoragetypes.StoreKey)
	epochMemStoreKey := storetypes.NewMemoryStoreKey(epochstoragetypes.MemStoreKey)
	stateStore.MountStoreWithDB(epochStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(epochMemStoreKey, storetypes.StoreTypeMemory, nil)

	paramsStoreKey := sdk.NewKVStoreKey(paramstypes.StoreKey)
	stateStore.MountStoreWithDB(paramsStoreKey, storetypes.StoreTypeIAVL, db)
	tkey := sdk.NewTransientStoreKey(paramstypes.TStoreKey)
	stateStore.MountStoreWithDB(tkey, storetypes.StoreTypeIAVL, db)

	conflictStoreKey := sdk.NewKVStoreKey(conflicttypes.StoreKey)
	conflictMemStoreKey := storetypes.NewMemoryStoreKey(conflicttypes.MemStoreKey)
	stateStore.MountStoreWithDB(conflictStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(conflictMemStoreKey, storetypes.StoreTypeMemory, nil)

	downtimeKey := sdk.NewKVStoreKey(downtimemoduletypes.StoreKey)
	stateStore.MountStoreWithDB(downtimeKey, storetypes.StoreTypeIAVL, db)

	require.NoError(t, stateStore.LoadLatestVersion())

	paramsKeeper := paramskeeper.NewKeeper(cdc, pairingtypes.Amino, paramsStoreKey, tkey)
	paramsKeeper.Subspace(spectypes.ModuleName)
	paramsKeeper.Subspace(epochstoragetypes.ModuleName)
	paramsKeeper.Subspace(pairingtypes.ModuleName)
	paramsKeeper.Subspace(protocoltypes.ModuleName)
	paramsKeeper.Subspace(downtimemoduletypes.ModuleName)
	// paramsKeeper.Subspace(conflicttypes.ModuleName) //TODO...

	epochparamsSubspace, _ := paramsKeeper.GetSubspace(epochstoragetypes.ModuleName)

	pairingparamsSubspace, _ := paramsKeeper.GetSubspace(pairingtypes.ModuleName)

	specparamsSubspace, _ := paramsKeeper.GetSubspace(spectypes.ModuleName)

	protocolparamsSubspace, _ := paramsKeeper.GetSubspace(protocoltypes.ModuleName)

	projectsparamsSubspace, _ := paramsKeeper.GetSubspace(projectstypes.ModuleName)

	plansparamsSubspace, _ := paramsKeeper.GetSubspace(planstypes.ModuleName)

	subscriptionparamsSubspace, _ := paramsKeeper.GetSubspace(subscriptiontypes.ModuleName)

	dualstakingparamsSubspace, _ := paramsKeeper.GetSubspace(dualstakingtypes.ModuleName)

	conflictparamsSubspace := paramstypes.NewSubspace(cdc,
		conflicttypes.Amino,
		conflictStoreKey,
		conflictMemStoreKey,
		"ConflictParams",
	)

	downtimeParamsSubspace, _ := paramsKeeper.GetSubspace(downtimemoduletypes.ModuleName)

	ks := Keepers{}
	ks.AccountKeeper = mockAccountKeeper{}
	ks.BankKeeper = mockBankKeeper{balance: make(map[string]sdk.Coins)}
	ks.StakingKeeper = mockStakingKeeper{}
	ks.Spec = *speckeeper.NewKeeper(cdc, specStoreKey, specMemStoreKey, specparamsSubspace)
	ks.Epochstorage = *epochstoragekeeper.NewKeeper(cdc, epochStoreKey, epochMemStoreKey, epochparamsSubspace, &ks.BankKeeper, &ks.AccountKeeper, ks.Spec)
	ks.Dualstaking = *dualstakingkeeper.NewKeeper(cdc, dualstakingStoreKey, dualstakingMemStoreKey, dualstakingparamsSubspace, &ks.BankKeeper, &ks.AccountKeeper, ks.Epochstorage, ks.Spec)
	ks.Plans = *planskeeper.NewKeeper(cdc, plansStoreKey, plansMemStoreKey, plansparamsSubspace, ks.Epochstorage, ks.Spec)
	ks.Projects = *projectskeeper.NewKeeper(cdc, projectsStoreKey, projectsMemStoreKey, projectsparamsSubspace, ks.Epochstorage)
	ks.Protocol = *protocolkeeper.NewKeeper(cdc, protocolStoreKey, protocolMemStoreKey, protocolparamsSubspace)
	ks.Subscription = *subscriptionkeeper.NewKeeper(cdc, subscriptionStoreKey, subscriptionMemStoreKey, subscriptionparamsSubspace, &ks.BankKeeper, &ks.AccountKeeper, &ks.Epochstorage, ks.Projects, ks.Plans)
	ks.Downtime = downtimekeeper.NewKeeper(cdc, downtimeKey, downtimeParamsSubspace, ks.Epochstorage)
	ks.Pairing = *pairingkeeper.NewKeeper(cdc, pairingStoreKey, pairingMemStoreKey, pairingparamsSubspace, &ks.BankKeeper, &ks.AccountKeeper, ks.Spec, &ks.Epochstorage, ks.Projects, ks.Subscription, ks.Plans, ks.Downtime, ks.Dualstaking)
	ks.ParamsKeeper = paramsKeeper
	ks.Conflict = *conflictkeeper.NewKeeper(cdc, conflictStoreKey, conflictMemStoreKey, conflictparamsSubspace, &ks.BankKeeper, &ks.AccountKeeper, ks.Pairing, ks.Epochstorage, ks.Spec)
	ks.BlockStore = MockBlockStore{height: 0, blockHistory: make(map[int64]*tenderminttypes.Block)}

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	ks.Pairing.SetParams(ctx, pairingtypes.DefaultParams())
	ks.Spec.SetParams(ctx, spectypes.DefaultParams())
	ks.Subscription.SetParams(ctx, subscriptiontypes.DefaultParams())
	ks.Epochstorage.SetParams(ctx, epochstoragetypes.DefaultParams())
	ks.Dualstaking.SetParams(ctx, dualstakingtypes.DefaultParams())
	ks.Conflict.SetParams(ctx, conflicttypes.DefaultParams())
	ks.Projects.SetParams(ctx, projectstypes.DefaultParams())
	protocolParams := protocoltypes.DefaultParams()
	protocolParams.Version = protocoltypes.Version{ProviderTarget: "0.0.1", ProviderMin: "0.0.0", ConsumerTarget: "0.0.1", ConsumerMin: "0.0.0"}
	protocoltypes.UpdateLatestParams(protocolParams)
	ks.Protocol.SetParams(ctx, protocolParams)
	ks.Plans.SetParams(ctx, planstypes.DefaultParams())
	ks.Downtime.SetParams(ctx, downtimev1.DefaultParams())

	ks.Epochstorage.PushFixatedParams(ctx, 0, 0)

	ss := Servers{}
	ss.EpochServer = epochstoragekeeper.NewMsgServerImpl(ks.Epochstorage)
	ss.SpecServer = speckeeper.NewMsgServerImpl(ks.Spec)
	ss.PlansServer = planskeeper.NewMsgServerImpl(ks.Plans)
	ss.PairingServer = pairingkeeper.NewMsgServerImpl(ks.Pairing)
	ss.ConflictServer = conflictkeeper.NewMsgServerImpl(ks.Conflict)
	ss.ProjectServer = projectskeeper.NewMsgServerImpl(ks.Projects)
	ss.ProtocolServer = protocolkeeper.NewMsgServerImpl(ks.Protocol)
	ss.SubscriptionServer = subscriptionkeeper.NewMsgServerImpl(ks.Subscription)
	ss.DualstakingServer = dualstakingkeeper.NewMsgServerImpl(ks.Dualstaking)

	core.SetEnvironment(&core.Environment{BlockStore: &ks.BlockStore})

	ks.Epochstorage.SetEpochDetails(ctx, *epochstoragetypes.DefaultGenesis().EpochDetails)

	ks.Dualstaking.InitDelegations(ctx, *common.DefaultGenesis())
	ks.Dualstaking.InitDelegators(ctx, *common.DefaultGenesis())
	ks.Dualstaking.InitUnbondings(ctx, []types.RawMessage{})
	ks.Plans.InitPlans(ctx, *common.DefaultGenesis())
	ks.Subscription.InitSubscriptions(ctx, *common.DefaultGenesis())
	ks.Subscription.InitSubscriptionsTimers(ctx, []types.RawMessage{})
	ks.Projects.InitDevelopers(ctx, *common.DefaultGenesis())
	ks.Projects.InitProjects(ctx, *common.DefaultGenesis())

	NewBlock(ctx, &ks)
	ctx = ctx.WithBlockTime(time.Now())

	return &ss, &ks, sdk.WrapSDKContext(ctx)
}

func SimulateParamChange(ctx sdk.Context, paramKeeper paramskeeper.Keeper, subspace, key, value string) (err error) {
	proposal := &paramproposal.ParameterChangeProposal{Changes: []paramproposal.ParamChange{{Subspace: subspace, Key: key, Value: value}}}
	err = spec.HandleParameterChangeProposal(ctx, paramKeeper, proposal)
	return
}

func SimulatePlansAddProposal(ctx sdk.Context, plansKeeper planskeeper.Keeper, plansToPropose []planstypes.Plan) error {
	proposal := planstypes.NewPlansAddProposal("mockProposal", "mockProposal plans add for testing", plansToPropose)
	err := proposal.ValidateBasic()
	if err != nil {
		return err
	}
	proposalHandler := plans.NewPlansProposalsHandler(plansKeeper)
	err = proposalHandler(ctx, proposal)
	return err
}

func SimulatePlansDelProposal(ctx sdk.Context, plansKeeper planskeeper.Keeper, plansToDelete []string) error {
	proposal := planstypes.NewPlansDelProposal("mockProposal", "mockProposal plans delete for testing", plansToDelete)
	err := proposal.ValidateBasic()
	if err != nil {
		return err
	}
	proposalHandler := plans.NewPlansProposalsHandler(plansKeeper)
	err = proposalHandler(ctx, proposal)
	return err
}

func SimulateSpecAddProposal(ctx sdk.Context, specKeeper speckeeper.Keeper, specsToPropose []spectypes.Spec) error {
	proposal := spectypes.NewSpecAddProposal("mockProposal", "mockProposal specs add for testing", specsToPropose)
	err := proposal.ValidateBasic()
	if err != nil {
		return err
	}
	proposalHandler := spec.NewSpecProposalsHandler(specKeeper)
	err = proposalHandler(ctx, proposal)
	return err
}

func SimulateUnstakeProposal(ctx sdk.Context, pairingKeeper pairingkeeper.Keeper, providersInfo []pairingtypes.ProviderUnstakeInfo) error {
	proposal := pairingtypes.NewUnstakeProposal("mockProposal", "mockProposal unstake provider for testing", providersInfo)
	err := proposal.ValidateBasic()
	if err != nil {
		return err
	}
	proposalHandler := pairing.NewPairingProposalsHandler(pairingKeeper)
	err = proposalHandler(ctx, proposal)
	return err
}

func AdvanceBlock(ctx context.Context, ks *Keepers, customBlockTime ...time.Duration) context.Context {
	unwrapedCtx := sdk.UnwrapSDKContext(ctx)

	block := uint64(unwrapedCtx.BlockHeight() + 1)
	unwrapedCtx = unwrapedCtx.WithBlockHeight(int64(block))

	headerHash := make([]byte, BLOCK_HEADER_LEN)
	Randomizer.Read(headerHash)
	unwrapedCtx = unwrapedCtx.WithHeaderHash(headerHash)

	NewBlock(unwrapedCtx, ks)

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
func NewBlock(ctx sdk.Context, ks *Keepers) {
	// get the value and type of the Keepers struct
	keepersType := reflect.TypeOf(*ks)
	keepersValue := reflect.ValueOf(*ks)

	// iterate over all keepers and call BeginBlock (if it's implemented by the keeper)
	for i := 0; i < keepersType.NumField(); i++ {
		fieldValue := keepersValue.Field(i)

		if beginBlocker, ok := fieldValue.Interface().(KeeperBeginBlocker); ok {
			beginBlocker.BeginBlock(ctx)
		}
	}
}
