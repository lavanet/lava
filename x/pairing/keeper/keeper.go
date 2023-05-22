package keeper

import (
	"fmt"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/lavanet/lava/common"
	"github.com/lavanet/lava/x/pairing/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   sdk.StoreKey
		memKey     sdk.StoreKey
		paramstore paramtypes.Subspace

		bankKeeper         types.BankKeeper
		accountKeeper      types.AccountKeeper
		specKeeper         types.SpecKeeper
		epochStorageKeeper types.EpochstorageKeeper
		projectsKeeper     types.ProjectsKeeper
		subscriptionKeeper types.SubscriptionKeeper

		badgeTimerStore common.TimerStore
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey sdk.StoreKey,
	ps paramtypes.Subspace,

	bankKeeper types.BankKeeper,
	accountKeeper types.AccountKeeper,
	specKeeper types.SpecKeeper,
	epochStorageKeeper types.EpochstorageKeeper,
	projectsKeeper types.ProjectsKeeper,
	subscriptionKeeper types.SubscriptionKeeper,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	keeper := &Keeper{
		cdc:                cdc,
		storeKey:           storeKey,
		memKey:             memKey,
		paramstore:         ps,
		bankKeeper:         bankKeeper,
		accountKeeper:      accountKeeper,
		specKeeper:         specKeeper,
		epochStorageKeeper: epochStorageKeeper,
		projectsKeeper:     projectsKeeper,
		subscriptionKeeper: subscriptionKeeper,
	}

	badgeTimerCallback := func(ctx sdk.Context, timerKey []byte, badgeUsedCuKey []byte) {
		keeper.RemoveBadgeUsedCu(ctx, badgeUsedCuKey)
	}
	badgeTimerStore := common.NewTimerStore(storeKey, cdc, types.BadgeTimerStorePrefix).WithCallbackByBlockHeight(badgeTimerCallback)
	keeper.badgeTimerStore = *badgeTimerStore

	epochStorageKeeper.AddFixationRegistry(string(types.KeyServicersToPairCount), func(ctx sdk.Context) any { return keeper.ServicersToPairCountRaw(ctx) })
	epochStorageKeeper.AddFixationRegistry(string(types.KeyStakeToMaxCUList), func(ctx sdk.Context) any { return keeper.StakeToMaxCUListRaw(ctx) })

	return keeper
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) BeginBlock(ctx sdk.Context) {
	k.IncrementTimer(ctx)
	if k.epochStorageKeeper.IsEpochStart(ctx) {
		// run functions that are supposed to run in epoch start
		k.EpochStart(ctx, types.EPOCHS_NUM_TO_CHECK_CU_FOR_UNRESPONSIVE_PROVIDER, types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS)
	}
}

func (k Keeper) IncrementTimer(ctx sdk.Context) {
	k.badgeTimerStore.Tick(ctx)
}
