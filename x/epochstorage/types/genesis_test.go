package types_test

import (
	"testing"

	"github.com/lavanet/lava/v2/x/epochstorage/types"
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
				StakeStorageList: []types.StakeStorage{
					{
						Index: "0",
					},
					{
						Index: "1",
					},
				},
				EpochDetails: &types.EpochDetails{
					StartBlock:    65,
					EarliestStart: 76,
				},
				FixatedParamsList: []types.FixatedParams{
					{
						Index: "0",
					},
					{
						Index: "1",
					},
				},
				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		{
			desc: "duplicated stakeStorage",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				StakeStorageList: []types.StakeStorage{
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
			desc: "duplicated fixatedParams",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				FixatedParamsList: []types.FixatedParams{
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
