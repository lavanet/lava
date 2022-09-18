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

				UniquePaymentStorageClientProviderList: []types.UniquePaymentStorageClientProvider{
					{
						Index: "0",
					},
					{
						Index: "1",
					},
				},
				ProviderPaymentStorageList: []types.ProviderPaymentStorage{
					{
						Index: "0",
					},
					{
						Index: "1",
					},
				},
				EpochPaymentsList: []types.EpochPayments{
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
			desc: "duplicated uniquePaymentStorageClientProvider",
			genState: &types.GenesisState{
				UniquePaymentStorageClientProviderList: []types.UniquePaymentStorageClientProvider{
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
			desc: "duplicated providerPaymentStorage",
			genState: &types.GenesisState{
				ProviderPaymentStorageList: []types.ProviderPaymentStorage{
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
			desc: "duplicated epochPayments",
			genState: &types.GenesisState{
				EpochPaymentsList: []types.EpochPayments{
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
