package dualstaking_test

import (
	"testing"

	keepertest "github.com/lavanet/lava/v2/testutil/keeper"
	"github.com/lavanet/lava/v2/testutil/nullify"
	"github.com/lavanet/lava/v2/x/dualstaking"
	"github.com/lavanet/lava/v2/x/dualstaking/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.DualstakingKeeper(t)
	dualstaking.InitGenesis(ctx, *k, genesisState)
	got := dualstaking.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)
	require.ElementsMatch(t, genesisState.DelegatorRewardList, got.DelegatorRewardList)

	// this line is used by starport scaffolding # genesis/test/assert
}
