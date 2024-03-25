package pairing_test

import (
	"testing"

	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/pairing"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		UniqueEpochSessions: []types.UniqueEpochSessionGenesis{
			{
				Epoch:              0,
				UniqueEpochSession: string(types.UniqueEpochSessionKey("0", "0", "0", 0)),
			},
			{
				Epoch:              1,
				UniqueEpochSession: string(types.UniqueEpochSessionKey("1", "1", "1", 1)),
			},
		},
		ProviderEpochCus: []types.ProviderEpochCuGenesis{
			{
				Epoch:           0,
				Provider:        "0",
				ProviderEpochCu: types.ProviderEpochCu{ServicedCu: 10, ComplainersCu: 100},
			},
			{
				Epoch:           1,
				Provider:        "1",
				ProviderEpochCu: types.ProviderEpochCu{ServicedCu: 20, ComplainersCu: 200},
			},
		},
		ProviderConsumerEpochCus: []types.ProviderConsumerEpochCuGenesis{
			{
				Epoch:                   0,
				Provider:                "0",
				Project:                 "0",
				ProviderConsumerEpochCu: types.ProviderConsumerEpochCu{Cu: 10},
			},
			{
				Epoch:                   1,
				Provider:                "1",
				Project:                 "1",
				ProviderConsumerEpochCu: types.ProviderConsumerEpochCu{Cu: 20},
			},
		},
		BadgeUsedCuList: []types.BadgeUsedCu{
			{
				BadgeUsedCuKey: []byte{byte(0)},
			},
			{
				BadgeUsedCuKey: []byte{byte(1)},
			},
		},
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.PairingKeeper(t)
	pairing.InitGenesis(ctx, *k, genesisState)
	got := pairing.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.UniqueEpochSessions, got.UniqueEpochSessions)
	require.ElementsMatch(t, genesisState.ProviderEpochCus, got.ProviderEpochCus)
	require.ElementsMatch(t, genesisState.ProviderConsumerEpochCus, got.ProviderConsumerEpochCus)
	require.ElementsMatch(t, genesisState.BadgeUsedCuList, got.BadgeUsedCuList)
	// this line is used by starport scaffolding # genesis/test/assert
}
