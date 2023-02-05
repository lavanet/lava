package packagemanager

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/packagemanager/keeper"
	"github.com/lavanet/lava/x/packagemanager/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set all the packageVersionsStorage
	for _, elem := range genState.PackageVersionsStorageList {
		k.SetPackageVersionsStorage(ctx, elem)
	}
	// this line is used by starport scaffolding # genesis/module/init
	k.SetParams(ctx, genState.Params)
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.PackageVersionsStorageList = k.GetAllPackageVersionsStorage(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
