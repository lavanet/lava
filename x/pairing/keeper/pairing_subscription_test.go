package keeper_test

import (
	"testing"

	"github.com/lavanet/lava/testutil/common"
	"github.com/lavanet/lava/utils/slices"
	planstypes "github.com/lavanet/lava/x/plans/types"
	projectstypes "github.com/lavanet/lava/x/projects/types"
	"github.com/stretchr/testify/require"
)

func TestGetPairingForSubscription(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(0, 0, 1)    // 0 sub, 0 adm, 1 dev
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	_, client1Addr := ts.GetAccount(common.CONSUMER, 0)
	_, dev1Addr := ts.Account("dev1")

	ts.AdvanceEpoch()

	projectData := projectstypes.ProjectData{
		Name:        "project",
		Enabled:     true,
		Policy:      &ts.plan.PlanPolicy,
		ProjectKeys: slices.Slice(projectstypes.ProjectDeveloperKey(dev1Addr)),
	}
	err := ts.TxSubscriptionAddProject(client1Addr, projectData)
	require.Nil(t, err)

	ts.AdvanceEpoch()

	pairing, err := ts.QueryPairingGetPairing(ts.spec.Index, dev1Addr)
	require.Nil(t, err)
	providerAddr := pairing.Providers[0].Address

	verify, err := ts.QueryPairingVerifyPairing(ts.spec.Index, dev1Addr, providerAddr, ts.BlockHeight())
	require.Nil(t, err)
	require.True(t, verify.Valid)

	err = ts.TxSubscriptionDelProject(client1Addr, "project")
	require.Nil(t, err)

	ts.AdvanceEpoch()

	_, err = ts.QueryPairingGetPairing(ts.spec.Index, dev1Addr)
	require.NotNil(t, err)

	verify, err = ts.QueryPairingVerifyPairing(ts.spec.Index, dev1Addr, providerAddr, ts.BlockHeight())
	require.NotNil(t, err)
	require.False(t, verify.Valid)
}

func TestRelayPaymentSubscription(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, client1Addr := ts.GetAccount(common.CONSUMER, 0)

	ts.AdvanceEpoch()

	pairing, err := ts.QueryPairingGetPairing(ts.spec.Index, client1Addr)
	require.Nil(t, err)
	providerAddr := pairing.Providers[0].Address

	verify, err := ts.QueryPairingVerifyPairing(ts.spec.Index, client1Addr, providerAddr, ts.BlockHeight())
	require.Nil(t, err)
	require.True(t, verify.Valid)

	proj, err := ts.QueryProjectDeveloper(client1Addr)
	require.Nil(t, err)

	sub, err := ts.QuerySubscriptionCurrent(proj.Project.Subscription)
	require.Nil(t, err)
	require.NotNil(t, sub.Sub)

	policies := slices.Slice(
		proj.Project.AdminPolicy,
		proj.Project.SubscriptionPolicy,
		&ts.plan.PlanPolicy,
	)

	allowedCu, _ := ts.Keepers.Pairing.CalculateEffectiveAllowedCuPerEpochFromPolicies(
		policies, proj.Project.UsedCu, sub.Sub.MonthCuLeft)

	tests := []struct {
		name  string
		cu    uint64
		valid bool
	}{
		{"happyflow", ts.spec.ApiCollections[0].Apis[0].ComputeUnits, true},
		{"epochCULimit", allowedCu + 1, false},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			relaySession := ts.newRelaySession(providerAddr, uint64(i), tt.cu, ts.BlockHeight(), 0)
			signRelaySession(relaySession, client1Acct.SK)
			_, err = ts.TxPairingRelayPayment(providerAddr, relaySession)
			require.Equal(t, tt.valid, err == nil,
				"results incorrect for usage of %d err == nil: %t", tt.cu, err == nil)
		})
	}
}

func TestRelayPaymentSubscriptionCU(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(0, 0, 1)    // 0 sub, 0 adm, 1 dev
	ts.setupForPayments(1, 1, 0) // 1 provider, 2 client, default providers-to-pair

	_, providerAddr := ts.GetAccount(common.PROVIDER, 0)
	client1Acct, client1Addr := ts.GetAccount(common.CONSUMER, 0)
	_, dev1Addr := ts.Account("dev1")

	consumers := slices.Slice(client1Addr, dev1Addr)

	projectData := projectstypes.ProjectData{
		Name:    "proj_b",
		Enabled: true,
		ProjectKeys: []projectstypes.ProjectKey{
			projectstypes.NewProjectKey(dev1Addr).
				AddType(projectstypes.ProjectKey_ADMIN).
				AddType(projectstypes.ProjectKey_DEVELOPER),
		},
		Policy: &ts.plan.PlanPolicy,
	}
	err := ts.TxSubscriptionAddProject(client1Addr, projectData)
	require.Nil(t, err)

	ts.AdvanceEpoch()

	// verify both projects exist
	projA, err := ts.QueryProjectDeveloper(client1Addr)
	require.Nil(t, err)
	projB, err := ts.QueryProjectDeveloper(dev1Addr)
	require.Nil(t, err)

	// verify that both consumers are paired
	for _, consumer := range consumers {
		pairing, err := ts.QueryPairingGetPairing(ts.spec.Index, consumer)
		require.Nil(t, err)
		providerAddr := pairing.Providers[0].Address
		verify, err := ts.QueryPairingVerifyPairing(ts.spec.Index, consumer, providerAddr, ts.BlockHeight())
		require.Nil(t, err)
		require.True(t, verify.Valid)
	}

	// both projects have adminPolicy, subscriptionPolicy = nil -> they go by the plan policy
	// waste all the subscription's CU

	totalCuLimit := ts.plan.PlanPolicy.TotalCuLimit
	epochCuLimit := ts.plan.PlanPolicy.EpochCuLimit

	i := 0
	for ; uint64(i) < totalCuLimit/epochCuLimit; i++ {
		relaySession := ts.newRelaySession(providerAddr, uint64(i), epochCuLimit, ts.BlockHeight(), 0)
		signRelaySession(relaySession, client1Acct.SK)

		_, err = ts.TxPairingRelayPayment(providerAddr, relaySession)
		require.Nil(t, err)

		ts.AdvanceEpoch()
	}

	// last iteration should finish the plan and subscription quota
	relaySession := ts.newRelaySession(providerAddr, uint64(i+1), epochCuLimit, ts.BlockHeight(), uint64(i+1))
	signRelaySession(relaySession, client1Acct.SK)
	_, err = ts.TxPairingRelayPayment(providerAddr, relaySession)
	require.NotNil(t, err)

	// verify that project A wasted all of the subscription's CU
	sub, err := ts.QuerySubscriptionCurrent(projA.Project.Subscription)
	require.Nil(t, err)
	require.NotNil(t, sub.Sub)
	require.Equal(t, uint64(0), sub.Sub.MonthCuLeft)
	projA, err = ts.QueryProjectDeveloper(client1Addr)
	require.Nil(t, err)
	require.Equal(t, sub.Sub.MonthCuTotal, projA.Project.UsedCu)
	require.Equal(t, uint64(0), projB.Project.UsedCu)

	// try to use CU on projB. Should fail because A wasted it all
	relaySession.SessionId += 1
	signRelaySession(relaySession, client1Acct.SK)
	_, err = ts.TxPairingRelayPayment(providerAddr, relaySession)
	require.NotNil(t, err)
}

func TestStrictestPolicyGeolocation(t *testing.T) {
	ts := newTester(t)

	// make the plan policy's geolocation 7(=111)
	// (done before setupForPayments() below so subscription will use ths plan)
	// will overwrite the default "mock" plan
	ts.plan.PlanPolicy.GeolocationProfile = 7
	ts.AddPlan("mock", ts.plan)

	ts.setupForPayments(1, 1, 0) // 1 provider, 0 client, default providers-to-pair

	_, client1Addr := ts.GetAccount(common.CONSUMER, 0)

	proj, err := ts.QueryProjectDeveloper(client1Addr)
	require.Nil(t, err)
	projectID := proj.Project.Index

	ts.AdvanceEpoch()

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
			adminPolicy := &planstypes.Policy{
				GeolocationProfile: tt.geolocationAdminPolicy,
				MaxProvidersToPair: 2,
			}
			subscriptionPolicy := &planstypes.Policy{
				GeolocationProfile: tt.geolocationSubPolicy,
				MaxProvidersToPair: 2,
			}

			_, err = ts.TxProjectSetPolicy(projectID, client1Addr, *adminPolicy)
			require.Nil(t, err)

			ts.AdvanceEpoch()

			_, err = ts.TxProjectSetSubscriptionPolicy(projectID, client1Addr, *subscriptionPolicy)
			require.Nil(t, err)

			ts.AdvanceEpoch()

			// the only provider is set with geolocation=1. So only geolocation that ANDs
			// with 1 and output non-zero result, will output a provider for pairing
			res, err := ts.QueryPairingGetPairing(ts.spec.Index, client1Addr)
			require.Nil(t, err)
			if tt.success {
				require.NotEqual(t, 0, len(res.Providers))
			} else {
				require.Equal(t, 0, len(res.Providers))
			}
		})
	}
}

func TestStrictestPolicyProvidersToPair(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(6, 1, 0) // 6 provider, 1 client, default providers-to-pair

	_, client1Addr := ts.GetAccount(common.CONSUMER, 0)

	res, err := ts.QueryProjectDeveloper(client1Addr)
	require.Nil(t, err)

	proj := res.Project
	ts.AdvanceEpoch()

	providersToPairTestTemplates := []struct {
		name                       string
		providersToPairAdminPolicy uint64
		providersToPairSubPolicy   uint64
		effectiveProvidersToPair   int
		adminPolicyValid           bool
		subscriptionPolicyValid    bool
	}{
		{"effective providersToPair = 2", uint64(4), uint64(2), 2, true, true},
		{"admin policy providersToPair = 1", uint64(1), uint64(3), 3, false, true},
		{"sub policy providersToPair = 1", uint64(3), uint64(1), 3, true, false},
	}

	for _, tt := range providersToPairTestTemplates {
		t.Run(tt.name, func(t *testing.T) {
			adminPolicy := &planstypes.Policy{
				GeolocationProfile: 1,
				MaxProvidersToPair: tt.providersToPairAdminPolicy,
			}
			subscriptionPolicy := &planstypes.Policy{
				GeolocationProfile: 1,
				MaxProvidersToPair: tt.providersToPairSubPolicy,
			}

			_, err = ts.TxProjectSetPolicy(proj.Index, client1Addr, *adminPolicy)
			if !tt.adminPolicyValid {
				require.NotNil(t, err)
				return
			}
			// else
			require.Nil(t, err)

			ts.AdvanceEpoch()

			_, err = ts.TxProjectSetSubscriptionPolicy(proj.Index, client1Addr, *subscriptionPolicy)
			if !tt.subscriptionPolicyValid {
				require.NotNil(t, err)
				return
			}
			// else
			require.Nil(t, err)

			ts.AdvanceEpoch()

			res, err := ts.QueryPairingGetPairing(ts.spec.Index, client1Addr)
			require.Nil(t, err)
			require.Equal(t, tt.effectiveProvidersToPair, len(res.Providers))
		})
	}
}

func TestStrictestPolicyCuPerEpoch(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, client1Addr := ts.GetAccount(common.CONSUMER, 0)
	_, providerAddr := ts.GetAccount(common.PROVIDER, 0)

	res, err := ts.QueryProjectDeveloper(client1Addr)
	require.Nil(t, err)
	proj := res.Project

	ts.AdvanceEpoch()

	providersToPairTestTemplates := []struct {
		name                     string
		cuPerEpochAdminPolicy    uint64
		cuPerEpochSubPolicy      uint64
		useMostOfProjectCu       bool
		wasteSubscriptionCu      bool
		effectiveCuPerEpochLimit uint64
	}{
		{"admin policy min CU", ts.plan.PlanPolicy.EpochCuLimit - 10, ts.plan.PlanPolicy.EpochCuLimit + 10, false, false, ts.plan.PlanPolicy.EpochCuLimit - 10},
		{"sub policy min CU", ts.plan.PlanPolicy.EpochCuLimit + 10, ts.plan.PlanPolicy.EpochCuLimit - 10, false, false, ts.plan.PlanPolicy.EpochCuLimit - 10},
		{"use most of the project's CU", ts.plan.PlanPolicy.EpochCuLimit, ts.plan.PlanPolicy.EpochCuLimit, true, false, uint64(10)},
		{"waste subscription CU", ts.plan.PlanPolicy.EpochCuLimit, ts.plan.PlanPolicy.EpochCuLimit, false, true, uint64(0)},
	}

	for _, tt := range providersToPairTestTemplates {
		t.Run(tt.name, func(t *testing.T) {
			// add a new project to the subscription just to waste the subcsription's cu
			if tt.wasteSubscriptionCu {
				_, waste1Addr := ts.AddAccount("waste", 0, testBalance)

				projectData := projectstypes.ProjectData{
					Name:    "low_cu_project",
					Enabled: true,
					ProjectKeys: []projectstypes.ProjectKey{
						projectstypes.NewProjectKey(waste1Addr).
							AddType(projectstypes.ProjectKey_ADMIN).
							AddType(projectstypes.ProjectKey_DEVELOPER),
					},
					Policy: &ts.plan.PlanPolicy,
				}

				err = ts.TxSubscriptionAddProject(proj.Subscription, projectData)
				require.Nil(t, err)

				ts.AdvanceEpoch()

				sub, err := ts.QuerySubscriptionCurrent(proj.Subscription)
				require.Nil(t, err)
				require.NotNil(t, sub.Sub)

				relaySession := ts.newRelaySession(providerAddr, 100, sub.Sub.MonthCuLeft, ts.BlockHeight(), 0)
				signRelaySession(relaySession, client1Acct.SK)

				_, err = ts.TxPairingRelayPayment(providerAddr, relaySession)
				require.Nil(t, err)

				ts.AdvanceEpoch()
			}

			adminPolicy := &planstypes.Policy{
				GeolocationProfile: 1,
				EpochCuLimit:       tt.cuPerEpochAdminPolicy,
				TotalCuLimit:       ts.plan.PlanPolicy.TotalCuLimit,
				MaxProvidersToPair: ts.plan.PlanPolicy.MaxProvidersToPair,
			}
			subscriptionPolicy := &planstypes.Policy{
				GeolocationProfile: 1,
				EpochCuLimit:       tt.cuPerEpochSubPolicy,
				TotalCuLimit:       ts.plan.PlanPolicy.TotalCuLimit,
				MaxProvidersToPair: ts.plan.PlanPolicy.MaxProvidersToPair,
			}

			_, err = ts.TxProjectSetPolicy(proj.Index, client1Addr, *adminPolicy)
			require.Nil(t, err)

			ts.AdvanceEpoch()

			_, err = ts.TxProjectSetSubscriptionPolicy(proj.Index, client1Addr, *subscriptionPolicy)
			require.Nil(t, err)

			ts.AdvanceEpoch()

			totalCuLimit := ts.plan.PlanPolicy.TotalCuLimit
			epochCuLimit := ts.plan.PlanPolicy.EpochCuLimit

			// leave 10 CU in the project
			if tt.useMostOfProjectCu {
				for i := 0; uint64(i) < adminPolicy.TotalCuLimit/epochCuLimit; i++ {
					cuSum := epochCuLimit

					res, err := ts.QueryProjectDeveloper(client1Addr)
					require.Nil(t, err)
					proj := res.Project

					if totalCuLimit-proj.UsedCu <= cuSum {
						cuSum -= 10
					}

					relaySession := ts.newRelaySession(providerAddr, uint64(i), cuSum, ts.BlockHeight(), 0)
					signRelaySession(relaySession, client1Acct.SK)

					_, err = ts.TxPairingRelayPayment(providerAddr, relaySession)
					require.Nil(t, err)

					ts.AdvanceEpoch()
				}
			}

			verify, err := ts.QueryPairingVerifyPairing(ts.spec.Index, client1Addr, providerAddr, ts.BlockHeight())
			require.Nil(t, err)
			require.Equal(t, tt.effectiveCuPerEpochLimit, verify.CuPerEpoch)
		})
	}
}

func TestPairingNotChangingDueToCuOveruse(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(100, 1, 0) // 1 provider, 1 client, default providers-to-pair

	client1Acct, client1Addr := ts.GetAccount(common.CONSUMER, 0)

	// add 10 months to the subscription
	_, err := ts.TxSubscriptionBuy(client1Addr, client1Addr, ts.plan.Index, 10)
	require.Nil(t, err)

	totalCuLimit := ts.plan.PlanPolicy.TotalCuLimit
	epochCuLimit := ts.plan.PlanPolicy.EpochCuLimit

	for i := 0; i < int(totalCuLimit/epochCuLimit); i++ {
		res, err := ts.QueryPairingGetPairing(ts.spec.Index, client1Addr)
		require.Nil(t, err)
		providerAddr := res.Providers[0].Address

		relaySession := ts.newRelaySession(providerAddr, uint64(i), epochCuLimit, ts.BlockHeight(), 0)
		signRelaySession(relaySession, client1Acct.SK)

		_, err = ts.TxPairingRelayPayment(providerAddr, relaySession)
		require.Nil(t, err)

		ts.AdvanceEpoch()
	}

	res, err := ts.QueryPairingGetPairing(ts.spec.Index, client1Addr)
	require.Nil(t, err)
	firstPairing := res.Providers

	// advance an epoch block by block, while spending more than allowed: pairing must not change
	for i := 0; i < int(ts.EpochBlocks()-1); i++ {
		ts.AdvanceBlock()

		res, err := ts.QueryPairingGetPairing(ts.spec.Index, client1Addr)
		require.Nil(t, err)
		providerAddr := res.Providers[0].Address

		relaySession := ts.newRelaySession(providerAddr, uint64(i), epochCuLimit, ts.BlockHeight(), 0)
		signRelaySession(relaySession, client1Acct.SK)

		_, err = ts.TxPairingRelayPayment(providerAddr, relaySession)
		require.NotNil(t, err)

		require.Equal(t, firstPairing, res.Providers)
	}
}

func TestAddProjectAfterPlanUpdate(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(0, 0, 1)    // 0 sub, 0 adm, 1 dev
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	_, client1Addr := ts.GetAccount(common.CONSUMER, 0)
	_, providerAddr := ts.GetAccount(common.PROVIDER, 0)
	_, dev1Addr := ts.Account("dev1")

	oldEpochCuLimit := ts.plan.PlanPolicy.EpochCuLimit

	// edit the plan in subscription purchased (allow less CU per epoch)
	ts.plan.PlanPolicy.EpochCuLimit -= 50
	err := ts.TxProposalAddPlans(ts.plan)
	require.Nil(t, err)

	ts.AdvanceEpoch()

	// add another project under the subcscription
	projectData := projectstypes.ProjectData{
		Name:    "second",
		Enabled: true,
		ProjectKeys: []projectstypes.ProjectKey{
			projectstypes.NewProjectKey(dev1Addr).
				AddType(projectstypes.ProjectKey_ADMIN).
				AddType(projectstypes.ProjectKey_DEVELOPER),
		},
		Policy: nil,
	}

	err = ts.TxSubscriptionAddProject(client1Addr, projectData)
	require.Nil(t, err)

	ts.AdvanceEpoch()

	proj, err := ts.QueryProjectDeveloper(dev1Addr)
	require.Nil(t, err)

	// new policy to the second project: stricter than the old plan, weaker than the new plan
	adminPolicy := ts.plan.PlanPolicy
	adminPolicy.EpochCuLimit = oldEpochCuLimit - 30

	_, err = ts.TxProjectSetPolicy(proj.Project.Index, dev1Addr, adminPolicy)
	require.Nil(t, err)

	// advance epoch to set the new policy
	ts.AdvanceEpoch()

	verify, err := ts.QueryPairingVerifyPairing(ts.spec.Index, dev1Addr, providerAddr, ts.BlockHeight())
	require.Nil(t, err)

	// in terms of strictness: newPlan < adminPolicy < oldPlan, but newPlan should not apply
	// to the second project (since it's under a subscription that uses the old plan)
	require.Equal(t, adminPolicy.EpochCuLimit, verify.CuPerEpoch)
}
