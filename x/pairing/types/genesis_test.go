package types_test

import (
	"testing"

	"github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func TestGenesisState_Validate(t *testing.T) {
	for _, tc := range []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc: "valid genesis state",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				UniqueEpochSessions: []types.UniqueEpochSessionGenesis{
					{
						Epoch:              0,
						UniqueEpochSession: "0",
					},
					{
						Epoch:              1,
						UniqueEpochSession: "1",
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
				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		{
			desc: "duplicated UniqueEpochSessions",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				UniqueEpochSessions: []types.UniqueEpochSessionGenesis{
					{
						Epoch:              0,
						UniqueEpochSession: "0",
					},
					{
						Epoch:              0,
						UniqueEpochSession: "0",
					},
				},
			},
			valid: false,
		},
		{
			desc: "duplicated ProviderEpochCus",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ProviderEpochCus: []types.ProviderEpochCuGenesis{
					{
						Epoch:           0,
						Provider:        "0",
						ProviderEpochCu: types.ProviderEpochCu{ServicedCu: 10, ComplainersCu: 100},
					},
					{
						Epoch:           0,
						Provider:        "0",
						ProviderEpochCu: types.ProviderEpochCu{ServicedCu: 10, ComplainersCu: 100},
					},
				},
			},
			valid: false,
		},
		{
			desc: "duplicated epochPayments",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				ProviderConsumerEpochCus: []types.ProviderConsumerEpochCuGenesis{
					{
						Epoch:                   0,
						Provider:                "0",
						Project:                 "0",
						ProviderConsumerEpochCu: types.ProviderConsumerEpochCu{Cu: 10},
					},
					{
						Epoch:                   0,
						Provider:                "0",
						Project:                 "0",
						ProviderConsumerEpochCu: types.ProviderConsumerEpochCu{Cu: 10},
					},
				},
			},
			valid: false,
		},
		// this line is used by starport scaffolding # types/genesis/testcase
	} {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
