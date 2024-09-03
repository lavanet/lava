package dualstaking

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v3/x/dualstaking/keeper"
	"github.com/lavanet/lava/v3/x/dualstaking/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	k.SetParams(ctx, genState.Params)

	k.InitDelegations(ctx, genState.DelegationsFS)
	k.InitDelegators(ctx, genState.DelegatorsFS)

	// Set all the DelegatorReward
	for _, elem := range genState.DelegatorRewardList {
		k.SetDelegatorReward(ctx, elem)
	}
}

// ExportGenesis returns the module's exported genesis
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	genesis.DelegationsFS = k.ExportDelegations(ctx)
	genesis.DelegatorsFS = k.ExportDelegators(ctx)
	genesis.DelegatorRewardList = k.GetAllDelegatorReward(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
