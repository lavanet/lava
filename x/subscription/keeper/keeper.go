package keeper

import (
	"fmt"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/lavanet/lava/common"
	"github.com/lavanet/lava/x/subscription/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   sdk.StoreKey
		memKey     sdk.StoreKey
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
	memKey sdk.StoreKey,
	ps paramtypes.Subspace,

	bankKeeper types.BankKeeper,
	accountKeeper types.AccountKeeper,
	epochstorageKeeper types.EpochstorageKeeper,
	projectsKeeper types.ProjectsKeeper,
	plansKeeper types.PlansKeeper,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	fs := *common.NewFixationStore(storeKey, cdc, types.SubsFixationPrefix)

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

	subsTimerCallback := func(ctx sdk.Context, subkey []byte, _ []byte) {
		keeper.advanceMonth(ctx, subkey)
	}

	keeper.subsTS = *common.NewTimerStore(storeKey, cdc, types.SubsTimerPrefix).
		WithCallbackByBlockTime(subsTimerCallback)

	return keeper
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) BeginBlock(ctx sdk.Context) {
	k.subsFS.AdvanceBlock(ctx)
	k.subsTS.Tick(ctx)
}
