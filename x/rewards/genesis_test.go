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
		PendingIbcIprpcFunds: []types.PendingIbcIprpcFund{
			{
				Index:    1,
				Creator:  "1",
				Spec:     "s1",
				Duration: 1,
				Fund:     sdk.NewCoin("denom", math.OneInt()),
				Expiry:   1,
			},
			{
				Index:    2,
				Creator:  "2",
				Spec:     "s2",
				Duration: 2,
				Fund:     sdk.NewCoin("denom", math.OneInt().AddRaw(1)),
				Expiry:   2,
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
	require.ElementsMatch(t, genesisState.PendingIbcIprpcFunds, got.PendingIbcIprpcFunds)
	// this line is used by starport scaffolding # genesis/test/assert
}
