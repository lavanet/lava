package spec

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v3/x/spec/keeper"
	"github.com/lavanet/lava/v3/x/spec/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set all the spec
	for _, elem := range genState.SpecList {
		k.SetSpec(ctx, elem)
	}

	// this line is used by starport scaffolding # genesis/module/init
	k.SetParams(ctx, genState.Params)
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.SpecList = k.GetAllSpec(ctx)
	genesis.SpecCount = uint64(len(genesis.SpecList))

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
