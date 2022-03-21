package user_test

import (
	"testing"

	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/user"
	"github.com/lavanet/lava/x/user/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		UserStakeList: []types.UserStake{
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
		BlockDeadlineForCallback: types.BlockDeadlineForCallback{
			Deadline: types.BlockNum{Num: 0},
		},
		UnstakingUsersAllSpecsList: []types.UnstakingUsersAllSpecs{
			{
				Id: 0,
			},
			{
				Id: 1,
			},
		},
		UnstakingUsersAllSpecsCount: 2,
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.UserKeeper(t)
	user.InitGenesis(ctx, *k, genesisState)
	got := user.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.UserStakeList, got.UserStakeList)
	require.ElementsMatch(t, genesisState.SpecStakeStorageList, got.SpecStakeStorageList)
	require.Equal(t, genesisState.BlockDeadlineForCallback, got.BlockDeadlineForCallback)
	require.ElementsMatch(t, genesisState.UnstakingUsersAllSpecsList, got.UnstakingUsersAllSpecsList)
	require.Equal(t, genesisState.UnstakingUsersAllSpecsCount, got.UnstakingUsersAllSpecsCount)
	// this line is used by starport scaffolding # genesis/test/assert
}
