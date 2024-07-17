package conflict

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/conflict/keeper"
	"github.com/lavanet/lava/v2/x/conflict/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set all the conflictVote
	for _, elem := range genState.ConflictVoteList {
		k.SetConflictVote(ctx, elem)
	}
	// this line is used by starport scaffolding # genesis/module/init
	k.SetParams(ctx, genState.Params)

	k.PushFixations(ctx) // this needs to be at the last module genesis
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.ConflictVoteList = k.GetAllConflictVote(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
