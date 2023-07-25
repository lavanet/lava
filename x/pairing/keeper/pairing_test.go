package keeper_test

import (
	"math"
	"testing"

	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/utils/slices"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

func TestPairingUniqueness(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(2, 0, 0) // 2 sub, 0 adm, 0 dev

	var balance int64 = 10000
	stake := balance / 10

	_, sub1Addr := ts.Account("sub1")
	_, sub2Addr := ts.Account("sub2")

	_, err := ts.TxSubscriptionBuy(sub1Addr, sub1Addr, ts.plan.Index, 1)
	require.Nil(t, err)
	_, err = ts.TxSubscriptionBuy(sub2Addr, sub2Addr, ts.plan.Index, 1)
	require.Nil(t, err)

	for i := 1; i <= 1000; i++ {
		_, addr := ts.AddAccount(common.PROVIDER, i, balance)
		err := ts.StakeProvider(addr, ts.spec, stake)
		require.Nil(t, err)
	}

	ts.AdvanceEpoch()

	// test that 2 different clients get different pairings
	pairing1, err := ts.QueryPairingGetPairing(ts.spec.Index, sub1Addr)
	require.Nil(t, err)
	pairing2, err := ts.QueryPairingGetPairing(ts.spec.Index, sub2Addr)
	require.Nil(t, err)

	filter := func(p epochstoragetypes.StakeEntry) string { return p.Address }

	providerAddrs1 := slices.Filter(pairing1.Providers, filter)
	providerAddrs2 := slices.Filter(pairing2.Providers, filter)

	require.Equal(t, len(pairing1.Providers), len(pairing2.Providers))
	require.False(t, slices.UnorderedEqual(providerAddrs1, providerAddrs2))

	ts.AdvanceEpoch()

	// test that in different epoch we get different pairings for consumer1
	pairing11, err := ts.QueryPairingGetPairing(ts.spec.Index, sub1Addr)
	require.Nil(t, err)

	providerAddrs11 := slices.Filter(pairing11.Providers, filter)

	require.Equal(t, len(pairing1.Providers), len(pairing11.Providers))
	require.False(t, slices.UnorderedEqual(providerAddrs1, providerAddrs11))

	// test that get pairing gives the same results for the whole epoch
	epochBlocks := ts.EpochBlocks()
	for i := uint64(0); i < epochBlocks-1; i++ {
		ts.AdvanceBlock()

		pairing111, err := ts.QueryPairingGetPairing(ts.spec.Index, sub1Addr)
		require.Nil(t, err)

		for i := range pairing11.Providers {
			providerAddr := pairing11.Providers[i].Address
			require.Equal(t, providerAddr, pairing111.Providers[i].Address)
			require.Nil(t, err)
			verify, err := ts.QueryPairingVerifyPairing(ts.spec.Index, sub1Addr, providerAddr, ts.BlockHeight())
			require.Nil(t, err)
			require.True(t, verify.Valid)
		}
	}
}

func TestValidatePairingDeterminism(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 0) // 1 sub, 0 adm, 0 dev

	var balance int64 = 10000
	stake := balance / 10

	_, sub1Addr := ts.Account("sub1")

	_, err := ts.TxSubscriptionBuy(sub1Addr, sub1Addr, ts.plan.Index, 1)
	require.Nil(t, err)

	for i := 1; i <= 10; i++ {
		_, addr := ts.AddAccount(common.PROVIDER, i, balance)
		err := ts.StakeProvider(addr, ts.spec, stake)
		require.Nil(t, err)
	}

	ts.AdvanceEpoch()

	// test that 2 different clients get different pairings
	pairing, err := ts.QueryPairingGetPairing(ts.spec.Index, sub1Addr)
	require.Nil(t, err)

	block := ts.BlockHeight()
	testAllProviders := func() {
		for _, provider := range pairing.Providers {
			providerAddr := provider.Address
			verify, err := ts.QueryPairingVerifyPairing(ts.spec.Index, sub1Addr, providerAddr, block)
			require.Nil(t, err)
			require.True(t, verify.Valid)
		}
	}

	count := ts.BlocksToSave() - ts.BlockHeight()
	for i := 0; i < int(count); i++ {
		ts.AdvanceBlock()
		testAllProviders()
	}
}

func TestGetPairing(t *testing.T) {
	ts := newTester(t)

	// do not use ts.setupForPayments(1, 1, 0), because it kicks off with AdvanceEpoch()
	// (for the benefit of users) but the "zeroEpoch" test below expects to start at the
	// same epoch of staking the providers.
	ts.addClient(1)
	ts.addProvider(1)

	// BLOCK_TIME = 30sec (testutil/keeper/keepers_init.go)
	constBlockTime := testkeeper.BLOCK_TIME
	epochBlocks := ts.EpochBlocks()

	// test: different epoch, valid tells if the payment request should work
	tests := []struct {
		name                string
		validPairingExists  bool
		isEpochTimesChanged bool
	}{
		{"zeroEpoch", false, false},
		{"firstEpoch", true, false},
		{"commonEpoch", true, false},
		{"epochTimesChanged", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Advance an epoch according to the test
			switch tt.name {
			case "zeroEpoch":
				// do nothing
			case "firstEpoch":
				ts.AdvanceEpoch()
			case "commonEpoch":
				ts.AdvanceEpochs(5)
			case "epochTimesChanged":
				ts.AdvanceEpochs(5)
				ts.AdvanceBlocks(epochBlocks/2, constBlockTime/2)
				ts.AdvanceBlocks(epochBlocks / 2)
			}

			_, clientAddr := ts.GetAccount(common.CONSUMER, 0)
			_, providerAddr := ts.GetAccount(common.PROVIDER, 0)

			// get pairing for client (for epoch zero expect to fail)
			pairing, err := ts.QueryPairingGetPairing(ts.spec.Index, clientAddr)
			if !tt.validPairingExists {
				require.NotNil(t, err)
			} else {
				require.Nil(t, err)

				// verify the expected provider
				require.Equal(t, providerAddr, pairing.Providers[0].Address)

				// verify the current epoch
				epochThis := ts.EpochStart()
				require.Equal(t, epochThis, pairing.CurrentEpoch)

				// verify the SpecLastUpdatedBlock
				require.Equal(t, ts.spec.BlockLastUpdated, pairing.SpecLastUpdatedBlock)

				// if prevEpoch == 0 -> averageBlockTime = 0
				// else calculate the time (like the actual get-pairing function)
				epochPrev, err := ts.Keepers.Epochstorage.GetPreviousEpochStartForBlock(ts.Ctx, epochThis)
				require.Nil(t, err)

				var averageBlockTime uint64
				if epochPrev != 0 {
					// calculate average block time base on total time from first block of
					// previous epoch until first block of this epoch and block dfference.
					blockCore1 := ts.Keepers.BlockStore.LoadBlock(int64(epochPrev))
					blockCore2 := ts.Keepers.BlockStore.LoadBlock(int64(epochThis))
					delta := blockCore2.Time.Sub(blockCore1.Time).Seconds()
					averageBlockTime = uint64(delta) / (epochThis - epochPrev)
				}

				overlapBlocks := ts.Keepers.Pairing.EpochBlocksOverlap(ts.Ctx)
				nextEpochStart, err := ts.Keepers.Epochstorage.GetNextEpoch(ts.Ctx, epochThis)
				require.Nil(t, err)

				// calculate the block in which the next pairing will happen (+overlap)
				nextPairingBlock := nextEpochStart + overlapBlocks
				// calculate number of blocks from the current block to the next epoch
				blocksUntilNewEpoch := nextPairingBlock - ts.BlockHeight()
				// calculate time left for the next pairing (blocks left* avg block time)
				timeLeftToNextPairing := blocksUntilNewEpoch * averageBlockTime

				if !tt.isEpochTimesChanged {
					require.Equal(t, timeLeftToNextPairing, pairing.TimeLeftToNextPairing)
				} else {
					// averageBlockTime in get-pairing query -> minimal average across sampled epoch
					// averageBlockTime in this test -> normal average across epoch
					// we've used a smaller blocktime some of the time -> averageBlockTime from
					// get-pairing is smaller than the averageBlockTime calculated in this test
					require.Less(t, pairing.TimeLeftToNextPairing, timeLeftToNextPairing)
				}

				// verify nextPairingBlock
				require.Equal(t, nextPairingBlock, pairing.BlockOfNextPairing)
			}
		})
	}
}

func TestPairingStatic(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(1, 0, 0) // 1 sub, 0 adm, 0 dev

	_, sub1Addr := ts.Account("sub1")

	ts.spec.ProvidersTypes = spectypes.Spec_static
	// will overwrite the default "mock" spec
	// (no TxProposalAddSpecs because the mock spec does not pass validaton)
	ts.AddSpec("mock", ts.spec)

	ts.AdvanceEpoch()

	_, err := ts.TxSubscriptionBuy(sub1Addr, sub1Addr, ts.plan.Index, 1)
	require.Nil(t, err)

	for i := 0; i < int(ts.plan.PlanPolicy.MaxProvidersToPair)*2; i++ {
		_, addr := ts.AddAccount(common.PROVIDER, i, testBalance)
		err := ts.StakeProvider(addr, ts.spec, testStake+int64(i))
		require.Nil(t, err)
	}

	// we expect to get all the providers in static spec

	ts.AdvanceEpoch()

	pairing, err := ts.QueryPairingGetPairing(ts.spec.Index, sub1Addr)
	require.Nil(t, err)

	for i, provider := range pairing.Providers {
		require.Equal(t, provider.Stake.Amount.Int64(), testStake+int64(i))
	}
}

func TestAddonPairing(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 0, 0) // 1 provider, 0 client, default providers-to-pair

	mandatory := spectypes.CollectionData{
		ApiInterface: "mandatory",
		InternalPath: "",
		Type:         "",
		AddOn:        "",
	}
	mandatoryAddon := spectypes.CollectionData{
		ApiInterface: "mandatory",
		InternalPath: "",
		Type:         "",
		AddOn:        "addon",
	}
	optional := spectypes.CollectionData{
		ApiInterface: "optional",
		InternalPath: "",
		Type:         "",
		AddOn:        "optional",
	}
	ts.spec.ApiCollections = []*spectypes.ApiCollection{
		{
			Enabled:        true,
			CollectionData: mandatory,
		},
		{
			Enabled:        true,
			CollectionData: optional,
		},
		{
			Enabled:        true,
			CollectionData: mandatoryAddon,
		},
	}

	// will overwrite the default "mock" spec
	ts.AddSpec("mock", ts.spec)
	specId := ts.spec.Index

	mandatoryChainPolicy := &planstypes.ChainPolicy{
		ChainId:     specId,
		Collections: []spectypes.CollectionData{mandatory},
	}
	mandatoryAddonChainPolicy := &planstypes.ChainPolicy{
		ChainId:     specId,
		Collections: []spectypes.CollectionData{mandatoryAddon},
	}
	optionalAddonChainPolicy := &planstypes.ChainPolicy{
		ChainId:     specId,
		Collections: []spectypes.CollectionData{optional},
	}
	optionalAndMandatoryAddonChainPolicy := &planstypes.ChainPolicy{
		ChainId:     specId,
		Collections: []spectypes.CollectionData{mandatoryAddon, optional},
	}

	templates := []struct {
		name                      string
		planChainPolicy           *planstypes.ChainPolicy
		subscChainPolicy          *planstypes.ChainPolicy
		projChainPolicy           *planstypes.ChainPolicy
		expectedProviders         int
		expectedStrictestPolicies []string
	}{
		{
			name:              "empty",
			expectedProviders: 12,
		},
		{
			name:              "mandatory in plan",
			planChainPolicy:   mandatoryChainPolicy,
			expectedProviders: 12, // stub provider also gets picked
		},
		{
			name:              "mandatory in subsc",
			subscChainPolicy:  mandatoryChainPolicy,
			projChainPolicy:   nil,
			expectedProviders: 12, // stub provider also gets picked
		},
		{
			name:              "mandatory in proj",
			projChainPolicy:   mandatoryChainPolicy,
			expectedProviders: 12, // stub provider also gets picked
		},
		{
			name:                      "addon in plan",
			planChainPolicy:           mandatoryAddonChainPolicy,
			subscChainPolicy:          nil,
			projChainPolicy:           nil,
			expectedProviders:         6,
			expectedStrictestPolicies: []string{"addon"},
		},
		{
			name:                      "addon in subsc",
			subscChainPolicy:          mandatoryAddonChainPolicy,
			expectedProviders:         6,
			expectedStrictestPolicies: []string{"addon"},
		},
		{
			name:                      "addon in proj",
			projChainPolicy:           mandatoryAddonChainPolicy,
			expectedProviders:         6,
			expectedStrictestPolicies: []string{"addon"},
		},
		{
			name:                      "optional in plan",
			planChainPolicy:           optionalAddonChainPolicy,
			expectedProviders:         7,
			expectedStrictestPolicies: []string{"optional"},
		},
		{
			name:                      "optional in subsc",
			subscChainPolicy:          optionalAddonChainPolicy,
			expectedProviders:         7,
			expectedStrictestPolicies: []string{"optional"},
		},
		{
			name:                      "optional in proj",
			projChainPolicy:           optionalAddonChainPolicy,
			expectedProviders:         7,
			expectedStrictestPolicies: []string{"optional"},
		},
		{
			name:                      "optional and addon in plan",
			planChainPolicy:           optionalAndMandatoryAddonChainPolicy,
			expectedProviders:         4,
			expectedStrictestPolicies: []string{"optional", "addon"},
		},
		{
			name:                      "optional and addon in subsc",
			subscChainPolicy:          optionalAndMandatoryAddonChainPolicy,
			expectedProviders:         4,
			expectedStrictestPolicies: []string{"optional", "addon"},
		},
		{
			name:                      "optional and addon in proj",
			projChainPolicy:           optionalAndMandatoryAddonChainPolicy,
			expectedProviders:         4,
			expectedStrictestPolicies: []string{"optional", "addon"},
		},
		{
			name:                      "optional and addon in plan, addon in subsc",
			planChainPolicy:           optionalAndMandatoryAddonChainPolicy,
			subscChainPolicy:          mandatoryAddonChainPolicy,
			expectedProviders:         4,
			expectedStrictestPolicies: []string{"optional", "addon"},
		},
		{
			name:                      "optional and addon in subsc, addon in plan",
			planChainPolicy:           mandatoryAddonChainPolicy,
			subscChainPolicy:          optionalAndMandatoryAddonChainPolicy,
			expectedProviders:         4,
			expectedStrictestPolicies: []string{"optional", "addon"},
		},
		{
			name:                      "optional and addon in subsc, addon in proj",
			subscChainPolicy:          optionalAndMandatoryAddonChainPolicy,
			projChainPolicy:           mandatoryAddonChainPolicy,
			expectedProviders:         4,
			expectedStrictestPolicies: []string{"optional", "addon"},
		},
		{
			name:                      "optional in subsc, addon in proj",
			subscChainPolicy:          optionalAndMandatoryAddonChainPolicy,
			projChainPolicy:           mandatoryAddonChainPolicy,
			expectedProviders:         4,
			expectedStrictestPolicies: []string{"optional", "addon"},
		},
	}

	mandatorySupportingEndpoints := []epochstoragetypes.Endpoint{{
		IPPORT:        "123",
		Geolocation:   1,
		Addons:        []string{mandatory.AddOn},
		ApiInterfaces: []string{mandatory.ApiInterface},
	}}
	mandatoryAddonSupportingEndpoints := []epochstoragetypes.Endpoint{{
		IPPORT:        "123",
		Geolocation:   1,
		Addons:        []string{mandatoryAddon.AddOn},
		ApiInterfaces: []string{mandatoryAddon.ApiInterface},
	}}
	mandatoryAndMandatoryAddonSupportingEndpoints := slices.Concat(
		mandatorySupportingEndpoints, mandatoryAddonSupportingEndpoints)

	optionalSupportingEndpoints := []epochstoragetypes.Endpoint{{
		IPPORT:        "123",
		Geolocation:   1,
		Addons:        []string{optional.AddOn},
		ApiInterfaces: []string{optional.ApiInterface},
	}}
	optionalAndMandatorySupportingEndpoints := slices.Concat(
		mandatorySupportingEndpoints, optionalSupportingEndpoints)
	optionalAndMandatoryAddonSupportingEndpoints := slices.Concat(
		mandatoryAddonSupportingEndpoints, optionalSupportingEndpoints)

	allSupportingEndpoints := slices.Concat(
		mandatorySupportingEndpoints, optionalAndMandatoryAddonSupportingEndpoints)

	mandatoryAndOptionalSingleEndpoint := []epochstoragetypes.Endpoint{{
		IPPORT:        "123",
		Geolocation:   1,
		Addons:        []string{},
		ApiInterfaces: []string{mandatoryAddon.ApiInterface, optional.ApiInterface},
	}}

	err := ts.addProviderEndpoints(2, mandatorySupportingEndpoints)
	require.NoError(t, err)
	err = ts.addProviderEndpoints(2, mandatoryAndMandatoryAddonSupportingEndpoints)
	require.NoError(t, err)
	err = ts.addProviderEndpoints(2, optionalAndMandatorySupportingEndpoints)
	require.NoError(t, err)
	err = ts.addProviderEndpoints(1, mandatoryAndOptionalSingleEndpoint)
	require.NoError(t, err)
	err = ts.addProviderEndpoints(2, optionalAndMandatoryAddonSupportingEndpoints)
	require.NoError(t, err)
	err = ts.addProviderEndpoints(2, allSupportingEndpoints)
	require.NoError(t, err)
	require.NoError(t, err)
	// total 11 providers

	err = ts.addProviderEndpoints(2, optionalSupportingEndpoints)
	require.Error(t, err)

	for _, tt := range templates {
		t.Run(tt.name, func(t *testing.T) {
			defaultPolicy := func() planstypes.Policy {
				return planstypes.Policy{
					ChainPolicies:      []planstypes.ChainPolicy{},
					GeolocationProfile: math.MaxUint64,
					MaxProvidersToPair: 12,
					TotalCuLimit:       math.MaxUint64,
					EpochCuLimit:       math.MaxUint64,
				}
			}

			plan := ts.plan // original mock template
			plan.PlanPolicy = defaultPolicy()

			if tt.planChainPolicy != nil {
				plan.PlanPolicy.ChainPolicies = []planstypes.ChainPolicy{*tt.planChainPolicy}
			}

			err := ts.TxProposalAddPlans(plan)
			require.Nil(t, err)

			_, sub1Addr := ts.AddAccount("sub", 0, 10000)

			_, err = ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 1)
			require.Nil(t, err)

			// get the admin project and set its policies
			subProjects, err := ts.QuerySubscriptionListProjects(sub1Addr)
			require.Nil(t, err)
			require.Equal(t, 1, len(subProjects.Projects))

			projectID := subProjects.Projects[0]

			if tt.projChainPolicy != nil {
				projPolicy := defaultPolicy()
				projPolicy.ChainPolicies = []planstypes.ChainPolicy{*tt.projChainPolicy}
				_, err = ts.TxProjectSetPolicy(projectID, sub1Addr, projPolicy)
				require.Nil(t, err)
			}

			// apply policy change
			ts.AdvanceEpoch()

			if tt.subscChainPolicy != nil {
				subscPolicy := defaultPolicy()
				subscPolicy.ChainPolicies = []planstypes.ChainPolicy{*tt.subscChainPolicy}
				_, err = ts.TxProjectSetSubscriptionPolicy(projectID, sub1Addr, subscPolicy)
				require.Nil(t, err)
			}

			// apply policy change
			ts.AdvanceEpochs(2)

			project, err := ts.GetProjectForBlock(projectID, ts.BlockHeight())
			require.NoError(t, err)

			strictestPolicy, err := ts.Keepers.Pairing.GetProjectStrictestPolicy(ts.Ctx, project, specId)
			require.NoError(t, err)
			if len(tt.expectedStrictestPolicies) > 0 {
				require.NotEqual(t, 0, len(strictestPolicy.ChainPolicies))
				require.NotEqual(t, 0, len(strictestPolicy.ChainPolicies[0].Collections))
				addons := map[string]struct{}{}
				for _, collection := range strictestPolicy.ChainPolicies[0].Collections {
					if collection.AddOn != "" {
						addons[collection.AddOn] = struct{}{}
					}
				}
				for _, expected := range tt.expectedStrictestPolicies {
					_, ok := addons[expected]
					require.True(t, ok, "did not find addon in strictest policy %s, policy: %#v", expected, strictestPolicy)
				}
			}

			pairing, err := ts.QueryPairingGetPairing(ts.spec.Index, sub1Addr)
			if tt.expectedProviders > 0 {
				require.Nil(t, err)
				require.Equal(t, tt.expectedProviders, len(pairing.Providers), "received providers %#v", pairing)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestSelectedProvidersPairing(t *testing.T) {
	ts := newTester(t)

	ts.addProvider(200)

	policy := &planstypes.Policy{
		GeolocationProfile: math.MaxUint64,
		MaxProvidersToPair: 3,
	}

	allowed := planstypes.SELECTED_PROVIDERS_MODE_ALLOWED
	exclusive := planstypes.SELECTED_PROVIDERS_MODE_EXCLUSIVE
	disabled := planstypes.SELECTED_PROVIDERS_MODE_DISABLED

	maxProvidersToPair, err := ts.Keepers.Pairing.CalculateEffectiveProvidersToPairFromPolicies(
		[]*planstypes.Policy{&ts.plan.PlanPolicy, policy},
	)
	require.Nil(t, err)

	ts.addProvider(200)
	_, p1 := ts.GetAccount(common.PROVIDER, 0)
	_, p2 := ts.GetAccount(common.PROVIDER, 1)
	_, p3 := ts.GetAccount(common.PROVIDER, 2)
	_, p4 := ts.GetAccount(common.PROVIDER, 3)
	_, p5 := ts.GetAccount(common.PROVIDER, 4)

	providerSets := []struct {
		planProviders []string
		subProviders  []string
		projProviders []string
	}{
		{[]string{}, []string{}, []string{}},                                 // set #0
		{[]string{p1, p2, p3}, []string{}, []string{}},                       // set #1
		{[]string{p1, p2}, []string{}, []string{}},                           // set #2
		{[]string{p3, p4}, []string{}, []string{}},                           // set #3
		{[]string{p1, p2, p3}, []string{p1, p2}, []string{}},                 // set #4
		{[]string{p1, p2, p3}, []string{}, []string{p1, p3}},                 // set #5
		{[]string{}, []string{p1, p2, p3}, []string{p1, p2}},                 // set #6
		{[]string{p1}, []string{p1, p2, p3}, []string{p1, p2}},               // set #7
		{[]string{p1, p2, p3, p4, p5}, []string{p1, p2, p3, p4}, []string{}}, // set #8
	}

	expectedSelectedProviders := [][]string{
		{p1, p2, p3},     // expected providers for intersection of set #1
		{p1, p2},         // expected providers for intersection of set #2
		{p3, p4},         // expected providers for intersection of set #3
		{p1, p2},         // expected providers for intersection of set #4
		{p1, p3},         // expected providers for intersection of set #5
		{p1, p2},         // expected providers for intersection of set #6
		{p1},             // expected providers for intersection of set #7
		{p1, p2, p3, p4}, // expected providers for intersection of set #8
	}

	// TODO: add mixed mode test cases (once implemented)
	templates := []struct {
		name              string
		planMode          planstypes.SELECTED_PROVIDERS_MODE
		subMode           planstypes.SELECTED_PROVIDERS_MODE
		projMode          planstypes.SELECTED_PROVIDERS_MODE
		providersSet      int
		expectedProviders int
	}{
		// normal pairing cases
		{"ALLOWED mode normal pairing", allowed, allowed, allowed, 0, 0},
		{"DISABLED mode normal pairing", disabled, allowed, allowed, 0, 0},

		// basic pairing checks cases
		{"EXCLUSIVE mode selected MaxProvidersToPair providers", exclusive, allowed, allowed, 1, 0},
		{"EXCLUSIVE mode selected less than MaxProvidersToPair providers", exclusive, allowed, allowed, 2, 1},
		{"EXCLUSIVE mode selected less than MaxProvidersToPair different providers", exclusive, allowed, allowed, 3, 2},

		// intersection checks cases
		{"EXCLUSIVE mode intersection between plan/sub policies", exclusive, exclusive, exclusive, 4, 3},
		{"EXCLUSIVE mode intersection between plan/proj policies", exclusive, exclusive, exclusive, 5, 4},
		{"EXCLUSIVE mode intersection between sub/proj policies", exclusive, exclusive, exclusive, 6, 5},
		{"EXCLUSIVE mode intersection between all policies", exclusive, exclusive, exclusive, 7, 6},

		// selected providers more than MaxProvidersToPair
		{"EXCLUSIVE mode selected more than MaxProvidersToPair providers", exclusive, exclusive, exclusive, 8, 7},

		// provider unstake checks cases
		{"EXCLUSIVE mode provider unstakes after first pairing", exclusive, exclusive, exclusive, 1, 0},
		{"EXCLUSIVE mode non-staked provider stakes after first pairing", exclusive, exclusive, exclusive, 1, 0},
	}

	var expectedProvidersAfterUnstake []string

	for i, tt := range templates {
		t.Run(tt.name, func(t *testing.T) {
			_, sub1Addr := ts.AddAccount("sub", i, 10000)

			// create plan, propose it and buy subscription
			plan := ts.Plan("mock")
			providersSet := providerSets[tt.providersSet]

			plan.PlanPolicy.SelectedProvidersMode = tt.planMode
			plan.PlanPolicy.SelectedProviders = providersSet.planProviders

			err := ts.TxProposalAddPlans(plan)
			require.Nil(t, err)

			_, err = ts.TxSubscriptionBuy(sub1Addr, sub1Addr, plan.Index, 1)
			require.Nil(t, err)

			// get the admin project and set its policies
			res, err := ts.QuerySubscriptionListProjects(sub1Addr)
			require.Nil(t, err)
			require.Equal(t, 1, len(res.Projects))

			project, err := ts.GetProjectForBlock(res.Projects[0], ts.BlockHeight())
			require.Nil(t, err)

			policy.SelectedProvidersMode = tt.projMode
			policy.SelectedProviders = providersSet.projProviders

			_, err = ts.TxProjectSetPolicy(project.Index, sub1Addr, *policy)
			require.Nil(t, err)

			// skip epoch for the policy change to take effect
			ts.AdvanceEpoch()

			policy.SelectedProvidersMode = tt.subMode
			policy.SelectedProviders = providersSet.subProviders

			_, err = ts.TxProjectSetSubscriptionPolicy(project.Index, sub1Addr, *policy)
			require.Nil(t, err)

			// skip epoch for the policy change to take effect
			ts.AdvanceEpoch()
			// and another epoch to get pairing of two consecutive epochs
			ts.AdvanceEpoch()

			pairing, err := ts.QueryPairingGetPairing(ts.spec.Index, sub1Addr)
			require.Nil(t, err)

			providerAddresses1 := []string{}
			for _, provider := range pairing.Providers {
				providerAddresses1 = append(providerAddresses1, provider.Address)
			}

			if tt.name == "EXCLUSIVE mode provider unstakes after first pairing" {
				// unstake p1 and remove from expected providers
				_, err = ts.TxPairingUnstakeProvider(p1, ts.spec.Index)
				require.Nil(t, err)
				expectedProvidersAfterUnstake = expectedSelectedProviders[tt.expectedProviders][1:]
			} else if tt.name == "EXCLUSIVE mode non-staked provider stakes after first pairing" {
				err := ts.StakeProvider(p1, ts.spec, 10000000)
				require.Nil(t, err)
			}

			ts.AdvanceEpoch()

			pairing, err = ts.QueryPairingGetPairing(ts.spec.Index, sub1Addr)
			require.Nil(t, err)

			providerAddresses2 := []string{}
			for _, provider := range pairing.Providers {
				providerAddresses2 = append(providerAddresses2, provider.Address)
			}

			// check pairings
			switch tt.name {
			case "ALLOWED mode normal pairing", "DISABLED mode normal pairing":
				require.False(t, slices.UnorderedEqual(providerAddresses1, providerAddresses2))
				require.Equal(t, maxProvidersToPair, uint64(len(providerAddresses1)))
				require.Equal(t, maxProvidersToPair, uint64(len(providerAddresses2)))

			case "EXCLUSIVE mode selected MaxProvidersToPair providers":
				require.True(t, slices.UnorderedEqual(providerAddresses1, providerAddresses2))
				require.Equal(t, maxProvidersToPair, uint64(len(providerAddresses2)))
				require.True(t, slices.UnorderedEqual(expectedSelectedProviders[tt.expectedProviders], providerAddresses1))

			case "EXCLUSIVE mode selected less than MaxProvidersToPair providers",
				"EXCLUSIVE mode selected less than MaxProvidersToPair different providers",
				"EXCLUSIVE mode intersection between plan/sub policies",
				"EXCLUSIVE mode intersection between plan/proj policies",
				"EXCLUSIVE mode intersection between sub/proj policies",
				"EXCLUSIVE mode intersection between all policies":
				require.True(t, slices.UnorderedEqual(providerAddresses1, providerAddresses2))
				require.Less(t, uint64(len(providerAddresses1)), maxProvidersToPair)
				require.True(t, slices.UnorderedEqual(expectedSelectedProviders[tt.expectedProviders], providerAddresses1))

			case "EXCLUSIVE mode selected more than MaxProvidersToPair providers":
				require.True(t, slices.IsSubset(providerAddresses1, expectedSelectedProviders[tt.expectedProviders]))
				require.True(t, slices.IsSubset(providerAddresses2, expectedSelectedProviders[tt.expectedProviders]))
				require.Equal(t, maxProvidersToPair, uint64(len(providerAddresses1)))
				require.Equal(t, maxProvidersToPair, uint64(len(providerAddresses2)))

			case "EXCLUSIVE mode provider unstakes after first pairing":
				require.False(t, slices.UnorderedEqual(providerAddresses1, providerAddresses2))
				require.True(t, slices.UnorderedEqual(expectedSelectedProviders[tt.expectedProviders], providerAddresses1))
				require.True(t, slices.UnorderedEqual(expectedProvidersAfterUnstake, providerAddresses2))

			case "EXCLUSIVE mode non-staked provider stakes after first pairing":
				require.False(t, slices.UnorderedEqual(providerAddresses1, providerAddresses2))
				require.True(t, slices.UnorderedEqual(expectedSelectedProviders[tt.expectedProviders], providerAddresses2))
				require.True(t, slices.UnorderedEqual(expectedProvidersAfterUnstake, providerAddresses1))
			}
		})
	}
}
