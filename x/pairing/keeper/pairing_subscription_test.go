package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/x/pairing/types"
	subtypes "github.com/lavanet/lava/x/subscription/types"
	"github.com/stretchr/testify/require"
)

func TestGetPairingForSubscription(t *testing.T) {
	ts := setupForPaymentTest(t)
	var balance int64 = 10000
	consumer := common.CreateNewAccount(ts.ctx, *ts.keepers, balance)
	vrfpk_stub := "testvrfpk"
	_, err := ts.servers.SubscriptionServer.Buy(ts.ctx, &subtypes.MsgBuy{Creator: consumer.Addr.String(), Consumer: consumer.Addr.String(), Index: ts.plan.Index, Duration: 1, Vrfpk: vrfpk_stub})
	require.Nil(t, err)

	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	pairingReq := types.QueryGetPairingRequest{ChainID: ts.spec.Index, Client: consumer.Addr.String()}
	pairing, err := ts.keepers.Pairing.GetPairing(ts.ctx, &pairingReq)
	require.Nil(t, err)

	verifyPairingQuery := &types.QueryVerifyPairingRequest{ChainID: ts.spec.Index, Client: consumer.Addr.String(), Provider: pairing.Providers[0].Address, Block: uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight())}
	vefiry, err := ts.keepers.Pairing.VerifyPairing(ts.ctx, verifyPairingQuery)
	require.Nil(t, err)
	require.True(t, vefiry.Valid)

	project, vrfpk, err := ts.keepers.Projects.GetProjectForDeveloper(sdk.UnwrapSDKContext(ts.ctx), consumer.Addr.String(), uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight()))
	require.Nil(t, err)
	require.Equal(t, vrfpk, vrfpk_stub)
	err = ts.keepers.Projects.DeleteProject(sdk.UnwrapSDKContext(ts.ctx), project.Index)
	require.Nil(t, err)

	_, err = ts.keepers.Pairing.GetPairing(ts.ctx, &pairingReq)
	require.NotNil(t, err)

	verifyPairingQuery = &types.QueryVerifyPairingRequest{ChainID: ts.spec.Index, Client: consumer.Addr.String(), Provider: pairing.Providers[0].Address, Block: uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight())}
	vefiry, err = ts.keepers.Pairing.VerifyPairing(ts.ctx, verifyPairingQuery)
	require.NotNil(t, err)
	require.False(t, vefiry.Valid)
}

func TestRelayPaymentSubscription(t *testing.T) {
	ts := setupForPaymentTest(t)
	var balance int64 = 10000
	consumer := common.CreateNewAccount(ts.ctx, *ts.keepers, balance)

	_, err := ts.servers.SubscriptionServer.Buy(ts.ctx, &subtypes.MsgBuy{Creator: consumer.Addr.String(), Consumer: consumer.Addr.String(), Index: ts.plan.Index, Duration: 1})
	require.Nil(t, err)

	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	pairingReq := types.QueryGetPairingRequest{ChainID: ts.spec.Index, Client: consumer.Addr.String()}
	pairing, err := ts.keepers.Pairing.GetPairing(ts.ctx, &pairingReq)
	require.Nil(t, err)

	verifyPairingQuery := &types.QueryVerifyPairingRequest{ChainID: ts.spec.Index, Client: consumer.Addr.String(), Provider: pairing.Providers[0].Address, Block: uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight())}
	vefiry, err := ts.keepers.Pairing.VerifyPairing(ts.ctx, verifyPairingQuery)
	require.Nil(t, err)
	require.True(t, vefiry.Valid)

	_, _, err = ts.keepers.Projects.GetProjectForDeveloper(sdk.UnwrapSDKContext(ts.ctx), consumer.Addr.String(), uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight()))
	require.Nil(t, err)

	tests := []struct {
		name  string
		cu    uint64
		valid bool
	}{
		{"happyflow", ts.spec.Apis[0].ComputeUnits, true},
		{"epochCULimit", ts.plan.ComputeUnitsPerEpoch, false},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relayRequest := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), tt.cu, ts.spec.Name, nil)
			relayRequest.SessionId = uint64(i)
			relayRequest.Sig, err = sigs.SignRelay(ts.clients[0].SK, *relayRequest)
			require.Nil(t, err)
			_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: []*types.RelaySession{relayRequest}})
			require.Equal(t, tt.valid, err == nil)
		})
	}
}

func TestRelayPaymentSubscriptionCU(t *testing.T) {
	ts := setupForPaymentTest(t)
	var balance int64 = 10000
	consumer := common.CreateNewAccount(ts.ctx, *ts.keepers, balance)

	_, err := ts.servers.SubscriptionServer.Buy(ts.ctx, &subtypes.MsgBuy{Creator: consumer.Addr.String(), Consumer: consumer.Addr.String(), Index: ts.plan.Index, Duration: 1})
	require.Nil(t, err)

	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	pairingReq := types.QueryGetPairingRequest{ChainID: ts.spec.Index, Client: consumer.Addr.String()}
	pairing, err := ts.keepers.Pairing.GetPairing(ts.ctx, &pairingReq)
	require.Nil(t, err)

	verifyPairingQuery := &types.QueryVerifyPairingRequest{ChainID: ts.spec.Index, Client: consumer.Addr.String(), Provider: pairing.Providers[0].Address, Block: uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight())}
	vefiry, err := ts.keepers.Pairing.VerifyPairing(ts.ctx, verifyPairingQuery)
	require.Nil(t, err)
	require.True(t, vefiry.Valid)

	_, _, err = ts.keepers.Projects.GetProjectForDeveloper(sdk.UnwrapSDKContext(ts.ctx), consumer.Addr.String(), uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight()))
	require.Nil(t, err)

	i := 0
	for ; uint64(i) < ts.plan.ComputeUnits/ts.plan.ComputeUnitsPerEpoch; i++ {

		relayRequest := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), ts.plan.ComputeUnitsPerEpoch, ts.spec.Name, nil)
		relayRequest.SessionId = uint64(i)
		relayRequest.Sig, err = sigs.SignRelay(consumer.SK, *relayRequest)
		require.Nil(t, err)
		_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: []*types.RelaySession{relayRequest}})
		require.Nil(t, err)

		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	//last iteration should finish the plan quota
	relayRequest := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), ts.plan.ComputeUnitsPerEpoch, ts.spec.Name, nil)
	relayRequest.SessionId = uint64(i + 1)
	relayRequest.Sig, err = sigs.SignRelay(consumer.SK, *relayRequest)
	require.Nil(t, err)
	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: []*types.RelaySession{relayRequest}})
	require.NotNil(t, err)
}
