package rewards

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/rewards/keeper"
	"github.com/lavanet/lava/x/rewards/types"
	timerstoretypes "github.com/lavanet/lava/x/timerstore/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// this line is used by starport scaffolding # genesis/module/init
	k.SetParams(ctx, genState.Params)
	k.InitRewardsRefillTS(ctx, *timerstoretypes.DefaultGenesis())
	k.RefillRewardsPools(ctx, nil, nil)
	k.SetAllBasePay(ctx, genState.BasePays)
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)
	genesis.RefillRewardsTS = k.ExportRewardsRefillTS(ctx)
	genesis.BasePays = k.GetAllBasePay(ctx)

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
