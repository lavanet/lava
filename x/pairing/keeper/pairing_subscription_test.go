package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/utils/sigs"
	"github.com/lavanet/lava/x/pairing/types"
	projectstypes "github.com/lavanet/lava/x/projects/types"
	subtypes "github.com/lavanet/lava/x/subscription/types"
	"github.com/stretchr/testify/require"
)

func TestGetPairingForSubscription(t *testing.T) {
	ts := setupForPaymentTest(t)
	var balance int64 = 10000

	consumer := common.CreateNewAccount(ts.ctx, *ts.keepers, balance).Addr.String()
	vrfpk_stub := "testvrfpk"
	msgBuy := &subtypes.MsgBuy{
		Creator:  consumer,
		Consumer: consumer,
		Index:    ts.plan.Index,
		Duration: 1,
		Vrfpk:    vrfpk_stub,
	}
	_, err := ts.servers.SubscriptionServer.Buy(ts.ctx, msgBuy)
	require.Nil(t, err)

	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	ctx := sdk.UnwrapSDKContext(ts.ctx)

	pairingReq := types.QueryGetPairingRequest{
		ChainID: ts.spec.Index,
		Client:  consumer,
	}
	pairing, err := ts.keepers.Pairing.GetPairing(ts.ctx, &pairingReq)
	require.Nil(t, err)

	verifyPairingQuery := &types.QueryVerifyPairingRequest{
		ChainID:  ts.spec.Index,
		Client:   consumer,
		Provider: pairing.Providers[0].Address,
		Block:    uint64(ctx.BlockHeight()),
	}
	vefiry, err := ts.keepers.Pairing.VerifyPairing(ts.ctx, verifyPairingQuery)
	require.Nil(t, err)
	require.True(t, vefiry.Valid)

	project, vrfpk, err := ts.keepers.Projects.GetProjectForDeveloper(ctx, consumer, uint64(ctx.BlockHeight()))
	require.Nil(t, err)
	require.Equal(t, vrfpk, vrfpk_stub)

	err = ts.keepers.Projects.DeleteProject(ctx, project.Index)
	require.Nil(t, err)

	_, err = ts.keepers.Pairing.GetPairing(ts.ctx, &pairingReq)
	require.NotNil(t, err)

	verifyPairingQuery = &types.QueryVerifyPairingRequest{
		ChainID:  ts.spec.Index,
		Client:   consumer,
		Provider: pairing.Providers[0].Address,
		Block:    uint64(ctx.BlockHeight()),
	}
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

	proj, _, err := ts.keepers.Projects.GetProjectForDeveloper(sdk.UnwrapSDKContext(ts.ctx), consumer.Addr.String(), uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight()))
	require.Nil(t, err)

	policies := []*projectstypes.Policy{proj.AdminPolicy, proj.SubscriptionPolicy, &ts.plan.PlanPolicy}
	sub, found := ts.keepers.Subscription.GetSubscription(sdk.UnwrapSDKContext(ts.ctx), proj.GetSubscription())
	require.True(t, found)
	allowedCu := ts.keepers.Pairing.CalculateEffectiveAllowedCuPerEpochFromPolicies(policies, proj.GetUsedCu(), sub.GetMonthCuLeft())

	tests := []struct {
		name  string
		cu    uint64
		valid bool
	}{
		{"happyflow", ts.spec.Apis[0].ComputeUnits, true},
		{"epochCULimit", allowedCu + 1, false},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relayRequest := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), tt.cu, ts.spec.Name, nil)
			relayRequest.SessionId = uint64(i)
			relayRequest.Sig, err = sigs.SignRelay(consumer.SK, *relayRequest)
			require.Nil(t, err)
			_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: []*types.RelaySession{relayRequest}})
			require.Equal(t, tt.valid, err == nil, "results incorrect for usage of %d err == nil: %t", tt.cu, err == nil)
		})
	}
}

func TestRelayPaymentSubscriptionCU(t *testing.T) {
	ts := setupForPaymentTest(t)
	var balance int64 = 10000
	consumerA := common.CreateNewAccount(ts.ctx, *ts.keepers, balance)
	consumerB := common.CreateNewAccount(ts.ctx, *ts.keepers, balance)

	consumers := []common.Account{consumerA, consumerB}

	_, err := ts.servers.SubscriptionServer.Buy(ts.ctx, &subtypes.MsgBuy{Creator: consumerA.Addr.String(), Consumer: consumerA.Addr.String(), Index: ts.plan.Index, Duration: 1})
	require.Nil(t, err)

	consumerBProjectData := projectstypes.ProjectData{
		Name:        "consumerBProject",
		Description: "",
		Enabled:     true,
		ProjectKeys: []projectstypes.ProjectKey{{
			Key: consumerB.Addr.String(),
			Types: []projectstypes.ProjectKey_KEY_TYPE{
				projectstypes.ProjectKey_ADMIN,
				projectstypes.ProjectKey_DEVELOPER,
			},
			Vrfpk: "",
		}},
		Policy: &ts.plan.PlanPolicy,
	}
	err = ts.keepers.Subscription.AddProjectToSubscription(sdk.UnwrapSDKContext(ts.ctx), consumerA.Addr.String(), consumerBProjectData)
	require.Nil(t, err)

	// verify both projects exist
	projA, _, err := ts.keepers.Projects.GetProjectForDeveloper(sdk.UnwrapSDKContext(ts.ctx), consumerA.Addr.String(), uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight()))
	require.Nil(t, err)
	projB, _, err := ts.keepers.Projects.GetProjectForDeveloper(sdk.UnwrapSDKContext(ts.ctx), consumerB.Addr.String(), uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight()))
	require.Nil(t, err)

	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// verify that both consumers are paired
	for _, consumer := range consumers {
		pairingReq := types.QueryGetPairingRequest{ChainID: ts.spec.Index, Client: consumer.Addr.String()}
		pairing, err := ts.keepers.Pairing.GetPairing(ts.ctx, &pairingReq)
		require.Nil(t, err)

		verifyPairingQuery := &types.QueryVerifyPairingRequest{ChainID: ts.spec.Index, Client: consumer.Addr.String(), Provider: pairing.Providers[0].Address, Block: uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight())}
		verify, err := ts.keepers.Pairing.VerifyPairing(ts.ctx, verifyPairingQuery)
		require.Nil(t, err)
		require.True(t, verify.Valid)
	}

	// both projects have adminPolicy, subscriptionPolicy = nil -> they go by the plan policy
	// waste all the subscription's CU on project A
	i := 0
	for ; uint64(i) < ts.plan.PlanPolicy.GetTotalCuLimit()/ts.plan.PlanPolicy.GetEpochCuLimit(); i++ {

		relayRequest := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), ts.plan.PlanPolicy.GetEpochCuLimit(), ts.spec.Name, nil)
		relayRequest.SessionId = uint64(i)
		relayRequest.Sig, err = sigs.SignRelay(consumerA.SK, *relayRequest)
		require.Nil(t, err)
		_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: []*types.RelaySession{relayRequest}})
		require.Nil(t, err)

		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	// last iteration should finish the plan and subscription quota
	relayRequest := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), ts.plan.PlanPolicy.GetEpochCuLimit(), ts.spec.Name, nil)
	relayRequest.SessionId = uint64(i + 1)
	relayRequest.Sig, err = sigs.SignRelay(consumerA.SK, *relayRequest)
	require.Nil(t, err)
	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: []*types.RelaySession{relayRequest}})
	require.NotNil(t, err)

	// verify that project A wasted all of the subscription's CU
	sub, found := ts.keepers.Subscription.GetSubscription(sdk.UnwrapSDKContext(ts.ctx), projA.Subscription)
	require.True(t, found)
	require.Equal(t, uint64(0), sub.MonthCuLeft)
	projA, _, err = ts.keepers.Projects.GetProjectForDeveloper(sdk.UnwrapSDKContext(ts.ctx), consumerA.Addr.String(), uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight()))
	require.Nil(t, err)
	require.Equal(t, sub.MonthCuTotal, projA.UsedCu)
	require.Equal(t, uint64(0), projB.UsedCu)

	// try to use CU on projB. Should fail because A wasted it all
	relayRequest.SessionId += 1
	relayRequest.Sig, err = sigs.SignRelay(consumerB.SK, *relayRequest)
	require.Nil(t, err)
	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: []*types.RelaySession{relayRequest}})
	require.NotNil(t, err)
}
