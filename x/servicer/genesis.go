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
	// Set if defined
	k.SetBlockDeadlineForCallback(ctx, types.BlockDeadlineForCallback{Deadline: types.BlockNum{Num: 0}})

	// Set all the unstakingServicersAllSpecs
	for _, elem := range genState.UnstakingServicersAllSpecsList {
		k.SetUnstakingServicersAllSpecs(ctx, elem)
	}

	// Set unstakingServicersAllSpecs count
	k.SetUnstakingServicersAllSpecsCount(ctx, genState.UnstakingServicersAllSpecsCount)
	// Set if defined
	if genState.CurrentSessionStart != nil {
		k.SetCurrentSessionStart(ctx, *genState.CurrentSessionStart)
	}
	// Set if defined
	if genState.PreviousSessionBlocks != nil {
		k.SetPreviousSessionBlocks(ctx, *genState.PreviousSessionBlocks)
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
	// Get all blockDeadlineForCallback
	blockDeadlineForCallback, found := k.GetBlockDeadlineForCallback(ctx)
	if found {
		genesis.BlockDeadlineForCallback = blockDeadlineForCallback
	}
	genesis.UnstakingServicersAllSpecsList = k.GetAllUnstakingServicersAllSpecs(ctx)
	genesis.UnstakingServicersAllSpecsCount = k.GetUnstakingServicersAllSpecsCount(ctx)
	// Get all currentSessionStart
	currentSessionStart, found := k.GetCurrentSessionStart(ctx)
	if found {
		genesis.CurrentSessionStart = &currentSessionStart
	}
	// Get all previousSessionBlocks
	previousSessionBlocks, found := k.GetPreviousSessionBlocks(ctx)
	if found {
		genesis.PreviousSessionBlocks = &previousSessionBlocks
	}
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
