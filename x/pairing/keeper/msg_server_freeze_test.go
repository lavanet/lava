package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

// Test freeze/unfreeze and its effect on the pairing list
func TestFreeze(t *testing.T) {
	providersNum := 2
	clientsNum := 1
	ts := setupClientsAndProvidersForUnresponsiveness(t, clientsNum, providersNum)

	// advance epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// get pairing list
	pairingList, err := ts.keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ts.ctx), ts.spec.GetIndex(), ts.clients[0].address)
	require.Nil(t, err)
	require.Equal(t, providersNum, len(pairingList))

	// freeze the first provider
	providerToFreeze := pairingList[0]
	_, err = ts.servers.PairingServer.Freeze(ts.ctx, &types.MsgFreeze{
		Creator:  providerToFreeze.Address,
		ChainIds: []string{ts.spec.GetIndex()},
		Reason:   "dummyReason"})
	require.Nil(t, err)

	// check that the provider is still shown in the pairing list
	pairingList, err = ts.keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ts.ctx), ts.spec.GetIndex(), ts.clients[0].address)
	require.Nil(t, err)
	require.Equal(t, providersNum, len(pairingList))
	require.Equal(t, providerToFreeze.Address, pairingList[0].Address)

	// advance epoch and verify the provider is not in the pairing list anymore
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	pairingList, err = ts.keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ts.ctx), ts.spec.GetIndex(), ts.clients[0].address)
	require.Nil(t, err)
	require.Equal(t, providersNum-1, len(pairingList))
	for _, provider := range pairingList {
		require.NotEqual(t, providerToFreeze.Address, provider.Address)
	}

	// unfreeze the provider and verify he's not in the pairing list
	_, err = ts.servers.PairingServer.Unfreeze(ts.ctx, &types.MsgUnfreeze{
		Creator:  providerToFreeze.Address,
		ChainIds: []string{ts.spec.GetIndex()},
	})
	require.Nil(t, err)
	pairingList, err = ts.keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ts.ctx), ts.spec.GetIndex(), ts.clients[0].address)
	require.Nil(t, err)
	require.Equal(t, providersNum-1, len(pairingList))
	for _, provider := range pairingList {
		require.NotEqual(t, providerToFreeze.Address, provider.Address)
	}

	// advance an epoch and verify the provider is in the pairing list
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	pairingList, err = ts.keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ts.ctx), ts.spec.GetIndex(), ts.clients[0].address)
	require.Nil(t, err)
	require.Equal(t, providersNum, len(pairingList))
	foundUnfrozenProvider := false
	for _, provider := range pairingList {
		if providerToFreeze.Address == provider.Address {
			foundUnfrozenProvider = true
		}
	}
	require.True(t, foundUnfrozenProvider)
}

// Test the freeze effect on the "providers" query
func TestProvidersQuery(t *testing.T) {
	providersNum := 2
	clientsNum := 1
	ts := setupClientsAndProvidersForUnresponsiveness(t, clientsNum, providersNum)

	// advance epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// get providers
	providersMsgResponse, err := ts.keepers.Pairing.Providers(ts.ctx, &types.QueryProvidersRequest{
		ChainID:             ts.spec.GetIndex(),
		ShowFrozenProviders: false,
	})
	require.Nil(t, err)
	providers := providersMsgResponse.GetStakeEntry()

	// freeze the first provider
	providerToFreeze := providers[0]
	_, err = ts.servers.PairingServer.Freeze(ts.ctx, &types.MsgFreeze{
		Creator:  providerToFreeze.Address,
		ChainIds: []string{ts.spec.GetIndex()},
		Reason:   "dummyReason"})
	require.Nil(t, err)

	// get providers without frozen providers and verify that providerToFreeze is not shown
	providersMsgResponse, err = ts.keepers.Pairing.Providers(ts.ctx, &types.QueryProvidersRequest{
		ChainID:             ts.spec.GetIndex(),
		ShowFrozenProviders: false,
	})
	require.Nil(t, err)
	for _, provider := range providersMsgResponse.StakeEntry {
		require.NotEqual(t, providerToFreeze.Address, provider.Address)
	}

	// get providers with frozen providers and verify that providerToFreeze is shown
	providersMsgResponse, err = ts.keepers.Pairing.Providers(ts.ctx, &types.QueryProvidersRequest{
		ChainID:             ts.spec.GetIndex(),
		ShowFrozenProviders: true,
	})
	require.Nil(t, err)
	foundFrozenProvider := false
	for _, provider := range providersMsgResponse.StakeEntry {
		if providerToFreeze.Address == provider.Address {
			foundFrozenProvider = true
		}

	}
	require.True(t, foundFrozenProvider)
}
