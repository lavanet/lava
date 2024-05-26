package rewards

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/rewards/types"
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
		PendingIprpcFunds: []types.PendingIprpcFund{
			{
				Index:       1,
				Creator:     "1",
				Spec:        "s1",
				Month:       1,
				Funds:       sdk.NewCoins(sdk.NewCoin("denom", math.OneInt())),
				Expiry:      1,
				CostCovered: sdk.NewCoin("denom", math.OneInt()),
			},
			{
				Index:       2,
				Creator:     "2",
				Spec:        "s2",
				Month:       2,
				Funds:       sdk.NewCoins(sdk.NewCoin("denom", math.OneInt().AddRaw(1))),
				Expiry:      2,
				CostCovered: sdk.NewCoin("denom", math.OneInt().AddRaw(1)),
			},
		},
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
	require.ElementsMatch(t, genesisState.PendingIprpcFunds, got.PendingIprpcFunds)
	// this line is used by starport scaffolding # genesis/test/assert
}
