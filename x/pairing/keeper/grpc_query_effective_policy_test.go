package keeper_test

import (
	"testing"

	"github.com/lavanet/lava/v2/testutil/common"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
	"github.com/stretchr/testify/require"
)

// Test effective policy query
func TestEffectivePolicy(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 client, 1 provider

	_, clientAddr := ts.GetAccount(common.CONSUMER, 0)

	project, err := ts.GetProjectDeveloperData(clientAddr, ts.BlockHeight())
	require.NoError(t, err)

	// the auto-created project from setupForPayments is subject to the mock plan policy
	// the mock policy assumed in this test is (implemented by common.CreateMockPolicy()):
	//
	// 		plantypes.Policy{
	// 			TotalCuLimit:       100000,
	// 			EpochCuLimit:       10000,
	// 			MaxProvidersToPair: 3,
	// 			GeolocationProfile: 1,
	// 		}
	//
	// when testing the effective policy, this will be the policy in mind

	adminPolicy := planstypes.Policy{
		TotalCuLimit:       99000,
		EpochCuLimit:       10000,
		MaxProvidersToPair: 3,
		GeolocationProfile: 1,
	}
	_, err = ts.TxProjectSetPolicy(project.ProjectID, clientAddr, &adminPolicy)
	require.NoError(t, err)

	subPolicy := planstypes.Policy{
		TotalCuLimit:       100000,
		EpochCuLimit:       9900,
		MaxProvidersToPair: 2,
		GeolocationProfile: planstypes.Geolocation_value["GL"],
	}
	_, err = ts.TxProjectSetSubscriptionPolicy(project.ProjectID, clientAddr, &subPolicy)
	require.NoError(t, err)

	// the effective policy function calcaulates the effective chain policy within it
	// if there is no chain policy in any of the policies, it makes one using this function
	chainPolicy, allowed := planstypes.GetStrictestChainPolicyForSpec(ts.spec.Index, []*planstypes.Policy{&adminPolicy, &subPolicy, &ts.plan.PlanPolicy})
	require.True(t, allowed)

	expectedEffectivePolicy := planstypes.Policy{
		ChainPolicies:      []planstypes.ChainPolicy{chainPolicy},
		TotalCuLimit:       99000,
		EpochCuLimit:       9900,
		MaxProvidersToPair: 2,
		GeolocationProfile: 1,
	}

	// apply the policy changes
	ts.AdvanceEpoch()

	res, err := ts.QueryPairingEffectivePolicy(ts.spec.Index, clientAddr)
	require.NoError(t, err)
	require.True(t, expectedEffectivePolicy.Equal(res.Policy))
	require.Nil(t, res.PendingPolicy) // there should be no pending policy

	// set a new policy without applying it (no advanceEpoch)
	_, err = ts.TxProjectSetPolicy(project.ProjectID, clientAddr, &planstypes.Policy{
		TotalCuLimit:       80000,
		EpochCuLimit:       5000,
		MaxProvidersToPair: 3,
		GeolocationProfile: 1,
	})
	require.NoError(t, err)
	res, err = ts.QueryPairingEffectivePolicy(ts.spec.Index, clientAddr)
	require.NoError(t, err)
	require.True(t, expectedEffectivePolicy.Equal(res.Policy)) // policy should still be the original one

	// there should be a new pending policy
	require.True(t, res.PendingPolicy.Equal(planstypes.Policy{
		ChainPolicies:      expectedEffectivePolicy.ChainPolicies,
		TotalCuLimit:       80000,
		EpochCuLimit:       5000,
		MaxProvidersToPair: 2,
		GeolocationProfile: 1,
	}))
}
