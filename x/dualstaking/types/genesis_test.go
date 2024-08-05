package types_test

import (
	"testing"

	"github.com/lavanet/lava/v2/x/dualstaking/types"
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
				DelegatorRewardList: []types.DelegatorReward{
					{
						Provider:  "p0",
						Delegator: "d0",
						ChainId:   "c0",
					},
					{
						Provider:  "p1",
						Delegator: "d1",
						ChainId:   "c1",
					},
				},
				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		{
			desc: "duplicated delegatorReward",
			genState: &types.GenesisState{
				DelegatorRewardList: []types.DelegatorReward{
					{
						Provider:  "p0",
						Delegator: "d0",
						ChainId:   "c0",
					},
					{
						Provider:  "p0",
						Delegator: "d0",
						ChainId:   "c0",
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
