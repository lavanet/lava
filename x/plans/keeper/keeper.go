package keeper

import (
	"fmt"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	fixationtypes "github.com/lavanet/lava/v2/x/fixationstore/types"
	"github.com/lavanet/lava/v2/x/plans/types"
)

type (
	Keeper struct {
		memKey     storetypes.StoreKey
		paramstore paramtypes.Subspace

		epochstorageKeeper types.EpochStorageKeeper
		specKeeper         types.SpecKeeper
		stakingKeeper      types.StakingKeeper

		plansFS fixationtypes.FixationStore
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,
	epochstorageKeeper types.EpochStorageKeeper,
	specKeeper types.SpecKeeper,
	fixationStoreKeeper types.FixationStoreKeeper,
	stakingKeeper types.StakingKeeper,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	fs := *fixationStoreKeeper.NewFixationStore(storeKey, types.PlanFixationStorePrefix)

	return &Keeper{
		memKey:             memKey,
		paramstore:         ps,
		epochstorageKeeper: epochstorageKeeper,
		plansFS:            fs,
		specKeeper:         specKeeper,
		stakingKeeper:      stakingKeeper,
	}
}

// Export all plans from the KVStore
func (k Keeper) ExportPlans(ctx sdk.Context) fixationtypes.GenesisState {
	return k.plansFS.Export(ctx)
}

// Init all plans in the KVStore
func (k Keeper) InitPlans(ctx sdk.Context, gs fixationtypes.GenesisState) {
	k.plansFS.Init(ctx, gs)
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) IsEpochStart(ctx sdk.Context) bool {
	return k.epochstorageKeeper.GetEpochStart(ctx) == uint64(ctx.BlockHeight())
}
