package keeper

import (
	"fmt"
	"strconv"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	collcompat "github.com/lavanet/lava/utils/collcompat"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/lavanet/lava/utils"
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

		schema              collections.Schema
		providerClusterQos  *collections.IndexedMap[collections.Triple[string, string, string], types.ProviderClusterQos, types.ChainClusterQosIndexes] // save qos info per provider, chain and cluster
		chainClusterQosKeys collections.KeySet[collections.Pair[string, string]]

		pairingQueryCache *map[types.PairingCacheKey][]epochstoragetypes.StakeEntry
		pairingRelayCache *map[types.PairingCacheKey][]epochstoragetypes.StakeEntry
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

	emptypairingRelayCache := map[types.PairingCacheKey][]epochstoragetypes.StakeEntry{}
	emptypairingQueryCache := map[types.PairingCacheKey][]epochstoragetypes.StakeEntry{}

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
		pairingQueryCache:  &emptypairingQueryCache,
		pairingRelayCache:  &emptypairingRelayCache,

		providerClusterQos: collections.NewIndexedMap(sb, types.ProviderClusterQosPrefix, "provider_cluster_qos",
			collections.TripleKeyCodec(collections.StringKey, collections.StringKey, collections.StringKey),
			collcompat.ProtoValue[types.ProviderClusterQos](cdc),
			types.NewChainClusterQosIndexes(sb),
		),

		chainClusterQosKeys: collections.NewKeySet(sb, types.ChainClusterQosKeysPrefix, "chain_cluster_qos_keys",
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

	keeper.providerQosFS = *fixationStoreKeeper.NewFixationStore(storeKey, types.ProviderQosStorePrefix)

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

	// reset pairing relay cache every block
	*k.pairingRelayCache = map[types.PairingCacheKey][]epochstoragetypes.StakeEntry{}
	if k.epochStorageKeeper.IsEpochStart(ctx) {
		// reset pairing query cache every epoch
		*k.pairingQueryCache = map[types.PairingCacheKey][]epochstoragetypes.StakeEntry{}

		k.PrepareMockDataForBenchmark(ctx)
		gm := ctx.GasMeter()
		before := gm.GasConsumed()
		k.UpdateQosExcellenceScores(ctx)
		after := gm.GasConsumed() - before
		utils.LavaFormatInfo("oren gas report", utils.LogAttr("gas", strconv.FormatUint(after, 10)))

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

func (k Keeper) UpdateQosExcellenceScores(ctx sdk.Context) {
	iter, err := k.chainClusterQosKeys.Iterate(ctx, nil)
	if err != nil {
		panic(err)
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		chainCluster, err := iter.Key()
		if err != nil {
			panic(err)
		}
		multiIter, err := k.providerClusterQos.Indexes.Index.MatchExact(ctx, chainCluster)
		if err != nil {
			panic(err)
		}
		defer multiIter.Close()

		for ; multiIter.Valid(); multiIter.Next() {
			chainClusterProvider, err := multiIter.PrimaryKey()
			if err != nil {
				panic(err)
			}

			pcq, err := k.providerClusterQos.Get(ctx, chainClusterProvider)
			if err != nil {
				panic(err)
			}
			qps := k.ConvertQosScoreToPairingQosScore(chainClusterProvider.K3(), pcq)
			err = k.SetQosPairingScore(ctx, chainClusterProvider.K1(), chainClusterProvider.K2(), chainClusterProvider.K3(), qps)
			if err != nil {
				panic(err)
			}
		}
	}
}

func (k Keeper) PrepareMockDataForBenchmark(ctx sdk.Context) {
	for chainInt := 0; chainInt < 50; chainInt++ {
		for providerInt := 0; providerInt < 100; providerInt++ {
			chainID := strconv.Itoa(chainInt)
			provider := strconv.Itoa(providerInt)
			k.SetProviderClusterQos(ctx, chainID, "dummy", provider, types.ProviderClusterQos{
				Score:      types.QosScore{Score: types.Frac{Num: math.LegacyNewDec(int64(providerInt + 1)), Denom: math.LegacyNewDec(int64(providerInt + 2))}},
				EpochScore: types.QosScore{},
			})
		}
	}
}
