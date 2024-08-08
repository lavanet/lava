package types_test

import (
	"testing"

	"github.com/lavanet/lava/v2/x/rewards/types"
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
			desc: "invalid iprpc subscriptions",
			genState: &types.GenesisState{
				Params:              types.DefaultParams(),
				RefillRewardsTS:     types.DefaultGenesis().RefillRewardsTS,
				BasePays:            types.DefaultGenesis().BasePays,
				IprpcSubscriptions:  []string{"invalidAddress"},
				MinIprpcCost:        types.DefaultGenesis().MinIprpcCost,
				IprpcRewards:        types.DefaultGenesis().IprpcRewards,
				IprpcRewardsCurrent: types.DefaultGenesis().GetIprpcRewardsCurrent(),
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
