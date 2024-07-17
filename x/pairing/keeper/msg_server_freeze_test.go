package keeper_test

import (
	"testing"

	"github.com/lavanet/lava/v2/testutil/common"
	"github.com/lavanet/lava/v2/utils/sigs"
	"github.com/stretchr/testify/require"
)

// Test freeze/unfreeze and its effect on the pairing list
func TestFreeze(t *testing.T) {
	ts := newTester(t)

	providersCount := 2
	ts.setupForPayments(providersCount, 1, providersCount) // 1 client, set providers-to-pair

	_, clientAddr := ts.GetAccount(common.CONSUMER, 0)

	// get pairing list
	res, err := ts.QueryPairingGetPairing(ts.spec.Index, clientAddr)
	require.NoError(t, err)
	pairingList := res.Providers
	require.Equal(t, providersCount, len(pairingList))

	providerToFreeze := pairingList[0]

	// test that unfreeze does nothing
	_, err = ts.TxPairingUnfreezeProvider(providerToFreeze.Address, ts.spec.Index)
	require.NoError(t, err)

	res, err = ts.QueryPairingGetPairing(ts.spec.Index, clientAddr)
	require.NoError(t, err)
	pairingList = res.Providers
	require.Equal(t, providersCount, len(pairingList))

	// freeze the first provider
	_, err = ts.TxPairingFreezeProvider(providerToFreeze.Address, ts.spec.Index)
	require.NoError(t, err)

	// check that the provider is still shown in the pairing list
	res, err = ts.QueryPairingGetPairing(ts.spec.Index, clientAddr)
	require.NoError(t, err)
	pairingList = res.Providers
	require.Equal(t, providersCount, len(pairingList))
	require.Equal(t, providerToFreeze.Address, pairingList[0].Address)

	// advance epoch and verify the provider is not in the pairing list anymore
	ts.AdvanceEpoch()

	res, err = ts.QueryPairingGetPairing(ts.spec.Index, clientAddr)
	require.NoError(t, err)
	pairingList = res.Providers
	require.Equal(t, providersCount-1, len(pairingList))
	for _, provider := range pairingList {
		require.NotEqual(t, providerToFreeze.Address, provider.Address)
	}

	// unfreeze the provider and verify it remain not in the pairing list
	_, err = ts.TxPairingUnfreezeProvider(providerToFreeze.Address, ts.spec.Index)
	require.NoError(t, err)

	res, err = ts.QueryPairingGetPairing(ts.spec.Index, clientAddr)
	require.NoError(t, err)
	pairingList = res.Providers
	require.Equal(t, providersCount-1, len(pairingList))
	for _, provider := range pairingList {
		require.NotEqual(t, providerToFreeze.Address, provider.Address)
	}

	// advance an epoch and verify the provider is back in the pairing list
	ts.AdvanceEpochs(2)

	res, err = ts.QueryPairingGetPairing(ts.spec.Index, clientAddr)
	require.NoError(t, err)
	pairingList = res.Providers
	require.Equal(t, providersCount, len(pairingList))
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
	ts := newTester(t)

	providersCount := 2
	ts.setupForPayments(providersCount, 1, providersCount) // 1 client, set providers-to-pair

	// get providers
	res, err := ts.QueryPairingProviders(ts.spec.Index, false)
	require.NoError(t, err)

	// freeze the first provider
	providerToFreeze := res.StakeEntry[0]
	_, err = ts.TxPairingFreezeProvider(providerToFreeze.Address, ts.spec.Index)
	require.NoError(t, err)

	// get providers without frozen providers and verify that providerToFreeze is not shown
	res, err = ts.QueryPairingProviders(ts.spec.Index, false)
	require.NoError(t, err)
	for _, provider := range res.StakeEntry {
		require.NotEqual(t, providerToFreeze.Address, provider.Address)
	}

	// get providers with frozen providers and verify that providerToFreeze is shown
	res, err = ts.QueryPairingProviders(ts.spec.Index, true)
	require.NoError(t, err)
	foundFrozenProvider := false
	for _, provider := range res.StakeEntry {
		if providerToFreeze.Address == provider.Address {
			foundFrozenProvider = true
		}
	}
	require.True(t, foundFrozenProvider)
}

func TestUnstakeFrozen(t *testing.T) {
	ts := newTester(t)

	providersCount := 2
	ts.setupForPayments(providersCount, 1, providersCount) // 1 client, set providers-to-pair

	_, clientAddr := ts.GetAccount(common.CONSUMER, 0)

	// get pairing list
	res, err := ts.QueryPairingGetPairing(ts.spec.Index, clientAddr)
	require.NoError(t, err)
	pairingList := res.Providers
	require.Equal(t, providersCount, len(pairingList))

	// freeze the first provider
	providerToFreeze := pairingList[0]
	_, err = ts.TxPairingFreezeProvider(providerToFreeze.Address, ts.spec.Index)
	require.NoError(t, err)

	// check that the provider is still shown in the pairing list
	res, err = ts.QueryPairingGetPairing(ts.spec.Index, clientAddr)
	require.NoError(t, err)
	pairingList = res.Providers
	require.Equal(t, providersCount, len(pairingList))
	require.Equal(t, providerToFreeze.Address, pairingList[0].Address)

	// advance epoch and verify the provider is not in the pairing list anymore
	ts.AdvanceEpoch()

	res, err = ts.QueryPairingGetPairing(ts.spec.Index, clientAddr)
	require.NoError(t, err)
	pairingList = res.Providers
	require.Equal(t, providersCount-1, len(pairingList))
	for _, provider := range pairingList {
		require.NotEqual(t, providerToFreeze.Address, provider.Address)
	}

	_, err = ts.TxPairingUnstakeProvider(providerToFreeze.Vault, ts.spec.Index)
	require.NoError(t, err)

	// unfreeze the provider and verify it remains not in the pairing list
	_, err = ts.TxPairingUnfreezeProvider(providerToFreeze.Address, ts.spec.Index)
	require.Error(t, err)

	res, err = ts.QueryPairingGetPairing(ts.spec.Index, clientAddr)
	require.NoError(t, err)
	pairingList = res.Providers
	require.Equal(t, providersCount-1, len(pairingList))
	for _, provider := range pairingList {
		require.NotEqual(t, providerToFreeze.Address, provider.Address)
	}

	// advance an epoch and verify the provider is in the pairing list
	ts.AdvanceEpoch()

	res, err = ts.QueryPairingGetPairing(ts.spec.Index, clientAddr)
	require.NoError(t, err)
	pairingList = res.Providers
	require.Equal(t, providersCount-1, len(pairingList))
	foundUnfrozenProvider := false
	for _, provider := range pairingList {
		if providerToFreeze.Address == provider.Address {
			foundUnfrozenProvider = true
		}
	}
	require.False(t, foundUnfrozenProvider)
}

func TestPaymentFrozen(t *testing.T) {
	ts := newTester(t)

	providersCount := 2
	ts.setupForPayments(providersCount, 1, providersCount) // 1 client, set providers-to-pair

	clientAcct, clientAddr := ts.GetAccount(common.CONSUMER, 0)

	blockPreFreeze := ts.BlockHeight()
	ts.AdvanceEpoch()

	// get pairing list
	res, err := ts.QueryPairingGetPairing(ts.spec.Index, clientAddr)
	require.NoError(t, err)
	pairingList := res.Providers
	require.Equal(t, providersCount, len(pairingList))

	// freeze the first provider
	providerToFreeze := pairingList[0]
	_, err = ts.TxPairingFreezeProvider(providerToFreeze.Address, ts.spec.Index)
	require.NoError(t, err)

	// check that the provider is still shown in the pairing list
	res, err = ts.QueryPairingGetPairing(ts.spec.Index, clientAddr)
	require.NoError(t, err)
	pairingList = res.Providers
	require.Equal(t, providersCount, len(pairingList))
	require.Equal(t, providerToFreeze.Address, pairingList[0].Address)

	// advance epoch and verify the provider remains not in the pairing list anymore
	ts.AdvanceEpoch()

	res, err = ts.QueryPairingGetPairing(ts.spec.Index, clientAddr)
	require.NoError(t, err)
	pairingList = res.Providers
	require.Equal(t, providersCount-1, len(pairingList))
	for _, provider := range pairingList {
		require.NotEqual(t, providerToFreeze.Address, provider.Address)
	}

	cusum := ts.spec.ApiCollections[0].Apis[0].ComputeUnits * 10
	relaySession := ts.newRelaySession(providerToFreeze.Address, 1, cusum, blockPreFreeze, 0)

	sig, err := sigs.Sign(clientAcct.SK, *relaySession)
	relaySession.Sig = sig
	require.NoError(t, err)

	_, err = ts.TxPairingRelayPayment(providerToFreeze.Address, relaySession)
	require.NoError(t, err)
}
