package projects

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/projects/keeper"
	"github.com/lavanet/lava/v2/x/projects/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	k.SetParams(ctx, genState.Params)
	k.InitProjects(ctx, genState.ProjectsFS)
	k.InitDevelopers(ctx, genState.DeveloperFS)
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)
	genesis.ProjectsFS = k.ExportProjects(ctx)
	genesis.DeveloperFS = k.ExportDevelopers(ctx)

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
