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

	_, err := ts.servers.SubscriptionServer.Subscribe(ts.ctx, &subtypes.MsgSubscribe{Creator: consumer.Addr.String(), Consumer: consumer.Addr.String(), Index: ts.plan.Index, IsYearly: false})
	require.Nil(t, err)

	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	pairingReq := types.QueryGetPairingRequest{ChainID: ts.spec.Index, Client: consumer.Addr.String()}
	pairing, err := ts.keepers.Pairing.GetPairing(ts.ctx, &pairingReq)
	require.Nil(t, err)

	verifyPairingQuery := &types.QueryVerifyPairingRequest{ChainID: ts.spec.Index, Client: consumer.Addr.String(), Provider: pairing.Providers[0].Address, Block: uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight())}
	vefiry, err := ts.keepers.Pairing.VerifyPairing(ts.ctx, verifyPairingQuery)
	require.Nil(t, err)
	require.True(t, vefiry.Valid)

	project, err := ts.keepers.Projects.GetProjectForDeveloper(sdk.UnwrapSDKContext(ts.ctx), consumer.Addr.String(), uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight()))
	require.Nil(t, err)

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

	_, err := ts.servers.SubscriptionServer.Subscribe(ts.ctx, &subtypes.MsgSubscribe{Creator: consumer.Addr.String(), Consumer: consumer.Addr.String(), Index: ts.plan.Index, IsYearly: false})
	require.Nil(t, err)

	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	pairingReq := types.QueryGetPairingRequest{ChainID: ts.spec.Index, Client: consumer.Addr.String()}
	pairing, err := ts.keepers.Pairing.GetPairing(ts.ctx, &pairingReq)
	require.Nil(t, err)

	verifyPairingQuery := &types.QueryVerifyPairingRequest{ChainID: ts.spec.Index, Client: consumer.Addr.String(), Provider: pairing.Providers[0].Address, Block: uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight())}
	vefiry, err := ts.keepers.Pairing.VerifyPairing(ts.ctx, verifyPairingQuery)
	require.Nil(t, err)
	require.True(t, vefiry.Valid)

	_, err = ts.keepers.Projects.GetProjectForDeveloper(sdk.UnwrapSDKContext(ts.ctx), consumer.Addr.String(), uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight()))
	require.Nil(t, err)

	// define tests - different epoch, valid tells if the payment request should work
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

			relayRequest := common.CreateRelay(
				t,
				*ts.providers[0],
				consumer,
				[]byte(ts.spec.Apis[0].Name),
				uint64(i),
				ts.spec.Name,
				tt.cu,
				sdk.UnwrapSDKContext(ts.ctx).BlockHeight(),
				0,
				-1,
				nil,
			)

			_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: []*types.RelayRequest{&relayRequest}})
			require.Equal(t, tt.valid, err == nil)
		})
	}
}

func TestRelayPaymentSubscriptionCU(t *testing.T) {
	ts := setupForPaymentTest(t)
	var balance int64 = 10000
	consumer := common.CreateNewAccount(ts.ctx, *ts.keepers, balance)

	_, err := ts.servers.SubscriptionServer.Subscribe(ts.ctx, &subtypes.MsgSubscribe{Creator: consumer.Addr.String(), Consumer: consumer.Addr.String(), Index: ts.plan.Index, IsYearly: false})
	require.Nil(t, err)

	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	pairingReq := types.QueryGetPairingRequest{ChainID: ts.spec.Index, Client: consumer.Addr.String()}
	pairing, err := ts.keepers.Pairing.GetPairing(ts.ctx, &pairingReq)
	require.Nil(t, err)

	verifyPairingQuery := &types.QueryVerifyPairingRequest{ChainID: ts.spec.Index, Client: consumer.Addr.String(), Provider: pairing.Providers[0].Address, Block: uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight())}
	vefiry, err := ts.keepers.Pairing.VerifyPairing(ts.ctx, verifyPairingQuery)
	require.Nil(t, err)
	require.True(t, vefiry.Valid)

	_, err = ts.keepers.Projects.GetProjectForDeveloper(sdk.UnwrapSDKContext(ts.ctx), consumer.Addr.String(), uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight()))
	require.Nil(t, err)

	i := 0
	for ; uint64(i) < ts.plan.ComputeUnits/ts.plan.ComputeUnitsPerEpoch; i++ {

		relayRequest := common.CreateRelay(
			t,
			*ts.providers[0],
			consumer,
			[]byte(ts.spec.Apis[0].Name),
			uint64(i),
			ts.spec.Name,
			ts.plan.ComputeUnitsPerEpoch,
			sdk.UnwrapSDKContext(ts.ctx).BlockHeight(),
			0,
			-1,
			nil,
		)

		_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: []*types.RelayRequest{&relayRequest}})
		require.Nil(t, err)

		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	//last iteration should finish the plan quota
	relayRequest := &types.RelayRequest{
		Provider:        ts.providers[0].Addr.String(),
		ApiUrl:          "",
		Data:            []byte(ts.spec.Apis[0].Name),
		SessionId:       uint64(i + 1),
		ChainID:         ts.spec.Name,
		CuSum:           ts.plan.ComputeUnitsPerEpoch,
		BlockHeight:     sdk.UnwrapSDKContext(ts.ctx).BlockHeight(),
		RelayNum:        0,
		RequestBlock:    -1,
		DataReliability: nil,
	}

	sig, err := sigs.SignRelay(consumer.SK, *relayRequest)
	relayRequest.Sig = sig
	require.Nil(t, err)

	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: []*types.RelayRequest{relayRequest}})
	require.NotNil(t, err)
}
