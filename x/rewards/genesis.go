package rewards

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/rewards/keeper"
	"github.com/lavanet/lava/x/rewards/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	k.SetParams(ctx, genState.Params)
	k.SetAllBasePay(ctx, genState.BasePays)
	k.InitRewardsRefillTS(ctx, genState.RefillRewardsTS)

	// refill pools only if it's a new chain (RefillRewardsTS will have no timers)
	// InitGenesis can also run due to a fork, in which we don't want to refill the pools
	if len(genState.RefillRewardsTS.TimeEntries) == 0 {
		k.RefillRewardsPools(ctx, nil, nil)
	}

	for _, sub := range genState.IprpcSubscriptions {
		k.SetIprpcSubscription(ctx, sub)
	}
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)
	genesis.RefillRewardsTS = k.ExportRewardsRefillTS(ctx)
	genesis.BasePays = k.GetAllBasePay(ctx)
	genesis.IprpcSubscriptions = k.GetAllIprpcSubscription(ctx)
	genesis.MinIprpcCost = k.GetMinIprpcCost(ctx)
	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
