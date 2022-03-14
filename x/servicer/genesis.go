package servicer

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/keeper"
	"github.com/lavanet/lava/x/servicer/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set all the stakeMap
	for _, elem := range genState.StakeMapList {
		k.SetStakeMap(ctx, elem)
	}
	// Set all the specStakeStorage
	for _, elem := range genState.SpecStakeStorageList {
		k.SetSpecStakeStorage(ctx, elem)
	}
	// this line is used by starport scaffolding # genesis/module/init
	k.SetParams(ctx, genState.Params)
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.StakeMapList = k.GetAllStakeMap(ctx)
	genesis.SpecStakeStorageList = k.GetAllSpecStakeStorage(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
