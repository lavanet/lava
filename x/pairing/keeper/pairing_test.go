package keeper_test

import (
	"math"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingscores "github.com/lavanet/lava/x/pairing/keeper/scores"
	"github.com/lavanet/lava/x/pairing/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
	projectstypes "github.com/lavanet/lava/x/projects/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	subscriptiontypes "github.com/lavanet/lava/x/subscription/types"
	"github.com/stretchr/testify/require"
)

func TestPairingUniqueness(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)

	// init keepers state
	spec := common.CreateMockSpec()
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	plan := common.CreateMockPlan()
	keepers.Plans.AddPlan(sdk.UnwrapSDKContext(ctx), plan)

	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	var balance int64 = 10000
	stake := balance / 10

	consumer1 := common.CreateNewAccount(ctx, *keepers, balance)
	common.BuySubscription(t, ctx, *keepers, *servers, consumer1, plan.Index)
	consumer2 := common.CreateNewAccount(ctx, *keepers, balance)
	common.BuySubscription(t, ctx, *keepers, *servers, consumer2, plan.Index)

	for i := 1; i <= 1000; i++ {
		provider := common.CreateNewAccount(ctx, *keepers, balance)
		common.StakeAccount(t, ctx, *keepers, *servers, provider, spec, stake)
	}

	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	// test that 2 different clients get different pairings
	providers1, err := keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ctx), spec.Index, consumer1.Addr)
	require.Nil(t, err)

	providers2, err := keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ctx), spec.Index, consumer2.Addr)
	require.Nil(t, err)

	require.Equal(t, len(providers1), len(providers2))

	different := false

	for _, provider := range providers1 {
		found := false
		for _, provider2 := range providers2 {
			if provider.Address == provider2.Address {
				found = true
			}
		}
		if !found {
			different = true
		}
	}

	require.True(t, different)

	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	// test that in different epoch we get different pairings for consumer1
	providers11, err := keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ctx), spec.Index, consumer1.Addr)
	require.Nil(t, err)

	require.Equal(t, len(providers1), len(providers11))
	different = false
	for i := range providers1 {
		if providers1[i].Address != providers11[i].Address {
			different = true
			break
		}
	}
	require.True(t, different)

	// test that get pairing gives the same results for the whole epoch
	epochBlocks := keepers.Epochstorage.EpochBlocksRaw(sdk.UnwrapSDKContext(ctx))
	for i := uint64(0); i < epochBlocks-1; i++ {
		ctx = testkeeper.AdvanceBlock(ctx, keepers)

		providers111, err := keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ctx), spec.Index, consumer1.Addr)
		require.Nil(t, err)

		for i := range providers1 {
			require.Equal(t, providers11[i].Address, providers111[i].Address)
			providerAddr, err := sdk.AccAddressFromBech32(providers11[i].Address)
			require.Nil(t, err)
			valid, _, _, _, _ := keepers.Pairing.ValidatePairingForClient(sdk.UnwrapSDKContext(ctx), spec.Index, consumer1.Addr, providerAddr, uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()))
			require.True(t, valid)
		}
	}
}

func TestValidatePairingDeterminism(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)

	// init keepers state
	spec := common.CreateMockSpec()
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	plan := common.CreateMockPlan()
	keepers.Plans.AddPlan(sdk.UnwrapSDKContext(ctx), plan)

	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	var balance int64 = 10000
	stake := balance / 10

	consumer1 := common.CreateNewAccount(ctx, *keepers, balance)
	common.BuySubscription(t, ctx, *keepers, *servers, consumer1, plan.Index)
	consumer2 := common.CreateNewAccount(ctx, *keepers, balance)
	common.BuySubscription(t, ctx, *keepers, *servers, consumer2, plan.Index)

	for i := 1; i <= 10; i++ {
		provider := common.CreateNewAccount(ctx, *keepers, balance)
		common.StakeAccount(t, ctx, *keepers, *servers, provider, spec, stake)
	}

	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	// test that 2 different clients get different pairings
	pairedProviders, err := keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ctx), spec.Index, consumer1.Addr)
	require.Nil(t, err)
	verifyPairingOncurrentBlock := uint64(sdk.UnwrapSDKContext(ctx).BlockHeight())
	testAllProviders := func() {
		for _, provider := range pairedProviders {
			providerAddress, err := sdk.AccAddressFromBech32(provider.Address)
			require.Nil(t, err)
			valid, _, _, _, errPairing := keepers.Pairing.ValidatePairingForClient(sdk.UnwrapSDKContext(ctx), spec.Index, consumer1.Addr, providerAddress, verifyPairingOncurrentBlock)
			require.Nil(t, errPairing)
			require.True(t, valid)
		}
	}
	startBlock := uint64(sdk.UnwrapSDKContext(ctx).BlockHeight())
	for i := startBlock; i < startBlock+(func() uint64 {
		blockToSave, err := keepers.Epochstorage.BlocksToSave(sdk.UnwrapSDKContext(ctx), i)
		require.Nil(t, err)
		return blockToSave
	})(); i++ {
		ctx = testkeeper.AdvanceBlock(ctx, keepers)
		testAllProviders()
	}
}

// Test that verifies that new get-pairing return values (CurrentEpoch, TimeLeftToNextPairing, SpecLastUpdatedBlock) is working properly
func TestGetPairing(t *testing.T) {
	// BLOCK_TIME = 30sec (testutil/keeper/keepers_init.go)
	constBlockTime := testkeeper.BLOCK_TIME

	// setup testnet with mock spec, stake a client and a provider
	ts := setupForPaymentTest(t)
	// get epochBlocks (number of blocks in an epoch)
	epochBlocks := ts.keepers.Epochstorage.EpochBlocksRaw(sdk.UnwrapSDKContext(ts.ctx))

	// define tests - different epoch, valid tells if the payment request should work
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
				ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
			case "commonEpoch":
				for i := 0; i < 5; i++ {
					ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
				}
			case "epochTimesChanged":
				for i := 0; i < 5; i++ {
					ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
				}
				smallerBlockTime := constBlockTime / 2
				ts.ctx = testkeeper.AdvanceBlocks(ts.ctx, ts.keepers, int(epochBlocks)/2, smallerBlockTime)
				ts.ctx = testkeeper.AdvanceBlocks(ts.ctx, ts.keepers, int(epochBlocks)/2)
			}

			// construct get-pairing request
			pairingReq := types.QueryGetPairingRequest{ChainID: ts.spec.Index, Client: ts.clients[0].Addr.String()}

			// get pairing for client (for epoch zero there is no pairing -> expect to fail)
			pairing, err := ts.keepers.Pairing.GetPairing(ts.ctx, &pairingReq)
			if !tt.validPairingExists {
				require.NotNil(t, err)
			} else {
				require.Nil(t, err)

				// verify the expected provider
				require.Equal(t, ts.providers[0].Addr.String(), pairing.Providers[0].Address)

				// verify the current epoch
				currentEpoch := ts.keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ts.ctx))
				require.Equal(t, currentEpoch, pairing.CurrentEpoch)

				// verify the SpecLastUpdatedBlock
				specLastUpdatedBlock := ts.spec.BlockLastUpdated
				require.Equal(t, specLastUpdatedBlock, pairing.SpecLastUpdatedBlock)

				// get timestamps from previous epoch
				prevEpoch, err := ts.keepers.Epochstorage.GetPreviousEpochStartForBlock(sdk.UnwrapSDKContext(ts.ctx), currentEpoch)
				require.Nil(t, err)

				// if prevEpoch == 0 -> averageBlockTime = 0, else calculate the time (like the actual get-pairing function)
				averageBlockTime := uint64(0)
				if prevEpoch != 0 {
					// get timestamps
					timestampList := []time.Time{}
					for block := prevEpoch; block <= currentEpoch; block++ {
						blockCore := ts.keepers.BlockStore.LoadBlock(int64(block))
						timestampList = append(timestampList, blockCore.Time)
					}

					// calculate average block time
					totalTime := uint64(0)
					for i := 1; i < len(timestampList); i++ {
						totalTime += uint64(timestampList[i].Sub(timestampList[i-1]).Seconds())
					}
					averageBlockTime = totalTime / epochBlocks
				}

				// Get the next epoch
				nextEpochStart, err := ts.keepers.Epochstorage.GetNextEpoch(sdk.UnwrapSDKContext(ts.ctx), currentEpoch)
				require.Nil(t, err)

				// Get epochBlocksOverlap
				overlapBlocks := ts.keepers.Pairing.EpochBlocksOverlap(sdk.UnwrapSDKContext(ts.ctx))

				// calculate the block in which the next pairing will happen (+overlap)
				nextPairingBlock := nextEpochStart + overlapBlocks

				// Get number of blocks from the current block to the next epoch
				blocksUntilNewEpoch := nextPairingBlock - uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight())

				// Calculate the time left for the next pairing in seconds (blocks left * avg block time)
				timeLeftToNextPairing := blocksUntilNewEpoch * averageBlockTime

				// verify the TimeLeftToNextPairing
				if !tt.isEpochTimesChanged {
					require.Equal(t, timeLeftToNextPairing, pairing.TimeLeftToNextPairing)
				} else {
					// averageBlockTime in get-pairing query -> minimal average across sampled epoch
					// averageBlockTime in this test -> normal average across epoch
					// we've used a smaller blocktime some of the time -> averageBlockTime from get-pairing is smaller than the averageBlockTime calculated in this test
					require.Less(t, pairing.TimeLeftToNextPairing, timeLeftToNextPairing)
				}

				// verify nextPairingBlock
				require.Equal(t, nextPairingBlock, pairing.BlockOfNextPairing)
			}
		})
	}
}

func TestPairingStatic(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)

	// init keepers state
	spec := common.CreateMockSpec()
	spec.ProvidersTypes = spectypes.Spec_static
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	plan := common.CreateMockPlan()
	keepers.Plans.AddPlan(sdk.UnwrapSDKContext(ctx), plan)

	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	consumer := common.CreateNewAccount(ctx, *keepers, balance)
	common.BuySubscription(t, ctx, *keepers, *servers, consumer, plan.Index)

	for i := uint64(0); i < plan.PlanPolicy.MaxProvidersToPair*2; i++ {
		provider := common.CreateNewAccount(ctx, *keepers, balance)
		common.StakeAccount(t, ctx, *keepers, *servers, provider, spec, stake+int64(i))
	}

	// we expect to get all the providers in static spec

	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	providers, err := keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ctx), spec.Index, consumer.Addr)
	require.Nil(t, err)

	for i, provider := range providers {
		require.Equal(t, provider.Stake.Amount.Int64(), stake+int64(i))
	}
}

func TestSelectedProvidersPairing(t *testing.T) {
	ts := setupForPaymentTest(t)
	_ctx := sdk.UnwrapSDKContext(ts.ctx)

	projPolicy := &planstypes.Policy{
		GeolocationProfile: math.MaxUint64,
		MaxProvidersToPair: 3,
	}

	err := ts.addProvider(200)
	require.Nil(t, err)

	allowed := planstypes.SELECTED_PROVIDERS_MODE_ALLOWED
	exclusive := planstypes.SELECTED_PROVIDERS_MODE_EXCLUSIVE
	disabled := planstypes.SELECTED_PROVIDERS_MODE_DISABLED

	maxProvidersToPair, err := ts.keepers.Pairing.CalculateEffectiveProvidersToPairFromPolicies(
		[]*planstypes.Policy{&ts.plan.PlanPolicy, projPolicy},
	)
	require.Nil(t, err)

	p1 := ts.providers[0].Addr.String()
	p2 := ts.providers[1].Addr.String()
	p3 := ts.providers[2].Addr.String()
	p4 := ts.providers[3].Addr.String()
	p5 := ts.providers[4].Addr.String()

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

	expectedProvidersAfterUnstake := []string{}
	for _, tt := range templates {
		t.Run(tt.name, func(t *testing.T) {
			// create plan, propose it and buy subscription
			plan := common.CreateMockPlan()
			subAddr := common.CreateNewAccount(ts.ctx, *ts.keepers, 10000).Addr.String()
			providersSet := providerSets[tt.providersSet]

			plan.PlanPolicy.SelectedProvidersMode = tt.planMode
			plan.PlanPolicy.SelectedProviders = providersSet.planProviders

			err := testkeeper.SimulatePlansAddProposal(_ctx, ts.keepers.Plans, []planstypes.Plan{plan})
			require.Nil(t, err)

			_, err = ts.servers.SubscriptionServer.Buy(ts.ctx, &subscriptiontypes.MsgBuy{
				Creator:  subAddr,
				Consumer: subAddr,
				Index:    plan.Index,
				Duration: 1,
			})
			require.Nil(t, err)

			// get the admin project and set its policies
			subProjects, err := ts.keepers.Subscription.ListProjects(ts.ctx, &subscriptiontypes.QueryListProjectsRequest{
				Subscription: subAddr,
			})
			require.Nil(t, err)
			require.Equal(t, 1, len(subProjects.Projects))

			adminProject, err := ts.keepers.Projects.GetProjectForBlock(_ctx, subProjects.Projects[0], uint64(_ctx.BlockHeight()))
			require.Nil(t, err)

			projPolicy.SelectedProvidersMode = tt.projMode
			projPolicy.SelectedProviders = providersSet.projProviders

			_, err = ts.servers.ProjectServer.SetPolicy(ts.ctx, &projectstypes.MsgSetPolicy{
				Creator: subAddr,
				Project: adminProject.Index,
				Policy:  *projPolicy,
			})
			require.Nil(t, err)

			// apply policy change
			ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
			_ctx = sdk.UnwrapSDKContext(ts.ctx)

			projPolicy.SelectedProvidersMode = tt.subMode
			projPolicy.SelectedProviders = providersSet.subProviders

			_, err = ts.servers.ProjectServer.SetSubscriptionPolicy(ts.ctx, &projectstypes.MsgSetSubscriptionPolicy{
				Creator:  subAddr,
				Projects: []string{adminProject.Index},
				Policy:   *projPolicy,
			})
			require.Nil(t, err)

			// apply policy change
			ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
			_ctx = sdk.UnwrapSDKContext(ts.ctx)

			// get pairing of two consecutive epochs
			ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
			_ctx = sdk.UnwrapSDKContext(ts.ctx)

			pairing, err := ts.keepers.Pairing.GetPairing(ts.ctx, &types.QueryGetPairingRequest{
				ChainID: ts.spec.Index,
				Client:  subAddr,
			})
			require.Nil(t, err)
			providerAddresses1 := []string{}
			for _, provider := range pairing.Providers {
				providerAddresses1 = append(providerAddresses1, provider.Address)
			}

			if tt.name == "EXCLUSIVE mode provider unstakes after first pairing" {
				_, err = ts.servers.PairingServer.UnstakeProvider(ts.ctx, &types.MsgUnstakeProvider{
					Creator: p1,
					ChainID: ts.spec.Index,
				})
				require.Nil(t, err)
				expectedProvidersAfterUnstake = expectedSelectedProviders[tt.expectedProviders][1:] // remove p1 from expected providers
			} else if tt.name == "EXCLUSIVE mode non-staked provider stakes after first pairing" {
				endpoints := []epochstoragetypes.Endpoint{{
					IPPORT:      "123",
					UseType:     ts.spec.GetApis()[0].ApiInterfaces[0].Interface,
					Geolocation: uint64(1),
				}}
				_, err = ts.servers.PairingServer.StakeProvider(ts.ctx, &types.MsgStakeProvider{
					Creator:     p1,
					ChainID:     ts.spec.Index,
					Amount:      sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(10000000)),
					Endpoints:   endpoints,
					Geolocation: uint64(1),
				})
				require.Nil(t, err)
			}

			ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
			_ctx = sdk.UnwrapSDKContext(ts.ctx)

			pairing, err = ts.keepers.Pairing.GetPairing(ts.ctx, &types.QueryGetPairingRequest{
				ChainID: ts.spec.Index,
				Client:  subAddr,
			})
			require.Nil(t, err)
			providerAddresses2 := []string{}
			for _, provider := range pairing.Providers {
				providerAddresses2 = append(providerAddresses2, provider.Address)
			}

			// check pairings
			switch tt.name {
			case "ALLOWED mode normal pairing", "DISABLED mode normal pairing":
				require.False(t, unorderedEqual(providerAddresses1, providerAddresses2))
				require.Equal(t, maxProvidersToPair, uint64(len(providerAddresses1)))
				require.Equal(t, maxProvidersToPair, uint64(len(providerAddresses2)))

			case "EXCLUSIVE mode selected MaxProvidersToPair providers":
				require.True(t, unorderedEqual(providerAddresses1, providerAddresses2))
				require.Equal(t, maxProvidersToPair, uint64(len(providerAddresses2)))
				require.True(t, unorderedEqual(expectedSelectedProviders[tt.expectedProviders], providerAddresses1))

			case "EXCLUSIVE mode selected less than MaxProvidersToPair providers",
				"EXCLUSIVE mode selected less than MaxProvidersToPair different providers",
				"EXCLUSIVE mode intersection between plan/sub policies",
				"EXCLUSIVE mode intersection between plan/proj policies",
				"EXCLUSIVE mode intersection between sub/proj policies",
				"EXCLUSIVE mode intersection between all policies":
				require.True(t, unorderedEqual(providerAddresses1, providerAddresses2))
				require.Less(t, uint64(len(providerAddresses1)), maxProvidersToPair)
				require.True(t, unorderedEqual(expectedSelectedProviders[tt.expectedProviders], providerAddresses1))

			case "EXCLUSIVE mode selected more than MaxProvidersToPair providers":
				require.True(t, IsSubset(providerAddresses1, expectedSelectedProviders[tt.expectedProviders]))
				require.True(t, IsSubset(providerAddresses2, expectedSelectedProviders[tt.expectedProviders]))
				require.Equal(t, maxProvidersToPair, uint64(len(providerAddresses1)))
				require.Equal(t, maxProvidersToPair, uint64(len(providerAddresses2)))

			case "EXCLUSIVE mode provider unstakes after first pairing":
				require.False(t, unorderedEqual(providerAddresses1, providerAddresses2))
				require.True(t, unorderedEqual(expectedSelectedProviders[tt.expectedProviders], providerAddresses1))
				require.True(t, unorderedEqual(expectedProvidersAfterUnstake, providerAddresses2))

			case "EXCLUSIVE mode non-staked provider stakes after first pairing":
				require.False(t, unorderedEqual(providerAddresses1, providerAddresses2))
				require.True(t, unorderedEqual(expectedSelectedProviders[tt.expectedProviders], providerAddresses2))
				require.True(t, unorderedEqual(expectedProvidersAfterUnstake, providerAddresses1))
			}
		})
	}
}

func unorderedEqual(first, second []string) bool {
	if len(first) != len(second) {
		return false
	}
	exists := make(map[string]bool)
	for _, value := range first {
		exists[value] = true
	}
	for _, value := range second {
		if !exists[value] {
			return false
		}
	}
	return true
}

func IsSubset(subset, superset []string) bool {
	// Create a map to store the elements of the superset
	elements := make(map[string]bool)

	// Populate the map with elements from the superset
	for _, elem := range superset {
		elements[elem] = true
	}

	// Check each element of the subset against the map
	for _, elem := range subset {
		if !elements[elem] {
			return false
		}
	}

	return true
}

func TestGeolocationPairingScores(t *testing.T) {
	ts := setupForPaymentTest(t)
	_ctx := sdk.UnwrapSDKContext(ts.ctx)

	// for convinience
	GL := uint64(planstypes.Geolocation_value["GL"])
	USE := uint64(planstypes.Geolocation_value["USE"])
	EU := uint64(planstypes.Geolocation_value["EU"])
	AS := uint64(planstypes.Geolocation_value["AS"])
	AF := uint64(planstypes.Geolocation_value["AF"])
	AU := uint64(planstypes.Geolocation_value["AU"])
	USC := uint64(planstypes.Geolocation_value["USC"])
	USW := uint64(planstypes.Geolocation_value["USW"])
	USE_EU := USE + EU

	minStake := sdk.NewInt(10)

	freePlanPolicy := planstypes.Policy{
		GeolocationProfile: 4, // USE
		TotalCuLimit:       10,
		EpochCuLimit:       2,
		MaxProvidersToPair: 5,
	}

	basicPlanPolicy := planstypes.Policy{
		GeolocationProfile: 0, // GLS
		TotalCuLimit:       10,
		EpochCuLimit:       2,
		MaxProvidersToPair: 14,
	}

	premiumPlanPolicy := planstypes.Policy{
		GeolocationProfile: 65535, // GL
		TotalCuLimit:       10,
		EpochCuLimit:       2,
		MaxProvidersToPair: 14,
	}

	// propose all plans and buy subscriptions
	freePlan := planstypes.Plan{
		Index:      "free",
		Block:      uint64(_ctx.BlockHeight()),
		Price:      sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(1)),
		PlanPolicy: freePlanPolicy,
	}

	basicPlan := planstypes.Plan{
		Index:      "basic",
		Block:      uint64(_ctx.BlockHeight()),
		Price:      sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(1)),
		PlanPolicy: basicPlanPolicy,
	}

	premiumPlan := planstypes.Plan{
		Index:      "premium",
		Block:      uint64(_ctx.BlockHeight()),
		Price:      sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(1)),
		PlanPolicy: premiumPlanPolicy,
	}

	plans := []planstypes.Plan{freePlan, basicPlan, premiumPlan}
	err := testkeeper.SimulatePlansAddProposal(_ctx, ts.keepers.Plans, plans)
	require.Nil(t, err)

	freeUser := common.CreateNewAccount(ts.ctx, *ts.keepers, 10000)
	basicUser := common.CreateNewAccount(ts.ctx, *ts.keepers, 10000)
	premiumUser := common.CreateNewAccount(ts.ctx, *ts.keepers, 10000)

	common.BuySubscription(t, ts.ctx, *ts.keepers, *ts.servers, freeUser, freePlan.Index)
	common.BuySubscription(t, ts.ctx, *ts.keepers, *ts.servers, basicUser, basicPlan.Index)
	common.BuySubscription(t, ts.ctx, *ts.keepers, *ts.servers, premiumUser, premiumPlan.Index)

	for geoName, geo := range planstypes.Geolocation_value {
		if geoName != "GL" && geoName != "GLS" {
			err = ts.addProviderGeolocation(5, uint64(geo))
			require.Nil(t, err)
		}
	}

	templates := []struct {
		name         string
		dev          common.Account
		planPolicy   planstypes.Policy
		changePolicy bool
		newGeo       uint64
		expectedGeo  []uint64
	}{
		// free plan (cannot change geolocation - verified in another test)
		{"default free plan", freeUser, freePlanPolicy, false, 0, []uint64{USE}},

		// basic plan (cannot change geolocation - verified in another test)
		{"default basic plan", basicUser, basicPlanPolicy, false, 0, []uint64{AF, AS, AU, EU, USE, USC, USW}},

		// premium plan (geolocation can change)
		{"default premium plan", premiumUser, premiumPlanPolicy, false, 0, []uint64{AF, AS, AU, EU, USE, USC, USW}},
		{"premium plan - set policy regular geo", premiumUser, premiumPlanPolicy, true, EU, []uint64{EU}},
		{"premium plan - set policy multiple geo", premiumUser, premiumPlanPolicy, true, USE_EU, []uint64{EU, USE}},
		{"premium plan - set policy global geo", premiumUser, premiumPlanPolicy, true, GL, []uint64{AF, AS, AU, EU, USE, USC, USW}},
	}

	for _, tt := range templates {
		t.Run(tt.name, func(t *testing.T) {
			devResponse, err := ts.keepers.Projects.Developer(ts.ctx, &projectstypes.QueryDeveloperRequest{
				Developer: tt.dev.Addr.String(),
			})
			require.Nil(t, err)

			projIndex := devResponse.Project.Index
			policies := []*planstypes.Policy{&tt.planPolicy}

			newPolicy := planstypes.Policy{}
			if tt.changePolicy {
				newPolicy = tt.planPolicy
				newPolicy.GeolocationProfile = tt.newGeo
				_, err = ts.servers.ProjectServer.SetPolicy(ts.ctx, &projectstypes.MsgSetPolicy{
					Creator: tt.dev.Addr.String(),
					Project: projIndex,
					Policy:  newPolicy,
				})
				require.Nil(t, err)
				policies = append(policies, &newPolicy)
			}

			ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers) // apply the new policy

			providersRes, err := ts.keepers.Pairing.Providers(ts.ctx, &types.QueryProvidersRequest{ChainID: ts.spec.Name})
			require.Nil(t, err)
			stakeEntries := providersRes.StakeEntry
			providerScores := []*pairingscores.PairingScore{}
			for i := range stakeEntries {
				providerScore := pairingscores.NewPairingScore(&stakeEntries[i])
				providerScores = append(providerScores, providerScore)
			}

			effectiveGeo, err := ts.keepers.Pairing.CalculateEffectiveGeolocationFromPolicies(policies)
			require.Nil(t, err)

			slots := pairingscores.CalcSlots(planstypes.Policy{
				GeolocationProfile: effectiveGeo,
				MaxProvidersToPair: tt.planPolicy.MaxProvidersToPair,
			}, minStake)

			geoSeen := map[uint64]bool{}
			for _, geo := range tt.expectedGeo {
				geoSeen[geo] = false
			}

			// calc scores and verify the scores are as expected
			for _, slot := range slots {
				err = pairingscores.CalcPairingScore(providerScores, pairingscores.GetStrategy(), slot, minStake)
				require.Nil(t, err)

				ok := pairingscores.VerifyGeoScoreForTesting(providerScores, slot, geoSeen)
				require.True(t, ok)
			}

			// verify that the slots have all the expected geos
			for _, found := range geoSeen {
				require.True(t, found)
			}
		})
	}
}

func isGeoInList(geo uint64, geoList []uint64) bool {
	for _, geoElem := range geoList {
		if geoElem == geo {
			return true
		}
	}
	return false
}

func TestDuplicateProviders(t *testing.T) {
	ts := setupForPaymentTest(t)
	_ctx := sdk.UnwrapSDKContext(ts.ctx)

	basicPlanPolicy := planstypes.Policy{
		GeolocationProfile: 0, // GLS
		TotalCuLimit:       10,
		EpochCuLimit:       2,
		MaxProvidersToPair: 14,
	}

	basicPlan := planstypes.Plan{
		Index:      "basic",
		Block:      uint64(_ctx.BlockHeight()),
		Price:      sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(1)),
		PlanPolicy: basicPlanPolicy,
	}

	basicUser := common.CreateNewAccount(ts.ctx, *ts.keepers, 10000)

	err := testkeeper.SimulatePlansAddProposal(_ctx, ts.keepers.Plans, []planstypes.Plan{basicPlan})
	require.Nil(t, err)

	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	common.BuySubscription(t, ts.ctx, *ts.keepers, *ts.servers, basicUser, basicPlan.Index)

	for geoName, geo := range planstypes.Geolocation_value {
		if geoName != "GL" && geoName != "GLS" {
			err := ts.addProviderGeolocation(5, uint64(geo))
			require.Nil(t, err)
		}
	}

	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	for i := 0; i < 100; i++ {
		pairingRes, err := ts.keepers.Pairing.GetPairing(ts.ctx, &types.QueryGetPairingRequest{
			ChainID: ts.spec.Index,
			Client:  basicUser.Addr.String(),
		})
		require.Nil(t, err)
		providerSeen := map[string]struct{}{}
		for _, provider := range pairingRes.Providers {
			_, found := providerSeen[provider.Address]
			require.False(t, found)
			providerSeen[provider.Address] = struct{}{}
		}
	}
}

// TestNoRequiredGeo checks that if no providers have the required geo, we still get a pairing list
func TestNoRequiredGeo(t *testing.T) {
	ts := setupForPaymentTest(t)
	_ctx := sdk.UnwrapSDKContext(ts.ctx)

	freePlanPolicy := planstypes.Policy{
		GeolocationProfile: 4, // USE
		TotalCuLimit:       10,
		EpochCuLimit:       2,
		MaxProvidersToPair: 5,
	}

	freePlan := planstypes.Plan{
		Index:      "free",
		Block:      uint64(_ctx.BlockHeight()),
		Price:      sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(1)),
		PlanPolicy: freePlanPolicy,
	}

	freeUser := common.CreateNewAccount(ts.ctx, *ts.keepers, 10000)

	err := testkeeper.SimulatePlansAddProposal(_ctx, ts.keepers.Plans, []planstypes.Plan{freePlan})
	require.Nil(t, err)

	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	common.BuySubscription(t, ts.ctx, *ts.keepers, *ts.servers, freeUser, freePlan.Index)

	// add 5 more providers that are not in US-E (the only allowed providers in the free plan)
	err = ts.addProviderGeolocation(5, uint64(planstypes.Geolocation_value["AS"]))
	require.Nil(t, err)

	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	pairingRes, err := ts.keepers.Pairing.GetPairing(ts.ctx, &types.QueryGetPairingRequest{
		ChainID: ts.spec.Index,
		Client:  freeUser.Addr.String(),
	})
	require.Nil(t, err)
	require.Equal(t, freePlanPolicy.MaxProvidersToPair, uint64(len(pairingRes.Providers)))
	for _, provider := range pairingRes.Providers {
		require.NotEqual(t, freePlanPolicy.GeolocationProfile, provider.Geolocation)
	}
}
