package spec_test

import (
	"testing"

	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/spec"
	"github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		SpecList: []types.Spec{
			{
				Id: 0,
			},
			{
				Id: 1,
			},
		},
		SpecCount: 2,
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.SpecKeeper(t)
	spec.InitGenesis(ctx, *k, genesisState)
	got := spec.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.SpecList, got.SpecList)
	require.Equal(t, genesisState.SpecCount, got.SpecCount)
	// this line is used by starport scaffolding # genesis/test/assert
}
