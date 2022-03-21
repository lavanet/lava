package user

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/user/keeper"
	"github.com/lavanet/lava/x/user/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set all the userStake
	for _, elem := range genState.UserStakeList {
		k.SetUserStake(ctx, elem)
	}
	// Set all the specStakeStorage
	for _, elem := range genState.SpecStakeStorageList {
		k.SetSpecStakeStorage(ctx, elem)
	}
	k.SetBlockDeadlineForCallback(ctx, genState.BlockDeadlineForCallback)

	// Set all the unstakingUsersAllSpecs
	for _, elem := range genState.UnstakingUsersAllSpecsList {
		k.SetUnstakingUsersAllSpecs(ctx, elem)
	}

	// Set unstakingUsersAllSpecs count
	k.SetUnstakingUsersAllSpecsCount(ctx, genState.UnstakingUsersAllSpecsCount)
	// this line is used by starport scaffolding # genesis/module/init
	k.SetParams(ctx, genState.Params)
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.UserStakeList = k.GetAllUserStake(ctx)
	genesis.SpecStakeStorageList = k.GetAllSpecStakeStorage(ctx)
	// Get all blockDeadlineForCallback
	blockDeadlineForCallback, found := k.GetBlockDeadlineForCallback(ctx)
	if found {
		genesis.BlockDeadlineForCallback = blockDeadlineForCallback
	}
	genesis.UnstakingUsersAllSpecsList = k.GetAllUnstakingUsersAllSpecs(ctx)
	genesis.UnstakingUsersAllSpecsCount = k.GetUnstakingUsersAllSpecsCount(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
