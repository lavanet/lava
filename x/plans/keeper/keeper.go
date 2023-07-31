package keeper

import (
	"fmt"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/lavanet/lava/common"
	commontypes "github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/x/plans/types"
)

type (
	Keeper struct {
		memKey     storetypes.StoreKey
		paramstore paramtypes.Subspace

		epochstorageKeeper types.EpochStorageKeeper
		specKeeper         types.SpecKeeper

		plansFS common.FixationStore
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,
	epochstorageKeeper types.EpochStorageKeeper,
	specKeeper types.SpecKeeper,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	fs := *common.NewFixationStore(storeKey, cdc, types.PlanFixationStorePrefix)

	return &Keeper{
		memKey:             memKey,
		paramstore:         ps,
		epochstorageKeeper: epochstorageKeeper,
		plansFS:            fs,
		specKeeper:         specKeeper,
	}
}

func (k Keeper) BeginBlock(ctx sdk.Context) {
	k.plansFS.AdvanceBlock(ctx)
}

// Export all plans from the KVStore
func (k Keeper) ExportPlans(ctx sdk.Context) []commontypes.RawMessage {
	return k.plansFS.Export(ctx)
}

// Init all plans in the KVStore
func (k Keeper) InitPlans(ctx sdk.Context, data []commontypes.RawMessage) {
	k.plansFS.Init(ctx, data)
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k Keeper) IsEpochStart(ctx sdk.Context) bool {
	return k.epochstorageKeeper.GetEpochStart(ctx) == uint64(ctx.BlockHeight())
}
