package subscription

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/subscription/keeper"
	"github.com/lavanet/lava/v2/x/subscription/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	k.SetParams(ctx, genState.Params)
	k.InitSubscriptions(ctx, genState.SubsFS)
	k.InitSubscriptionsTimers(ctx, genState.SubsTS)
	k.InitCuTrackers(ctx, genState.CuTrackerFS)
	k.InitCuTrackerTimers(ctx, genState.CuTrackerTS)
	k.SetAllAdjustment(ctx, genState.Adjustments)
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)
	genesis.SubsFS = k.ExportSubscriptions(ctx)
	genesis.SubsTS = k.ExportSubscriptionsTimers(ctx)
	genesis.CuTrackerFS = k.ExportCuTrackers(ctx)
	genesis.CuTrackerTS = k.ExportCuTrackerTimers(ctx)
	genesis.Adjustments = k.GetAllAdjustment(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
