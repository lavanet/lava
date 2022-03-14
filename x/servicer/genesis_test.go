package servicer_test

import (
	"testing"

	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/servicer"
	"github.com/lavanet/lava/x/servicer/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		StakeMapList: []types.StakeMap{
			{
				Index: "0",
			},
			{
				Index: "1",
			},
		},
		SpecStakeStorageList: []types.SpecStakeStorage{
			{
				Index: "0",
			},
			{
				Index: "1",
			},
		},
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.ServicerKeeper(t)
	servicer.InitGenesis(ctx, *k, genesisState)
	got := servicer.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.StakeMapList, got.StakeMapList)
	require.ElementsMatch(t, genesisState.SpecStakeStorageList, got.SpecStakeStorageList)
	// this line is used by starport scaffolding # genesis/test/assert
}
