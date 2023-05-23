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
		BadgeUsedCuList: []types.BadgeUsedCu{
			{
				BadgeUsedCuKey: []byte{byte(0)},
			},
			{
				BadgeUsedCuKey: []byte{byte(1)},
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

	require.ElementsMatch(t, genesisState.UniquePaymentStorageClientProviderList, got.UniquePaymentStorageClientProviderList)
	require.ElementsMatch(t, genesisState.ProviderPaymentStorageList, got.ProviderPaymentStorageList)
	require.ElementsMatch(t, genesisState.EpochPaymentsList, got.EpochPaymentsList)
	require.ElementsMatch(t, genesisState.BadgeUsedCuList, got.BadgeUsedCuList)
	// this line is used by starport scaffolding # genesis/test/assert
}
