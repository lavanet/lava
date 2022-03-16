package types_test

import (
	"testing"

	"github.com/lavanet/lava/x/servicer/types"
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
				BlockDeadlineForCallback: &types.BlockDeadlineForCallback{
					Deadline: new(types.BlockNum),
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
				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		{
			desc: "duplicated stakeMap",
			genState: &types.GenesisState{
				StakeMapList: []types.StakeMap{
					{
						Index: "0",
					},
					{
						Index: "0",
					},
				},
			},
			valid: false,
		},
		{
			desc: "duplicated specStakeStorage",
			genState: &types.GenesisState{
				SpecStakeStorageList: []types.SpecStakeStorage{
					{
						Index: "0",
					},
					{
						Index: "0",
					},
				},
			},
			valid: false,
		},
		{
			desc: "duplicated unstakingServicersAllSpecs",
			genState: &types.GenesisState{
				UnstakingServicersAllSpecsList: []types.UnstakingServicersAllSpecs{
					{
						Id: 0,
					},
					{
						Id: 0,
					},
				},
			},
			valid: false,
		},
		{
			desc: "invalid unstakingServicersAllSpecs count",
			genState: &types.GenesisState{
				UnstakingServicersAllSpecsList: []types.UnstakingServicersAllSpecs{
					{
						Id: 1,
					},
				},
				UnstakingServicersAllSpecsCount: 0,
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
