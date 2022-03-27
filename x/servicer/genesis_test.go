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
		BlockDeadlineForCallback: types.BlockDeadlineForCallback{
			Deadline: types.BlockNum{Num: 0},
		},
		UnstakingServicersAllSpecsList: []types.UnstakingServicersAllSpecs{
			{
				Id: 0,
			},
			{
				Id: 1,
			},
		},
		UnstakingServicersAllSpecsCount: 2,
		CurrentSessionStart: &types.CurrentSessionStart{
			Block: types.BlockNum{Num: 0},
		},
		PreviousSessionBlocks: &types.PreviousSessionBlocks{
			BlocksNum: 83,
		},
		SessionStorageForSpecList: []types.SessionStorageForSpec{
			{
				Index: "0",
			},
			{
				Index: "1",
			},
		},
		EarliestSessionStart: &types.EarliestSessionStart{
			Block: types.BlockNum{Num: 0},
		},
		UniquePaymentStorageUserServicerList: []types.UniquePaymentStorageUserServicer{
			{
				Index: "0",
			},
			{
				Index: "1",
			},
		},
		UserPaymentStorageList: []types.UserPaymentStorage{
			{
				Index: "0",
			},
			{
				Index: "1",
			},
		},
		SessionPaymentsList: []types.SessionPayments{
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
	require.Equal(t, genesisState.BlockDeadlineForCallback, got.BlockDeadlineForCallback)
	require.ElementsMatch(t, genesisState.UnstakingServicersAllSpecsList, got.UnstakingServicersAllSpecsList)
	require.Equal(t, genesisState.UnstakingServicersAllSpecsCount, got.UnstakingServicersAllSpecsCount)
	require.Equal(t, genesisState.CurrentSessionStart, got.CurrentSessionStart)
	require.Equal(t, genesisState.PreviousSessionBlocks, got.PreviousSessionBlocks)
	require.ElementsMatch(t, genesisState.SessionStorageForSpecList, got.SessionStorageForSpecList)
	require.Equal(t, genesisState.EarliestSessionStart, got.EarliestSessionStart)
	require.ElementsMatch(t, genesisState.UniquePaymentStorageUserServicerList, got.UniquePaymentStorageUserServicerList)
	require.ElementsMatch(t, genesisState.UserPaymentStorageList, got.UserPaymentStorageList)
	require.ElementsMatch(t, genesisState.SessionPaymentsList, got.SessionPaymentsList)
	// this line is used by starport scaffolding # genesis/test/assert
}
