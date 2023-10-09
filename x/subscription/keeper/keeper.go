package keeper

import (
	"fmt"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/lavanet/lava/common"
	commontypes "github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/x/subscription/types"
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

		subsFS common.FixationStore
		subsTS common.TimerStore
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
	fixationStoreKeeper types.FixationStoreKeeper,
	timerStoreKeeper types.TimerStoreKeeper,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	fs := *fixationStoreKeeper.NewFixationStore(storeKey, types.SubsFixationPrefix)

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

		subsFS: fs,
	}

	subsTimerCallback := func(ctx sdk.Context, subkey, _ []byte) {
		keeper.advanceMonth(ctx, subkey)
	}

	keeper.subsTS = *timerStoreKeeper.NewTimerStore(storeKey, types.SubsTimerPrefix).
		WithCallbackByBlockTime(subsTimerCallback)

	return keeper
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// ExportSubscriptions exports subscriptions data (for genesis)
func (k Keeper) ExportSubscriptions(ctx sdk.Context) commontypes.GenesisState {
	return k.subsFS.Export(ctx)
}

// ExportSubscriptionsTimers exports subscriptions timers data (for genesis)
func (k Keeper) ExportSubscriptionsTimers(ctx sdk.Context) []commontypes.RawMessage {
	return k.subsTS.Export(ctx)
}

// InitSubscriptions imports subscriptions data (from genesis)
func (k Keeper) InitSubscriptions(ctx sdk.Context, gs commontypes.GenesisState) {
	k.subsFS.Init(ctx, gs)
}

// InitSubscriptions imports subscriptions timers data (from genesis)
func (k Keeper) InitSubscriptionsTimers(ctx sdk.Context, data []commontypes.RawMessage) {
	k.subsTS.Init(ctx, data)
}
