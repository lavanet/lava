package keeper

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"cosmossdk.io/math"
	tmdb "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cometbft/cometbft/rpc/core"
	tenderminttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	distributionkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	commonconsts "github.com/lavanet/lava/v2/testutil/common/consts"
	"github.com/lavanet/lava/v2/utils/sigs"
	conflictkeeper "github.com/lavanet/lava/v2/x/conflict/keeper"
	conflicttypes "github.com/lavanet/lava/v2/x/conflict/types"
	downtimekeeper "github.com/lavanet/lava/v2/x/downtime/keeper"
	downtimemoduletypes "github.com/lavanet/lava/v2/x/downtime/types"
	downtimev1 "github.com/lavanet/lava/v2/x/downtime/v1"
	dualstakingkeeper "github.com/lavanet/lava/v2/x/dualstaking/keeper"
	dualstakingtypes "github.com/lavanet/lava/v2/x/dualstaking/types"
	epochstoragekeeper "github.com/lavanet/lava/v2/x/epochstorage/keeper"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	fixationkeeper "github.com/lavanet/lava/v2/x/fixationstore/keeper"
	fixationtypes "github.com/lavanet/lava/v2/x/fixationstore/types"
	"github.com/lavanet/lava/v2/x/pairing"
	pairingkeeper "github.com/lavanet/lava/v2/x/pairing/keeper"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
	"github.com/lavanet/lava/v2/x/plans"
	planskeeper "github.com/lavanet/lava/v2/x/plans/keeper"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
	projectskeeper "github.com/lavanet/lava/v2/x/projects/keeper"
	projectstypes "github.com/lavanet/lava/v2/x/projects/types"
	protocolkeeper "github.com/lavanet/lava/v2/x/protocol/keeper"
	protocoltypes "github.com/lavanet/lava/v2/x/protocol/types"
	rewardskeeper "github.com/lavanet/lava/v2/x/rewards/keeper"
	rewardstypes "github.com/lavanet/lava/v2/x/rewards/types"
	"github.com/lavanet/lava/v2/x/spec"
	speckeeper "github.com/lavanet/lava/v2/x/spec/keeper"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
	subscriptionkeeper "github.com/lavanet/lava/v2/x/subscription/keeper"
	subscriptiontypes "github.com/lavanet/lava/v2/x/subscription/types"
	timerstorekeeper "github.com/lavanet/lava/v2/x/timerstore/keeper"
	timerstoretypes "github.com/lavanet/lava/v2/x/timerstore/types"
	"github.com/stretchr/testify/require"
)

const (
	BLOCK_TIME       = 30 * time.Second
	BLOCK_HEADER_LEN = 32
)

var Randomizer *sigs.ZeroReader

// NOTE: the order of the keeper fields must follow that of calling app.mm.SetOrderBeginBlockers() in app/app.go
type Keepers struct {
	TimerStoreKeeper    *timerstorekeeper.Keeper
	FixationStoreKeeper *fixationkeeper.Keeper
	AccountKeeper       mockAccountKeeper
	BankKeeper          mockBankKeeper
	StakingKeeper       stakingkeeper.Keeper
	Spec                speckeeper.Keeper
	Epochstorage        epochstoragekeeper.Keeper
	Dualstaking         dualstakingkeeper.Keeper
	Subscription        subscriptionkeeper.Keeper
	Conflict            conflictkeeper.Keeper
	Pairing             pairingkeeper.Keeper
	Projects            projectskeeper.Keeper
	Plans               planskeeper.Keeper
	Protocol            protocolkeeper.Keeper
	ParamsKeeper        paramskeeper.Keeper
	BlockStore          MockBlockStore
	Downtime            downtimekeeper.Keeper
	SlashingKeeper      slashingkeeper.Keeper
	Rewards             rewardskeeper.Keeper
	Distribution        distributionkeeper.Keeper
}

type Servers struct {
	StakingServer      stakingtypes.MsgServer
	EpochServer        epochstoragetypes.MsgServer
	SpecServer         spectypes.MsgServer
	PairingServer      pairingtypes.MsgServer
	ConflictServer     conflicttypes.MsgServer
	ProjectServer      projectstypes.MsgServer
	ProtocolServer     protocoltypes.MsgServer
	SubscriptionServer subscriptiontypes.MsgServer
	DualstakingServer  dualstakingtypes.MsgServer
	PlansServer        planstypes.MsgServer
	SlashingServer     slashingtypes.MsgServer
	RewardsServer      rewardstypes.MsgServer
	DistributionServer distributiontypes.MsgServer
}

type KeeperBeginBlockerWithRequest interface {
	BeginBlock(ctx sdk.Context, req abci.RequestBeginBlock)
}

type KeeperBeginBlocker interface {
	BeginBlock(ctx sdk.Context)
}

type KeeperEndBlocker interface {
	EndBlock(ctx sdk.Context)
}

func InitAllKeepers(t testing.TB) (*Servers, *Keepers, context.Context) {
	seed := time.Now().Unix()
	// seed = 1695297312 // uncomment this to debug a specific scenario
	Randomizer = sigs.NewZeroReader(seed)
	fmt.Println("Reproduce With testing seed: ", seed)

	db := tmdb.NewMemDB()
	stateStore := store.NewCommitMultiStore(db)

	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	legacyCdc := codec.NewLegacyAmino()

	distributionStoreKey := sdk.NewKVStoreKey(distributiontypes.StoreKey)
	stateStore.MountStoreWithDB(distributionStoreKey, storetypes.StoreTypeIAVL, db)

	stakingStoreKey := sdk.NewKVStoreKey(stakingtypes.StoreKey)
	stateStore.MountStoreWithDB(stakingStoreKey, storetypes.StoreTypeIAVL, db)

	slashingStoreKey := sdk.NewKVStoreKey(slashingtypes.StoreKey)
	stateStore.MountStoreWithDB(slashingStoreKey, storetypes.StoreTypeIAVL, db)

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

	rewardsStoreKey := sdk.NewKVStoreKey(rewardstypes.StoreKey)
	rewardsMemStoreKey := storetypes.NewMemoryStoreKey(rewardstypes.MemStoreKey)
	stateStore.MountStoreWithDB(rewardsStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(rewardsMemStoreKey, storetypes.StoreTypeMemory, nil)

	require.NoError(t, stateStore.LoadLatestVersion())

	paramsKeeper := paramskeeper.NewKeeper(cdc, pairingtypes.Amino, paramsStoreKey, tkey)
	paramsKeeper.Subspace(spectypes.ModuleName)
	paramsKeeper.Subspace(epochstoragetypes.ModuleName)
	paramsKeeper.Subspace(pairingtypes.ModuleName)
	paramsKeeper.Subspace(protocoltypes.ModuleName)
	paramsKeeper.Subspace(downtimemoduletypes.ModuleName)
	paramsKeeper.Subspace(rewardstypes.ModuleName)
	paramsKeeper.Subspace(distributiontypes.ModuleName)
	paramsKeeper.Subspace(dualstakingtypes.ModuleName)
	// paramsKeeper.Subspace(conflicttypes.ModuleName) //TODO...

	epochparamsSubspace, _ := paramsKeeper.GetSubspace(epochstoragetypes.ModuleName)

	pairingparamsSubspace, _ := paramsKeeper.GetSubspace(pairingtypes.ModuleName)

	specparamsSubspace, _ := paramsKeeper.GetSubspace(spectypes.ModuleName)

	protocolparamsSubspace, _ := paramsKeeper.GetSubspace(protocoltypes.ModuleName)

	projectsparamsSubspace, _ := paramsKeeper.GetSubspace(projectstypes.ModuleName)

	plansparamsSubspace, _ := paramsKeeper.GetSubspace(planstypes.ModuleName)

	subscriptionparamsSubspace, _ := paramsKeeper.GetSubspace(subscriptiontypes.ModuleName)

	dualstakingparamsSubspace, _ := paramsKeeper.GetSubspace(dualstakingtypes.ModuleName)

	rewardsparamsSubspace, _ := paramsKeeper.GetSubspace(rewardstypes.ModuleName)

	conflictparamsSubspace := paramstypes.NewSubspace(cdc,
		conflicttypes.Amino,
		conflictStoreKey,
		conflictMemStoreKey,
		"ConflictParams",
	)

	downtimeParamsSubspace, _ := paramsKeeper.GetSubspace(downtimemoduletypes.ModuleName)

	ks := Keepers{}
	ks.TimerStoreKeeper = timerstorekeeper.NewKeeper(cdc)
	ks.AccountKeeper = mockAccountKeeper{}
	ks.BankKeeper = mockBankKeeper{}
	init_balance()
	ks.StakingKeeper = *stakingkeeper.NewKeeper(cdc, stakingStoreKey, ks.AccountKeeper, ks.BankKeeper, authtypes.NewModuleAddress(govtypes.ModuleName).String())
	ks.Distribution = distributionkeeper.NewKeeper(cdc, distributionStoreKey, ks.AccountKeeper, ks.BankKeeper, ks.StakingKeeper, authtypes.FeeCollectorName, authtypes.NewModuleAddress(govtypes.ModuleName).String())
	ks.Spec = *speckeeper.NewKeeper(cdc, specStoreKey, specMemStoreKey, specparamsSubspace, ks.StakingKeeper)
	ks.Epochstorage = *epochstoragekeeper.NewKeeper(cdc, epochStoreKey, epochMemStoreKey, epochparamsSubspace, &ks.BankKeeper, &ks.AccountKeeper, ks.Spec, ks.StakingKeeper)
	ks.FixationStoreKeeper = fixationkeeper.NewKeeper(cdc, ks.TimerStoreKeeper, ks.Epochstorage.BlocksToSaveRaw)
	ks.Dualstaking = *dualstakingkeeper.NewKeeper(cdc, dualstakingStoreKey, dualstakingMemStoreKey, dualstakingparamsSubspace, &ks.BankKeeper, &ks.StakingKeeper, &ks.AccountKeeper, ks.Epochstorage, ks.Spec, ks.FixationStoreKeeper)
	// register the staking hooks
	ks.StakingKeeper.SetHooks(stakingtypes.NewMultiStakingHooks(ks.Dualstaking.Hooks()))
	ks.SlashingKeeper = slashingkeeper.NewKeeper(cdc, legacyCdc, slashingStoreKey, ks.StakingKeeper, authtypes.NewModuleAddress(govtypes.ModuleName).String())
	ks.Plans = *planskeeper.NewKeeper(cdc, plansStoreKey, plansMemStoreKey, plansparamsSubspace, ks.Epochstorage, ks.Spec, ks.FixationStoreKeeper, ks.StakingKeeper)
	ks.Projects = *projectskeeper.NewKeeper(cdc, projectsStoreKey, projectsMemStoreKey, projectsparamsSubspace, ks.Epochstorage, ks.FixationStoreKeeper)
	ks.Protocol = *protocolkeeper.NewKeeper(cdc, protocolStoreKey, protocolMemStoreKey, protocolparamsSubspace, authtypes.NewModuleAddress(govtypes.ModuleName).String())
	ks.Downtime = downtimekeeper.NewKeeper(cdc, downtimeKey, downtimeParamsSubspace, ks.Epochstorage)
	ks.Rewards = *rewardskeeper.NewKeeper(cdc, rewardsStoreKey, rewardsMemStoreKey, rewardsparamsSubspace, ks.BankKeeper, ks.AccountKeeper, ks.Spec, ks.Epochstorage, ks.Downtime, ks.StakingKeeper, ks.Dualstaking, ks.Distribution, authtypes.FeeCollectorName, ks.TimerStoreKeeper, authtypes.NewModuleAddress(govtypes.ModuleName).String())
	ks.Subscription = *subscriptionkeeper.NewKeeper(cdc, subscriptionStoreKey, subscriptionMemStoreKey, subscriptionparamsSubspace, &ks.BankKeeper, &ks.AccountKeeper, &ks.Epochstorage, ks.Projects, ks.Plans, ks.Dualstaking, ks.Rewards, ks.Spec, ks.FixationStoreKeeper, ks.TimerStoreKeeper, ks.StakingKeeper)
	ks.Pairing = *pairingkeeper.NewKeeper(cdc, pairingStoreKey, pairingMemStoreKey, pairingparamsSubspace, &ks.BankKeeper, &ks.AccountKeeper, ks.Spec, &ks.Epochstorage, ks.Projects, ks.Subscription, ks.Plans, ks.Downtime, ks.Dualstaking, &ks.StakingKeeper, ks.FixationStoreKeeper, ks.TimerStoreKeeper)
	ks.ParamsKeeper = paramsKeeper
	ks.Conflict = *conflictkeeper.NewKeeper(cdc, conflictStoreKey, conflictMemStoreKey, conflictparamsSubspace, &ks.BankKeeper, &ks.AccountKeeper, ks.Pairing, ks.Epochstorage, ks.Spec, ks.StakingKeeper)
	ks.BlockStore = MockBlockStore{height: 0, blockHistory: make(map[int64]*tenderminttypes.Block)}

	ctx := sdk.NewContext(stateStore, tmproto.Header{}, false, log.NewNopLogger())

	// Initialize params
	stakingparams := stakingtypes.DefaultParams()
	stakingparams.BondDenom = commonconsts.TestTokenDenom
	ks.StakingKeeper.SetParams(ctx, stakingparams)
	ks.Distribution.SetParams(ctx, distributiontypes.DefaultParams())
	ks.SlashingKeeper.SetParams(ctx, slashingtypes.DefaultParams())
	ks.Pairing.SetParams(ctx, pairingtypes.DefaultParams())
	ks.Spec.SetParams(ctx, spectypes.DefaultParams())
	ks.Subscription.SetParams(ctx, subscriptiontypes.DefaultParams())
	ks.Epochstorage.SetParams(ctx, epochstoragetypes.DefaultParams())
	ks.Dualstaking.SetParams(ctx, dualstakingtypes.DefaultParams())
	ks.Conflict.SetParams(ctx, conflicttypes.DefaultParams())
	ks.Projects.SetParams(ctx, projectstypes.DefaultParams())
	protocolParams := protocoltypes.DefaultParams()
	protocolParams.Version = protocoltypes.Version{ProviderTarget: "0.0.1", ProviderMin: "0.0.0", ConsumerTarget: "0.0.1", ConsumerMin: "0.0.0"}
	ks.Protocol.SetParams(ctx, protocolParams)
	ks.Plans.SetParams(ctx, planstypes.DefaultParams())
	ks.Downtime.SetParams(ctx, downtimev1.DefaultParams())
	ks.Rewards.SetParams(ctx, rewardstypes.DefaultParams())
	dualStakingParams := dualstakingtypes.DefaultParams()
	dualStakingParams.MinSelfDelegation.Amount = math.NewInt(100)
	ks.Dualstaking.SetParams(ctx, dualStakingParams)

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
	ss.StakingServer = stakingkeeper.NewMsgServerImpl(&ks.StakingKeeper)
	ss.SlashingServer = slashingkeeper.NewMsgServerImpl(ks.SlashingKeeper)
	ss.RewardsServer = rewardskeeper.NewMsgServerImpl(ks.Rewards)
	ss.DistributionServer = distributionkeeper.NewMsgServerImpl(ks.Distribution)

	core.SetEnvironment(&core.Environment{BlockStore: &ks.BlockStore})

	ks.Epochstorage.SetEpochDetails(ctx, *epochstoragetypes.DefaultGenesis().EpochDetails)

	// pools
	allocationPoolBalance := uint64(30000000000000)

	err := ks.BankKeeper.AddToBalance(GetModuleAddress(string(rewardstypes.ValidatorsRewardsAllocationPoolName)),
		sdk.NewCoins(sdk.NewCoin(stakingparams.BondDenom, sdk.NewIntFromUint64(allocationPoolBalance))))
	require.NoError(t, err)

	err = ks.BankKeeper.AddToBalance(
		GetModuleAddress(string(rewardstypes.ProvidersRewardsAllocationPool)),
		sdk.NewCoins(sdk.NewCoin(stakingparams.BondDenom, sdk.NewIntFromUint64(allocationPoolBalance))))
	require.NoError(t, err)

	if !fixedTime {
		ctx = ctx.WithBlockTime(time.Now())
	} else {
		ctx = ctx.WithBlockTime(fixedDate)
	}

	ks.Dualstaking.InitDelegations(ctx, *fixationtypes.DefaultGenesis())
	ks.Dualstaking.InitDelegators(ctx, *fixationtypes.DefaultGenesis())
	ks.Plans.InitPlans(ctx, *fixationtypes.DefaultGenesis())
	ks.Subscription.InitSubscriptions(ctx, *fixationtypes.DefaultGenesis())
	ks.Subscription.InitSubscriptionsTimers(ctx, *timerstoretypes.DefaultGenesis())
	ks.Subscription.InitCuTrackers(ctx, *fixationtypes.DefaultGenesis())
	ks.Subscription.InitCuTrackerTimers(ctx, *timerstoretypes.DefaultGenesis())
	ks.Projects.InitDevelopers(ctx, *fixationtypes.DefaultGenesis())
	ks.Projects.InitProjects(ctx, *fixationtypes.DefaultGenesis())
	ks.Rewards.InitRewardsRefillTS(ctx, *timerstoretypes.DefaultGenesis())
	ks.Rewards.RefillRewardsPools(ctx, nil, nil)
	ks.Distribution.InitGenesis(ctx, *distributiontypes.DefaultGenesisState())

	NewBlock(ctx, &ks)

	return &ss, &ks, sdk.WrapSDKContext(ctx)
}

func SimulateParamChange(ctx sdk.Context, paramKeeper paramskeeper.Keeper, subspace, key, value string) (err error) {
	proposal := &paramproposal.ParameterChangeProposal{Changes: []paramproposal.ParamChange{{Subspace: subspace, Key: key, Value: value}}}
	err = spec.HandleParameterChangeProposal(ctx, paramKeeper, proposal)
	return
}

func SimulatePlansAddProposal(ctx sdk.Context, plansKeeper planskeeper.Keeper, plansToPropose []planstypes.Plan, modify bool) error {
	proposal := planstypes.NewPlansAddProposal("mockProposal", "mockProposal plans add for testing", plansToPropose)
	proposal.Modify = modify
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

func SimulateUnstakeProposal(ctx sdk.Context, pairingKeeper pairingkeeper.Keeper, providersInfo []pairingtypes.ProviderUnstakeInfo, delegatorsSlashing []pairingtypes.DelegatorSlashing) error {
	proposal := pairingtypes.NewUnstakeProposal("mockProposal", "mockProposal unstake provider for testing", providersInfo, delegatorsSlashing)
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
	EndBlock(unwrapedCtx, ks)
	unwrapedCtx = UpdateBlockCtx(ctx, ks, customBlockTime...)
	NewBlock(unwrapedCtx, ks)
	return sdk.WrapSDKContext(unwrapedCtx)
}

func UpdateBlockCtx(ctx context.Context, ks *Keepers, customBlockTime ...time.Duration) sdk.Context {
	unwrapedCtx := sdk.UnwrapSDKContext(ctx)
	block := uint64(unwrapedCtx.BlockHeight() + 1)
	unwrapedCtx = unwrapedCtx.WithBlockHeight(int64(block))

	headerHash := make([]byte, BLOCK_HEADER_LEN)
	Randomizer.Read(headerHash)
	unwrapedCtx = unwrapedCtx.WithHeaderHash(headerHash)

	if len(customBlockTime) > 0 {
		ks.BlockStore.AdvanceBlock(customBlockTime[0])
	} else {
		defaultBlockTime := ks.Downtime.GetParams(unwrapedCtx).DowntimeDuration
		ks.BlockStore.AdvanceBlock(defaultBlockTime)
	}

	b := ks.BlockStore.LoadBlock(int64(block))
	unwrapedCtx = unwrapedCtx.WithBlockTime(b.Header.Time)
	return unwrapedCtx
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

	for uint64(unwrapedCtx.BlockHeight()) < block {
		ctx = AdvanceBlock(ctx, ks, customBlockTime...)
		unwrapedCtx = sdk.UnwrapSDKContext(ctx)
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
	return AdvanceToBlock(ctx, ks, nextEpochBlockNum, customBlockTime...)
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

		if beginBlocker, ok := fieldValue.Interface().(KeeperBeginBlockerWithRequest); ok {
			beginBlocker.BeginBlock(ctx, abci.RequestBeginBlock{})
		}
	}
}

// Make sure you save the new context
func EndBlock(ctx sdk.Context, ks *Keepers) {
	// get the value and type of the Keepers struct
	keepersType := reflect.TypeOf(*ks)
	keepersValue := reflect.ValueOf(*ks)

	ks.StakingKeeper.BlockValidatorUpdates(ctx) // staking module end blocker

	// iterate over all keepers and call BeginBlock (if it's implemented by the keeper)
	for i := 0; i < keepersType.NumField(); i++ {
		fieldValue := keepersValue.Field(i)

		if endBlocker, ok := fieldValue.Interface().(KeeperEndBlocker); ok {
			endBlocker.EndBlock(ctx)
		}
	}
}

func GetModuleAddress(moduleName string) sdk.AccAddress {
	moduleAddress := authtypes.NewModuleAddress(moduleName).String()
	baseAccount := authtypes.NewBaseAccount(nil, nil, 0, 0)
	baseAccount.Address = moduleAddress
	return authtypes.NewModuleAccount(baseAccount, moduleName, authtypes.Burner, authtypes.Staking).GetAddress()
}
