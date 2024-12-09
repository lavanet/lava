package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/store/types"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	collcompat "github.com/lavanet/lava/v4/utils/collcompat"
	"github.com/lavanet/lava/v4/x/dualstaking/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   storetypes.StoreKey
		memKey     storetypes.StoreKey
		paramstore paramtypes.Subspace

		bankKeeper         types.BankKeeper
		stakingKeeper      types.StakingKeeper
		accountKeeper      types.AccountKeeper
		epochstorageKeeper types.EpochstorageKeeper
		specKeeper         types.SpecKeeper

		schema      collections.Schema
		delegations *collections.IndexedMap[collections.Pair[string, string], types.Delegation, types.DelegationIndexes] // key: [delegator, provider]
		rewards     collections.Map[collections.Pair[string, string], types.DelegatorReward]                             // key: [delegator, provider]
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,
	bankKeeper types.BankKeeper,
	stakingKeeper types.StakingKeeper,
	accountKeeper types.AccountKeeper,
	epochstorageKeeper types.EpochstorageKeeper,
	specKeeper types.SpecKeeper,
	fixationStoreKeeper types.FixationStoreKeeper,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	sb := collections.NewSchemaBuilder(collcompat.NewKVStoreService(storeKey))

	keeper := &Keeper{
		cdc:        cdc,
		storeKey:   storeKey,
		memKey:     memKey,
		paramstore: ps,

		bankKeeper:         bankKeeper,
		stakingKeeper:      stakingKeeper,
		accountKeeper:      accountKeeper,
		epochstorageKeeper: epochstorageKeeper,
		specKeeper:         specKeeper,

		delegations: collections.NewIndexedMap(sb, types.DelegationsPrefix, "delegations",
			collections.PairKeyCodec(collections.StringKey, collections.StringKey),
			collcompat.ProtoValue[types.Delegation](cdc),
			types.NewDelegationIndexes(sb)),

		rewards: collections.NewMap(sb, types.RewardPrefix, "rewards",
			collections.PairKeyCodec(collections.StringKey, collections.StringKey),
			collcompat.ProtoValue[types.DelegatorReward](cdc)),
	}

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

func (k Keeper) ChangeDelegationTimestampForTesting(ctx sdk.Context, provider, delegator string, timestamp int64) error {
	d, found := k.GetDelegation(ctx, provider, delegator)
	if !found {
		return fmt.Errorf("cannot change delegation timestamp: delegation not found. provider: %s, delegator: %s", provider, delegator)
	}
	d.Timestamp = timestamp
	return k.SetDelegation(ctx, d)
}

func (k Keeper) BeginBlock(ctx sdk.Context) {
	k.HandleSlashedValidators(ctx)
}
