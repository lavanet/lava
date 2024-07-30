package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/lavanet/lava/utils"
	collcompat "github.com/lavanet/lava/utils/collcompat"
	"github.com/lavanet/lava/x/epochstorage/types"
)

type (
	Keeper struct {
		cdc        codec.BinaryCodec
		storeKey   storetypes.StoreKey
		memKey     storetypes.StoreKey
		paramstore paramtypes.Subspace

		bankKeeper    types.BankKeeper
		accountKeeper types.AccountKeeper
		specKeeper    types.SpecKeeper
		stakingKeeper types.StakingKeeper

		fixationRegistries map[string]func(sdk.Context) any

		schema              collections.Schema
		stakeEntries        *collections.IndexedMap[collections.Triple[uint64, string, collections.Pair[uint64, string]], types.StakeEntry, types.EpochChainIdProviderIndexes]
		stakeEntriesCurrent *collections.IndexedMap[collections.Pair[string, string], types.StakeEntry, types.ChainIdVaultIndexes]
		epochHashes         collections.Map[uint64, []byte]
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,

	bankKeeper types.BankKeeper,
	accountKeeper types.AccountKeeper,
	specKeeper types.SpecKeeper,
	stakingKeeper types.StakingKeeper,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	sb := collections.NewSchemaBuilder(collcompat.NewKVStoreService(storeKey))

	keeper := &Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		memKey:        memKey,
		paramstore:    ps,
		bankKeeper:    bankKeeper,
		accountKeeper: accountKeeper,
		specKeeper:    specKeeper,
		stakingKeeper: stakingKeeper,

		fixationRegistries: make(map[string]func(sdk.Context) any),

		stakeEntries: collections.NewIndexedMap(sb, types.StakeEntriesPrefix, "stake_entries",
			collections.TripleKeyCodec(collections.Uint64Key, collections.StringKey,
				collections.PairKeyCodec(collections.Uint64Key, collections.StringKey)),
			collcompat.ProtoValue[types.StakeEntry](cdc), types.NewEpochChainIdProviderIndexes(sb)),

		stakeEntriesCurrent: collections.NewIndexedMap(sb, types.StakeEntriesCurrentPrefix, "stake_entries_current",
			collections.PairKeyCodec(collections.StringKey, collections.StringKey),
			collcompat.ProtoValue[types.StakeEntry](cdc), types.NewChainIdVaultIndexes(sb)),

		epochHashes: collections.NewMap(sb, types.EpochHashesPrefix, "epoch_hashes", collections.Uint64Key, collections.BytesValue),
	}

	keeper.AddFixationRegistry(string(types.KeyEpochBlocks), func(ctx sdk.Context) any { return keeper.EpochBlocksRaw(ctx) })
	keeper.AddFixationRegistry(string(types.KeyEpochsToSave), func(ctx sdk.Context) any { return keeper.EpochsToSaveRaw(ctx) })

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

func (k *Keeper) AddFixationRegistry(fixationKey string, getParamFunction func(sdk.Context) any) {
	if _, ok := k.fixationRegistries[fixationKey]; ok {
		// panic:ok: duplicate fixation registry is severe (triggered at init time)
		panic("duplicate fixation registry: " + fixationKey)
	}
	k.fixationRegistries[fixationKey] = getParamFunction
}

func (k *Keeper) GetFixationRegistries() map[string]func(sdk.Context) any {
	return k.fixationRegistries
}

func (k Keeper) BeginBlock(ctx sdk.Context) {
	if k.IsEpochStart(ctx) {
		// run functions that are supposed to run in epoch start
		k.EpochStart(ctx)

		// Notify world we have a new session

		details := map[string]string{"height": fmt.Sprintf("%d", ctx.BlockHeight()), "description": "New Block Epoch Started"}
		utils.LogLavaEvent(ctx, k.Logger(ctx), "new_epoch", details, "")
	}
}
