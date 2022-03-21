package types_test

import (
	"testing"

	"github.com/lavanet/lava/x/user/types"
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
				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		{
			desc: "duplicated userStake",
			genState: &types.GenesisState{
				UserStakeList: []types.UserStake{
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
			desc: "duplicated unstakingUsersAllSpecs",
			genState: &types.GenesisState{
				UnstakingUsersAllSpecsList: []types.UnstakingUsersAllSpecs{
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
			desc: "invalid unstakingUsersAllSpecs count",
			genState: &types.GenesisState{
				UnstakingUsersAllSpecsList: []types.UnstakingUsersAllSpecs{
					{
						Id: 1,
					},
				},
				UnstakingUsersAllSpecsCount: 0,
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
