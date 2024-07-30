package keeper

import (
	"fmt"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	timerstoretypes "github.com/lavanet/lava/x/timerstore/types"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	fixationtypes "github.com/lavanet/lava/x/fixationstore/types"
	"github.com/lavanet/lava/x/pairing/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   storetypes.StoreKey
		memKey     storetypes.StoreKey
		paramstore paramtypes.Subspace

		bankKeeper         types.BankKeeper
		accountKeeper      types.AccountKeeper
		specKeeper         types.SpecKeeper
		epochStorageKeeper types.EpochstorageKeeper
		projectsKeeper     types.ProjectsKeeper
		subscriptionKeeper types.SubscriptionKeeper
		planKeeper         types.PlanKeeper
		badgeTimerStore    timerstoretypes.TimerStore
		providerQosFS      fixationtypes.FixationStore
		downtimeKeeper     types.DowntimeKeeper
		dualstakingKeeper  types.DualstakingKeeper
		stakingKeeper      types.StakingKeeper

		pairingQueryCache *map[string][]epochstoragetypes.StakeEntry
		pairingRelayCache *map[string][]epochstoragetypes.StakeEntry
	}
)

// sanity checks at start time
func init() {
	if types.EPOCHS_NUM_TO_CHECK_CU_FOR_UNRESPONSIVE_PROVIDER == 0 {
		panic("types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS == 0")
	}
	if types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS == 0 {
		panic("types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS == 0")
	}
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,

	bankKeeper types.BankKeeper,
	accountKeeper types.AccountKeeper,
	specKeeper types.SpecKeeper,
	epochStorageKeeper types.EpochstorageKeeper,
	projectsKeeper types.ProjectsKeeper,
	subscriptionKeeper types.SubscriptionKeeper,
	planKeeper types.PlanKeeper,
	downtimeKeeper types.DowntimeKeeper,
	dualstakingKeeper types.DualstakingKeeper,
	stakingKeeper types.StakingKeeper,
	fixationStoreKeeper types.FixationStoreKeeper,
	timerStoreKeeper types.TimerStoreKeeper,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	emptypairingRelayCache := map[string][]epochstoragetypes.StakeEntry{}
	emptypairingQueryCache := map[string][]epochstoragetypes.StakeEntry{}

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
		planKeeper:         planKeeper,
		downtimeKeeper:     downtimeKeeper,
		dualstakingKeeper:  dualstakingKeeper,
		stakingKeeper:      stakingKeeper,
		pairingQueryCache:  &emptypairingQueryCache,
		pairingRelayCache:  &emptypairingRelayCache,
	}

	// note that the timer and badgeUsedCu keys are the same (so we can use only the second arg)
	badgeTimerCallback := func(ctx sdk.Context, badgeKey, _ []byte) {
		keeper.RemoveBadgeUsedCu(ctx, badgeKey)
	}
	badgeTimerStore := timerStoreKeeper.NewTimerStoreBeginBlock(storeKey, types.BadgeTimerStorePrefix).
		WithCallbackByBlockHeight(badgeTimerCallback)
	keeper.badgeTimerStore = *badgeTimerStore

	keeper.providerQosFS = *fixationStoreKeeper.NewFixationStore(storeKey, types.ProviderQosStorePrefix)

	return keeper
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) BeginBlock(ctx sdk.Context) {

	// reset pairing relay cache every block
	*k.pairingRelayCache = map[string][]epochstoragetypes.StakeEntry{}
	if k.epochStorageKeeper.IsEpochStart(ctx) {
		// reset pairing query cache every epoch
		*k.pairingQueryCache = map[string][]epochstoragetypes.StakeEntry{}
		// remove old session payments
		k.RemoveOldEpochPayments(ctx)
		// unstake/jail unresponsive providers
		k.PunishUnresponsiveProviders(ctx,
			types.EPOCHS_NUM_TO_CHECK_CU_FOR_UNRESPONSIVE_PROVIDER,
			types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS)
	}
}

func (k Keeper) InitProviderQoS(ctx sdk.Context, gs fixationtypes.GenesisState) {
	k.providerQosFS.Init(ctx, gs)
}

func (k Keeper) ExportProviderQoS(ctx sdk.Context) fixationtypes.GenesisState {
	return k.providerQosFS.Export(ctx)
}
