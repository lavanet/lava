package keeper

import (
	"fmt"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/lavanet/lava/x/fixationstore"
	"github.com/lavanet/lava/x/timerstore"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	fixationtypes "github.com/lavanet/lava/x/fixationstore/types"
	"github.com/lavanet/lava/x/subscription/types"
	timertypes "github.com/lavanet/lava/x/timerstore/types"
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

		subsFS fixationstore.FixationStore
		subsTS timerstore.TimerStore

		cuTrackerFS fixationstore.FixationStore // key: "<sub> <provider>", value: month aggregated CU
		cuTrackerTS timerstore.TimerStore
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
	fixationStoreKeeper types.FixationStoreKeeper,
	timerStoreKeeper types.TimerStoreKeeper,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

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
	}

	subsTimerCallback := func(ctx sdk.Context, subkey, _ []byte) {
		keeper.advanceMonth(ctx, subkey)
	}

	cuTrackerCallback := func(ctx sdk.Context, cuTrackerTimerKey []byte, cuTrackerTimerData []byte) {
		keeper.RewardAndResetCuTracker(ctx, cuTrackerTimerKey, cuTrackerTimerData)
	}

	keeper.subsTS = *timerStoreKeeper.NewTimerStore(storeKey, types.SubsTimerPrefix).
		WithCallbackByBlockTime(subsTimerCallback)

	keeper.cuTrackerTS = *timerStoreKeeper.NewTimerStore(storeKey, types.CuTrackerTimerPrefix).
		WithCallbackByBlockHeight(cuTrackerCallback)

	keeper.cuTrackerFS = *fixationStoreKeeper.NewFixationStore(storeKey, types.CuTrackerFixationPrefix)

	keeper.subsFS = *fixationStoreKeeper.NewFixationStore(storeKey, types.SubsFixationPrefix)

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
func (k Keeper) ExportSubscriptionsTimers(ctx sdk.Context) timertypes.GenesisState {
	return k.subsTS.Export(ctx)
}

// InitSubscriptions imports subscriptions data (from genesis)
func (k Keeper) InitSubscriptions(ctx sdk.Context, gs fixationtypes.GenesisState) {
	k.subsFS.Init(ctx, gs)
}

// InitSubscriptions imports subscriptions timers data (from genesis)
func (k Keeper) InitSubscriptionsTimers(ctx sdk.Context, data timertypes.GenesisState) {
	k.subsTS.Init(ctx, data)
}

// InitCuTrackerTimers imports CuTrackers timers data (from genesis)
func (k Keeper) InitCuTrackerTimers(ctx sdk.Context, data timertypes.GenesisState) {
	k.cuTrackerTS.Init(ctx, data)
}

// ExportCuTrackerTimers exports CuTracker timers data (for genesis)
func (k Keeper) ExportCuTrackerTimers(ctx sdk.Context) timertypes.GenesisState {
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
