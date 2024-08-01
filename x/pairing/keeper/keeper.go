package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	collcompat "github.com/lavanet/lava/utils/collcompat"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	timerstoretypes "github.com/lavanet/lava/x/timerstore/types"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
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
		reputationsFS      fixationtypes.FixationStore
		downtimeKeeper     types.DowntimeKeeper
		dualstakingKeeper  types.DualstakingKeeper
		stakingKeeper      types.StakingKeeper

		schema            collections.Schema
		reputations       *collections.IndexedMap[collections.Triple[string, string, string], types.Reputation, types.ReputationRefIndexes] // save qos info per provider, chain and cluster
		reputationRefKeys collections.KeySet[collections.Pair[string, string]]
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

	sb := collections.NewSchemaBuilder(collcompat.NewKVStoreService(storeKey))

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

		reputations: collections.NewIndexedMap(sb, types.ReputationPrefix, "reputations",
			collections.TripleKeyCodec(collections.StringKey, collections.StringKey, collections.StringKey),
			collcompat.ProtoValue[types.Reputation](cdc),
			types.NewReputationRefIndexes(sb),
		),

		reputationRefKeys: collections.NewKeySet(sb, types.ReputationRefKeysPrefix, "reputations_ref_keys",
			collections.PairKeyCodec(collections.StringKey, collections.StringKey),
		),
	}

	// note that the timer and badgeUsedCu keys are the same (so we can use only the second arg)
	badgeTimerCallback := func(ctx sdk.Context, badgeKey, _ []byte) {
		keeper.RemoveBadgeUsedCu(ctx, badgeKey)
	}
	badgeTimerStore := timerStoreKeeper.NewTimerStoreBeginBlock(storeKey, types.BadgeTimerStorePrefix).
		WithCallbackByBlockHeight(badgeTimerCallback)
	keeper.badgeTimerStore = *badgeTimerStore

	keeper.reputationsFS = *fixationStoreKeeper.NewFixationStore(storeKey, types.ProviderQosStorePrefix)

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	keeper.schema = schema

	return keeper
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) BeginBlock(ctx sdk.Context) {
	if k.epochStorageKeeper.IsEpochStart(ctx) {
		// remove old session payments
		k.RemoveOldEpochPayments(ctx)
		// unstake any unstaking providers
		k.CheckUnstakingForCommit(ctx)
		// unstake/jail unresponsive providers
		k.PunishUnresponsiveProviders(ctx,
			types.EPOCHS_NUM_TO_CHECK_CU_FOR_UNRESPONSIVE_PROVIDER,
			types.EPOCHS_NUM_TO_CHECK_FOR_COMPLAINERS)
	}
}

func (k Keeper) InitReputations(ctx sdk.Context, gs fixationtypes.GenesisState) {
	k.reputationsFS.Init(ctx, gs)
}

func (k Keeper) ExportReputations(ctx sdk.Context) fixationtypes.GenesisState {
	return k.reputationsFS.Export(ctx)
}
