package types_test

import (
	"testing"

	"github.com/lavanet/lava/x/packages/types"
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
				PackageEntryList: []types.PackageEntry{
					{
						PackageIndex: "0",
					},
					{
						PackageIndex: "1",
					},
				},
				PackageUniqueIndexList: []types.PackageUniqueIndex{
					{
						Id: 0,
					},
					{
						Id: 1,
					},
				},
				PackageUniqueIndexCount: 2,
				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		{
			desc: "duplicated packageEntry",
			genState: &types.GenesisState{
				PackageEntryList: []types.PackageEntry{
					{
						PackageIndex: "0",
					},
					{
						PackageIndex: "0",
					},
				},
			},
			valid: false,
		},
		{
			desc: "duplicated packageUniqueIndex",
			genState: &types.GenesisState{
				PackageUniqueIndexList: []types.PackageUniqueIndex{
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
			desc: "invalid packageUniqueIndex count",
			genState: &types.GenesisState{
				PackageUniqueIndexList: []types.PackageUniqueIndex{
					{
						Id: 1,
					},
				},
				PackageUniqueIndexCount: 0,
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
