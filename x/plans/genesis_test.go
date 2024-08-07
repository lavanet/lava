package plans_test

import (
	"testing"

	keepertest "github.com/lavanet/lava/v2/testutil/keeper"
	"github.com/lavanet/lava/v2/testutil/nullify"
	plans "github.com/lavanet/lava/v2/x/plans"
	"github.com/lavanet/lava/v2/x/plans/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.PlanKeeper(t)
	plans.InitGenesis(ctx, *k, genesisState)
	got := plans.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)
	// this line is used by starport scaffolding # genesis/test/assert
}
