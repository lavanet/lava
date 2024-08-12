package keeper

import (
	"fmt"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	fixationtypes "github.com/lavanet/lava/v2/x/fixationstore/types"
	"github.com/lavanet/lava/v2/x/subscription/types"
	timerstoretypes "github.com/lavanet/lava/v2/x/timerstore/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   storetypes.StoreKey
		memKey     storetypes.StoreKey
		paramstore paramtypes.Subspace

		bankKeeper         types.BankKeeper
		accountKeeper      types.AccountKeeper
		epochstorageKeeper types.EpochstorageKeeper
		projectsKeeper     types.ProjectsKeeper
		plansKeeper        types.PlansKeeper
		dualstakingKeeper  types.DualStakingKeeper
		rewardsKeeper      types.RewardsKeeper
		stakingKeeper      types.StakingKeeper
		specKeeper         types.SpecKeeper

		subsFS fixationtypes.FixationStore
		subsTS timerstoretypes.TimerStore

		cuTrackerFS fixationtypes.FixationStore // key: "<sub> <provider>", value: month aggregated CU
		cuTrackerTS timerstoretypes.TimerStore  // key: sub, value: credit for reward
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,

	bankKeeper types.BankKeeper,
	accountKeeper types.AccountKeeper,
	epochstorageKeeper types.EpochstorageKeeper,
	projectsKeeper types.ProjectsKeeper,
	plansKeeper types.PlansKeeper,
	dualstakingKeeper types.DualStakingKeeper,
	rewardsKeeper types.RewardsKeeper,
	specKeeper types.SpecKeeper,
	fixationStoreKeeper types.FixationStoreKeeper,
	timerStoreKeeper types.TimerStoreKeeper,
	stakingKeeper types.StakingKeeper,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	fs := *fixationStoreKeeper.NewFixationStore(storeKey, types.SubsFixationPrefix)
	cuTracker := *fixationStoreKeeper.NewFixationStore(storeKey, types.CuTrackerFixationPrefix)

	keeper := &Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		memKey:     memKey,
		paramstore: ps,

		bankKeeper:         bankKeeper,
		accountKeeper:      accountKeeper,
		epochstorageKeeper: epochstorageKeeper,
		projectsKeeper:     projectsKeeper,
		plansKeeper:        plansKeeper,
		dualstakingKeeper:  dualstakingKeeper,
		rewardsKeeper:      rewardsKeeper,
		stakingKeeper:      stakingKeeper,
		specKeeper:         specKeeper,

		subsFS:      fs,
		cuTrackerFS: cuTracker,
	}

	subsTimerCallback := func(ctx sdk.Context, subkey, _ []byte) {
		keeper.advanceMonth(ctx, subkey)
	}

	keeper.subsTS = *timerStoreKeeper.NewTimerStoreBeginBlock(storeKey, types.SubsTimerPrefix).
		WithCallbackByBlockTime(subsTimerCallback)

	cuTrackerCallback := func(ctx sdk.Context, cuTrackerTimerKey []byte, cuTrackerTimerData []byte) {
		keeper.RewardAndResetCuTracker(ctx, cuTrackerTimerKey, cuTrackerTimerData)
	}

	keeper.cuTrackerTS = *timerStoreKeeper.NewTimerStoreEndBlock(storeKey, types.CuTrackerTimerPrefix).
		WithCallbackByBlockHeight(cuTrackerCallback)

	return keeper
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// ExportSubscriptions exports subscriptions data (for genesis)
func (k Keeper) ExportSubscriptions(ctx sdk.Context) fixationtypes.GenesisState {
	return k.subsFS.Export(ctx)
}

// ExportSubscriptionsTimers exports subscriptions timers data (for genesis)
func (k Keeper) ExportSubscriptionsTimers(ctx sdk.Context) timerstoretypes.GenesisState {
	return k.subsTS.Export(ctx)
}

// InitSubscriptions imports subscriptions data (from genesis)
func (k Keeper) InitSubscriptions(ctx sdk.Context, gs fixationtypes.GenesisState) {
	k.subsFS.Init(ctx, gs)
}

// InitSubscriptions imports subscriptions timers data (from genesis)
func (k Keeper) InitSubscriptionsTimers(ctx sdk.Context, data timerstoretypes.GenesisState) {
	k.subsTS.Init(ctx, data)
}

// InitCuTrackerTimers imports CuTrackers timers data (from genesis)
func (k Keeper) InitCuTrackerTimers(ctx sdk.Context, data timerstoretypes.GenesisState) {
	k.cuTrackerTS.Init(ctx, data)
}

// ExportCuTrackerTimers exports CuTracker timers data (for genesis)
func (k Keeper) ExportCuTrackerTimers(ctx sdk.Context) timerstoretypes.GenesisState {
	return k.cuTrackerTS.Export(ctx)
}

// InitCuTrackers imports CuTracker data (from genesis)
func (k Keeper) InitCuTrackers(ctx sdk.Context, gs fixationtypes.GenesisState) {
	k.cuTrackerFS.Init(ctx, gs)
}

// ExportCuTrackers exports CuTrackers data (for genesis)
func (k Keeper) ExportCuTrackers(ctx sdk.Context) fixationtypes.GenesisState {
	return k.cuTrackerFS.Export(ctx)
}
