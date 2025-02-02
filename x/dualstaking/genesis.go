package dualstaking

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/x/dualstaking/keeper"
	"github.com/lavanet/lava/v5/x/dualstaking/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	k.SetParams(ctx, genState.Params)

	for _, d := range genState.Delegations {
		err := k.SetDelegation(ctx, d)
		if err != nil {
			panic(err)
		}
	}

	// Set all the DelegatorReward
	for _, elem := range genState.DelegatorRewardList {
		k.SetDelegatorReward(ctx, elem)
	}
}

// ExportGenesis returns the module's exported genesis
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	var err error
	genesis.Delegations, err = k.GetAllDelegations(ctx)
	if err != nil {
		panic(err)
	}

	genesis.DelegatorRewardList = k.GetAllDelegatorReward(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
