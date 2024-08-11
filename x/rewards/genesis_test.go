package rewards

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	testkeeper "github.com/lavanet/lava/v2/testutil/keeper"
	"github.com/lavanet/lava/v2/testutil/nullify"
	"github.com/lavanet/lava/v2/x/rewards/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		IprpcRewards: []types.IprpcReward{
			{
				Id:        0,
				SpecFunds: []types.Specfund{{Fund: []sdk.Coin{}}},
			},
			{
				Id:        1,
				SpecFunds: []types.Specfund{{Fund: []sdk.Coin{}}},
			},
		},
		IprpcRewardsCurrent: 2,
		// this line is used by starport scaffolding # genesis/test/state
	}

	_, keepers, goCtx := testkeeper.InitAllKeepers(t)
	ctx := sdk.UnwrapSDKContext(goCtx)
	k := keepers.Rewards
	InitGenesis(ctx, k, genesisState)
	got := ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.IprpcRewards, got.IprpcRewards)
	require.Equal(t, genesisState.IprpcRewardsCurrent, got.IprpcRewardsCurrent)
	// this line is used by starport scaffolding # genesis/test/assert
}
