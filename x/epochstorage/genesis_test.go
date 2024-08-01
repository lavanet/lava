package epochstorage_test

import (
	"testing"

	keepertest "github.com/lavanet/lava/v2/testutil/keeper"
	"github.com/lavanet/lava/v2/testutil/nullify"
	"github.com/lavanet/lava/v2/x/epochstorage"
	"github.com/lavanet/lava/v2/x/epochstorage/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		StakeStorageList: []types.StakeStorage{
			{
				Index: "0",
			},
			{
				Index: "1",
			},
		},
		EpochDetails: &types.EpochDetails{
			StartBlock:    60,
			EarliestStart: 93,
		},
		FixatedParamsList: []types.FixatedParams{
			{
				Index: "0",
			},
			{
				Index: "1",
			},
		},
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.EpochstorageKeeper(t)
	epochstorage.InitGenesis(ctx, *k, genesisState)
	got := epochstorage.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.StakeStorageList, got.StakeStorageList)
	require.Equal(t, genesisState.EpochDetails, got.EpochDetails)
	require.ElementsMatch(t, genesisState.FixatedParamsList, got.FixatedParamsList)
	// this line is used by starport scaffolding # genesis/test/assert
}
