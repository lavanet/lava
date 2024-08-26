package keeper

import (
	"fmt"
	"strconv"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	fixationtypes "github.com/lavanet/lava/v2/x/fixationstore/types"

	"github.com/lavanet/lava/v2/x/dualstaking/types"
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

		delegationFS fixationtypes.FixationStore // map proviers/chainID -> delegations
		delegatorFS  fixationtypes.FixationStore // map delegators -> providers
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
	}

	delegationFS := *fixationStoreKeeper.NewFixationStore(storeKey, types.DelegationPrefix)
	delegatorFS := *fixationStoreKeeper.NewFixationStore(storeKey, types.DelegatorPrefix)

	keeper.delegationFS = delegationFS
	keeper.delegatorFS = delegatorFS

	return keeper
}

// ExportDelegations exports dualstaking delegations data (for genesis)
func (k Keeper) ExportDelegations(ctx sdk.Context) fixationtypes.GenesisState {
	return k.delegationFS.Export(ctx)
}

// ExportDelegators exports dualstaking delegators data (for genesis)
func (k Keeper) ExportDelegators(ctx sdk.Context) fixationtypes.GenesisState {
	return k.delegatorFS.Export(ctx)
}

// InitDelegations imports dualstaking delegations data (from genesis)
func (k Keeper) InitDelegations(ctx sdk.Context, data fixationtypes.GenesisState) {
	k.delegationFS.Init(ctx, data)
}

// InitDelegators imports dualstaking delegators data (from genesis)
func (k Keeper) InitDelegators(ctx sdk.Context, data fixationtypes.GenesisState) {
	k.delegatorFS.Init(ctx, data)
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) ChangeDelegationTimestampForTesting(ctx sdk.Context, index string, block uint64, timestamp int64) error {
	var d types.Delegation
	entryBlock, _, _, found := k.delegationFS.FindEntryDetailed(ctx, index, block, &d)
	if !found {
		return fmt.Errorf("cannot change delegation timestamp: delegation not found. index: %s, block: %s", index, strconv.FormatUint(block, 10))
	}
	d.Timestamp = timestamp
	k.delegationFS.ModifyEntry(ctx, index, entryBlock, &d)
	return nil
}

func (k Keeper) BeginBlock(ctx sdk.Context, req abci.RequestBeginBlock) {
	k.HandleSlashedValidators(ctx)
}
