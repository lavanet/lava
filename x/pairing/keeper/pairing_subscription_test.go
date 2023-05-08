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

func TestStrictestPolicyGeolocation(t *testing.T) {
	ts := setupForPaymentTest(t)

	// make the plan policy's geolocation 7(=111)
	ts.plan.PlanPolicy.GeolocationProfile = 7
	err := ts.keepers.Plans.AddPlan(sdk.UnwrapSDKContext(ts.ctx), ts.plan)
	require.Nil(t, err)

	err = ts.keepers.Subscription.CreateSubscription(sdk.UnwrapSDKContext(ts.ctx),
		ts.clients[0].Addr.String(), ts.clients[0].Addr.String(), ts.plan.Index, 10, "")
	require.Nil(t, err)

	proj, _, err := ts.keepers.Projects.GetProjectForDeveloper(sdk.UnwrapSDKContext(ts.ctx),
		ts.clients[0].Addr.String(), uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight()))
	require.Nil(t, err)

	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	geolocationTestTemplates := []struct {
		name                   string
		geolocationAdminPolicy uint64
		geolocationSubPolicy   uint64
		success                bool
	}{
		{"effective geo = 1", uint64(1), uint64(1), true},
		{"effective geo = 3 (includes geo=1)", uint64(3), uint64(3), true},
		{"effective geo = 2", uint64(3), uint64(2), false},
		{"effective geo = 0 (planPolicy & subPolicy = 1)", uint64(2), uint64(1), false},
		{"effective geo = 0 (planPolicy & adminPolicy = 1)", uint64(1), uint64(2), false},
	}

	for _, tt := range geolocationTestTemplates {
		t.Run(tt.name, func(t *testing.T) {
			adminPolicy := &projectstypes.Policy{
				GeolocationProfile: tt.geolocationAdminPolicy,
			}
			subscriptionPolicy := &projectstypes.Policy{
				GeolocationProfile: tt.geolocationSubPolicy,
			}

			_, err = ts.servers.ProjectServer.SetAdminPolicy(ts.ctx, &projectstypes.MsgSetAdminPolicy{
				Creator: ts.clients[0].Addr.String(),
				Project: proj.Index,
				Policy:  *adminPolicy,
			})
			require.Nil(t, err)

			// apply the policy setting
			ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

			_, err = ts.servers.ProjectServer.SetSubscriptionPolicy(ts.ctx, &projectstypes.MsgSetSubscriptionPolicy{
				Creator:  ts.clients[0].Addr.String(),
				Projects: []string{proj.Index},
				Policy:   *subscriptionPolicy,
			})
			require.Nil(t, err)

			// apply the policy setting
			ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

			// the only provider is set with geolocation=1. So only geolocation that ANDs
			// with 1 and output non-zero result, will output a provider for pairing
			getPairingResponse, err := ts.keepers.Pairing.GetPairing(ts.ctx, &types.QueryGetPairingRequest{
				ChainID: ts.spec.Index,
				Client:  ts.clients[0].Addr.String(),
			})
			require.Nil(t, err)
			if tt.success {
				require.NotEqual(t, 0, len(getPairingResponse.Providers))
			} else {
				require.Equal(t, 0, len(getPairingResponse.Providers))
			}
		})
	}
}

func TestStrictestPolicyProvidersToPair(t *testing.T) {
	ts := setupForPaymentTest(t)

	// add 5 more providers so we can have enough providers for testing
	ts.addProvider(5)

	_, err := ts.servers.SubscriptionServer.Buy(ts.ctx, &subtypes.MsgBuy{
		Creator:  ts.clients[0].Addr.String(),
		Consumer: ts.clients[0].Addr.String(),
		Index:    ts.plan.Index,
		Duration: 10,
		Vrfpk:    "",
	})
	require.Nil(t, err)

	developerQueryResponse, err := ts.keepers.Projects.Developer(ts.ctx, &projectstypes.QueryDeveloperRequest{
		Developer: ts.clients[0].Addr.String(),
	})
	require.Nil(t, err)
	proj := developerQueryResponse.Project

	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	providersToPairTestTemplates := []struct {
		name                       string
		providersToPairAdminPolicy uint64
		providersToPairSubPolicy   uint64
		effectiveProvidersToPair   int
	}{
		{"effective providersToPair = 2", uint64(4), uint64(2), 2},
		{"sub policy providersToPair = 1", uint64(1), uint64(3), 3},
		{"admin policy providersToPair = 1", uint64(3), uint64(1), 3},
	}

	for _, tt := range providersToPairTestTemplates {
		t.Run(tt.name, func(t *testing.T) {
			adminPolicy := &projectstypes.Policy{
				GeolocationProfile: 1,
				MaxProvidersToPair: tt.providersToPairAdminPolicy,
			}
			subscriptionPolicy := &projectstypes.Policy{
				GeolocationProfile: 1,
				MaxProvidersToPair: tt.providersToPairSubPolicy,
			}

			_, err = ts.servers.ProjectServer.SetAdminPolicy(ts.ctx, &projectstypes.MsgSetAdminPolicy{
				Creator: ts.clients[0].Addr.String(),
				Project: proj.Index,
				Policy:  *adminPolicy,
			})
			require.Nil(t, err)

			// apply the policy setting
			ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

			_, err = ts.servers.ProjectServer.SetSubscriptionPolicy(ts.ctx, &projectstypes.MsgSetSubscriptionPolicy{
				Creator:  ts.clients[0].Addr.String(),
				Projects: []string{proj.Index},
				Policy:   *subscriptionPolicy,
			})
			require.Nil(t, err)

			// apply the policy setting
			ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

			getPairingResponse, err := ts.keepers.Pairing.GetPairing(ts.ctx, &types.QueryGetPairingRequest{
				ChainID: ts.spec.Index,
				Client:  ts.clients[0].Addr.String(),
			})
			require.Nil(t, err)
			require.Equal(t, tt.effectiveProvidersToPair, len(getPairingResponse.Providers))
		})
	}
}

func TestStrictestPolicyCuPerEpoch(t *testing.T) {
	ts := setupForPaymentTest(t)

	_, err := ts.servers.SubscriptionServer.Buy(ts.ctx, &subtypes.MsgBuy{
		Creator:  ts.clients[0].Addr.String(),
		Consumer: ts.clients[0].Addr.String(),
		Index:    ts.plan.Index,
		Duration: 10,
		Vrfpk:    "",
	})
	require.Nil(t, err)

	developerQueryResponse, err := ts.keepers.Projects.Developer(ts.ctx, &projectstypes.QueryDeveloperRequest{
		Developer: ts.clients[0].Addr.String(),
	})
	require.Nil(t, err)
	proj := developerQueryResponse.Project

	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	providersToPairTestTemplates := []struct {
		name                     string
		cuPerEpochAdminPolicy    uint64
		cuPerEpochSubPolicy      uint64
		useMostOfProjectCu       bool
		wasteSubscriptionCu      bool
		effectiveCuPerEpochLimit uint64
	}{
		{"admin policy min CU", uint64(90), uint64(110), false, false, uint64(90)},
		{"sub policy min CU", uint64(110), uint64(90), false, false, uint64(90)},
		{"use most of the project's CU", uint64(100), uint64(100), true, false, uint64(10)},
		{"waste subscription CU", uint64(100), uint64(100), false, true, uint64(0)},
	}

	for _, tt := range providersToPairTestTemplates {
		t.Run(tt.name, func(t *testing.T) {
			consumer := ts.clients[0]

			// add a new project to the subscription just to waste the subcsription's cu
			if tt.wasteSubscriptionCu {
				err = ts.addClient(1)
				require.Nil(t, err)

				consumerToWasteCu := ts.clients[1]

				// pair new client with provider
				ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

				projectData := projectstypes.ProjectData{
					Name:        "lowCuProject",
					Description: "project with low CU limit (per epoch)",
					Enabled:     true,
					ProjectKeys: []projectstypes.ProjectKey{{
						Key: consumerToWasteCu.Addr.String(),
						Types: []projectstypes.ProjectKey_KEY_TYPE{
							projectstypes.ProjectKey_DEVELOPER,
							projectstypes.ProjectKey_ADMIN,
						},
						Vrfpk: "",
					}},
					Policy: &ts.plan.PlanPolicy,
				}
				_, err = ts.servers.SubscriptionServer.AddProject(ts.ctx, &subtypes.MsgAddProject{
					Creator:     proj.Subscription,
					ProjectData: projectData,
				})
				require.Nil(t, err)

				sub, found := ts.keepers.Subscription.GetSubscription(sdk.UnwrapSDKContext(ts.ctx), proj.Subscription)
				require.True(t, found)

				relayRequest := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), sub.MonthCuLeft, ts.spec.Name, nil)
				relayRequest.SessionId = uint64(100)
				relayRequest.Sig, err = sigs.SignRelay(consumerToWasteCu.SK, *relayRequest)
				require.Nil(t, err)
				_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: []*types.RelaySession{relayRequest}})
				require.Nil(t, err)

				ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
			}

			adminPolicy := &projectstypes.Policy{
				GeolocationProfile: 1,
				EpochCuLimit:       tt.cuPerEpochAdminPolicy,
				TotalCuLimit:       1000,
			}
			subscriptionPolicy := &projectstypes.Policy{
				GeolocationProfile: 1,
				EpochCuLimit:       tt.cuPerEpochSubPolicy,
				TotalCuLimit:       1000,
			}

			_, err = ts.servers.ProjectServer.SetAdminPolicy(ts.ctx, &projectstypes.MsgSetAdminPolicy{
				Creator: consumer.Addr.String(),
				Project: proj.Index,
				Policy:  *adminPolicy,
			})
			require.Nil(t, err)

			// apply the policy setting
			ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

			_, err = ts.servers.ProjectServer.SetSubscriptionPolicy(ts.ctx, &projectstypes.MsgSetSubscriptionPolicy{
				Creator:  ts.clients[0].Addr.String(),
				Projects: []string{proj.Index},
				Policy:   *subscriptionPolicy,
			})
			require.Nil(t, err)

			// apply the policy setting
			ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

			// leave 10 CU in the project
			if tt.useMostOfProjectCu {
				for i := 0; uint64(i) < adminPolicy.TotalCuLimit/ts.plan.PlanPolicy.GetEpochCuLimit(); i++ {
					cuSum := ts.plan.PlanPolicy.GetEpochCuLimit()

					developerQueryResponse, err := ts.keepers.Projects.Developer(ts.ctx, &projectstypes.QueryDeveloperRequest{
						Developer: ts.clients[0].Addr.String(),
					})
					require.Nil(t, err)
					proj := developerQueryResponse.Project
					if proj.UsedCu >= 900 {
						cuSum = 90
					}

					relayRequest := common.BuildRelayRequest(ts.ctx, ts.providers[0].Addr.String(), []byte(ts.spec.Apis[0].Name), cuSum, ts.spec.Name, nil)
					relayRequest.SessionId = uint64(i)
					relayRequest.Sig, err = sigs.SignRelay(consumer.SK, *relayRequest)
					require.Nil(t, err)
					_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].Addr.String(), Relays: []*types.RelaySession{relayRequest}})
					require.Nil(t, err)

					ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
				}
			}

			_, _, _, cuPerEpochLimit, _, _, err := ts.keepers.Pairing.ValidatePairingForClient(sdk.UnwrapSDKContext(ts.ctx), ts.spec.Index,
				consumer.Addr, ts.providers[0].Addr, uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight()))
			require.Nil(t, err)

			require.Equal(t, tt.effectiveCuPerEpochLimit, cuPerEpochLimit)
		})
	}
}

func TestPairingNotChangingDueToCuOveruse(t *testing.T) {
	ts := setupForPaymentTest(t)
	err := ts.addProvider(100)
	require.Nil(t, err)

	// advance epoch to get pairing
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	_, err = ts.servers.SubscriptionServer.Buy(ts.ctx, &subtypes.MsgBuy{
		Creator:  ts.clients[0].Addr.String(),
		Consumer: ts.clients[0].Addr.String(),
		Index:    ts.plan.Index,
		Duration: 11,
		Vrfpk:    "",
	})
	require.Nil(t, err)

	for i := 0; i < int(ts.plan.PlanPolicy.TotalCuLimit)/int(ts.plan.PlanPolicy.EpochCuLimit); i++ {
		res, err := ts.keepers.Pairing.GetPairing(ts.ctx, &types.QueryGetPairingRequest{
			ChainID: ts.spec.Index,
			Client:  ts.clients[0].Addr.String(),
		})
		require.Nil(t, err)

		cuSum := ts.plan.PlanPolicy.GetEpochCuLimit()
		relayRequest := common.BuildRelayRequest(ts.ctx, res.Providers[0].Address, []byte(ts.spec.Apis[0].Name), cuSum, ts.spec.Name, nil)
		relayRequest.SessionId = uint64(i)
		relayRequest.Sig, err = sigs.SignRelay(ts.clients[0].SK, *relayRequest)
		require.Nil(t, err)
		_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: res.Providers[0].Address, Relays: []*types.RelaySession{relayRequest}})
		require.Nil(t, err)

		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}

	res, err := ts.keepers.Pairing.GetPairing(ts.ctx, &types.QueryGetPairingRequest{
		ChainID: ts.spec.Index,
		Client:  ts.clients[0].Addr.String(),
	})
	require.Nil(t, err)
	firstPairing := res.Providers

	// advance an epoch block by block. On each one try to spend more than it's allowed and check the pairing hasn't changed
	epochBlocks := ts.keepers.Epochstorage.EpochBlocksRaw(sdk.UnwrapSDKContext(ts.ctx))
	for i := 0; i < int(epochBlocks)-1; i++ {
		ts.ctx = testkeeper.AdvanceBlock(ts.ctx, ts.keepers)

		res, err := ts.keepers.Pairing.GetPairing(ts.ctx, &types.QueryGetPairingRequest{
			ChainID: ts.spec.Index,
			Client:  ts.clients[0].Addr.String(),
		})
		require.Nil(t, err)

		cuSum := ts.plan.PlanPolicy.GetEpochCuLimit()
		relayRequest := common.BuildRelayRequest(ts.ctx, res.Providers[0].Address, []byte(ts.spec.Apis[0].Name), cuSum, ts.spec.Name, nil)
		relayRequest.SessionId = uint64(i)
		relayRequest.Sig, err = sigs.SignRelay(ts.clients[0].SK, *relayRequest)
		require.Nil(t, err)
		_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: res.Providers[0].Address, Relays: []*types.RelaySession{relayRequest}})
		require.NotNil(t, err)

		require.Equal(t, firstPairing, res.Providers)
	}
}

func TestAddProjectAfterPlanUpdate(t *testing.T) {
	ts := setupForPaymentTest(t)
	err := ts.addClient(1)
	require.Nil(t, err)

	// advance epoch to get pairing
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	_, err = ts.servers.SubscriptionServer.Buy(ts.ctx, &subtypes.MsgBuy{
		Creator:  ts.clients[0].Addr.String(),
		Consumer: ts.clients[0].Addr.String(),
		Index:    ts.plan.Index,
		Duration: 11,
		Vrfpk:    "",
	})
	require.Nil(t, err)

	sub, found := ts.keepers.Subscription.GetSubscription(sdk.UnwrapSDKContext(ts.ctx), ts.clients[0].Addr.String())
	require.True(t, found)

	// advance epoch so the plan edit will be on a different block than the subscription purchase
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// edit the plan the subscription purchased (allow less CU per epoch)
	subPlan, found := ts.keepers.Plans.FindPlan(sdk.UnwrapSDKContext(ts.ctx), sub.PlanIndex, sub.PlanBlock)
	require.True(t, found)
	oldEpochCuLimit := subPlan.PlanPolicy.EpochCuLimit
	subPlan.PlanPolicy.EpochCuLimit -= 50
	err = ts.keepers.Plans.AddPlan(sdk.UnwrapSDKContext(ts.ctx), subPlan)
	require.Nil(t, err)

	// add another project under the subcscription
	projectData := projectstypes.ProjectData{
		Name:        "anotherProject",
		Description: "dummyDesc",
		Enabled:     true,
		ProjectKeys: []projectstypes.ProjectKey{
			{
				Key: ts.clients[1].Addr.String(),
				Types: []projectstypes.ProjectKey_KEY_TYPE{
					projectstypes.ProjectKey_DEVELOPER,
					projectstypes.ProjectKey_ADMIN,
				},
				Vrfpk: "",
			},
		},
		Policy: nil,
	}
	err = ts.keepers.Subscription.AddProjectToSubscription(sdk.UnwrapSDKContext(ts.ctx), ts.clients[0].Addr.String(), projectData)
	require.Nil(t, err)

	proj, _, err := ts.keepers.Projects.GetProjectForDeveloper(sdk.UnwrapSDKContext(ts.ctx), ts.clients[1].Addr.String(),
		uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight()))
	require.Nil(t, err)

	// set a new policy to the second project, making it more strict than the old plan but less strict than the new plan
	adminPolicy := ts.plan.PlanPolicy
	adminPolicy.EpochCuLimit = oldEpochCuLimit - 30

	err = ts.keepers.Projects.SetPolicy(sdk.UnwrapSDKContext(ts.ctx), []string{proj.Index}, &adminPolicy,
		ts.clients[1].Addr.String(), projectstypes.SET_ADMIN_POLICY)
	require.Nil(t, err)

	// advance epoch to set the new policy
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	_, _, _, cuPerEpochLimit, _, _, err := ts.keepers.Pairing.ValidatePairingForClient(sdk.UnwrapSDKContext(ts.ctx),
		ts.spec.Index, ts.clients[1].Addr, ts.providers[0].Addr, uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight()))
	require.Nil(t, err)

	// in terms of strictness: newPlan < adminPolicy < oldPlan but newPlan should not apply to the second project (since it's under a subscription that uses the old plan)
	require.Equal(t, adminPolicy.EpochCuLimit, cuPerEpochLimit)
}
