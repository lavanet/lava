package types_test

import (
	"testing"

	"github.com/lavanet/lava/v2/x/spec/types"
	"github.com/stretchr/testify/require"
)

func TestGenesisState_Validate(t *testing.T) {
	defaultParamsWithDuplicateBlacklistedExpeditedMsgs := types.DefaultParams()
	defaultParamsWithDuplicateBlacklistedExpeditedMsgs.AllowlistedExpeditedMsgs = []string{"a", "a"}

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
				SpecList: []types.Spec{
					{
						Index: "0",
					},
					{
						Index: "1",
					},
				},
				SpecCount: 2,
				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		{
			desc: "duplicated spec",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				SpecList: []types.Spec{
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
			desc: "invalid spec count",
			genState: &types.GenesisState{
				Params: types.DefaultParams(),
				SpecList: []types.Spec{
					{
						Index: "1",
					},
				},
				SpecCount: 0,
			},
			valid: false,
		},
		{
			desc: "duplicated message in BlacklistedExpeditedMessages",
			genState: &types.GenesisState{
				Params:   defaultParamsWithDuplicateBlacklistedExpeditedMsgs,
				SpecList: []types.Spec{},
			},
			valid: false,
		},
		{
			desc: "valid message in BlacklistedExpeditedMessages",
			genState: &types.GenesisState{
				Params:   types.DefaultParams(),
				SpecList: []types.Spec{},
			},
			valid: true,
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
