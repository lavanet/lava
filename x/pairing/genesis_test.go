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
				Epoch:     0,
				Provider:  "0",
				Project:   "0",
				ChainId:   "0",
				SessionId: 0,
			},
			{
				Epoch:     1,
				Provider:  "1",
				Project:   "1",
				ChainId:   "1",
				SessionId: 1,
			},
		},
		ProviderEpochCus: []types.ProviderEpochCuGenesis{
			{
				Epoch:           0,
				Provider:        "0",
				ChainId:         "0",
				ProviderEpochCu: types.ProviderEpochCu{ServicedCu: 10},
			},
			{
				Epoch:           1,
				Provider:        "1",
				ChainId:         "1",
				ProviderEpochCu: types.ProviderEpochCu{ServicedCu: 20},
			},
		},
		ProviderEpochComplainedCus: []types.ProviderEpochComplainerCuGenesis{
			{
				Epoch:                     0,
				Provider:                  "0",
				ChainId:                   "0",
				ProviderEpochComplainerCu: types.ProviderEpochComplainerCu{ComplainersCu: 100},
			},
			{
				Epoch:                     1,
				Provider:                  "1",
				ChainId:                   "1",
				ProviderEpochComplainerCu: types.ProviderEpochComplainerCu{ComplainersCu: 200},
			},
		},
		ProviderConsumerEpochCus: []types.ProviderConsumerEpochCuGenesis{
			{
				Epoch:                   0,
				Provider:                "0",
				Project:                 "0",
				ChainId:                 "0",
				ProviderConsumerEpochCu: types.ProviderConsumerEpochCu{Cu: 10},
			},
			{
				Epoch:                   1,
				Provider:                "1",
				Project:                 "1",
				ChainId:                 "1",
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
		Reputations: []types.ReputationGenesis{
			{
				ChainId:    "0",
				Cluster:    "0",
				Provider:   "0",
				Reputation: types.Reputation{},
			},
			{
				ChainId:    "1",
				Cluster:    "1",
				Provider:   "1",
				Reputation: types.Reputation{},
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
	require.ElementsMatch(t, genesisState.Reputations, got.Reputations)
	// this line is used by starport scaffolding # genesis/test/assert
}
