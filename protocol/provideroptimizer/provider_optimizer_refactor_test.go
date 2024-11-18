package provideroptimizer

import (
	"slices"
	"strconv"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/lavanet/lava/v3/utils"
	"github.com/lavanet/lava/v3/utils/rand"
	spectypes "github.com/lavanet/lava/v3/x/spec/types"
	"github.com/stretchr/testify/require"
)

const (
	TEST_AVERAGE_BLOCK_TIME_Refactor = 10 * time.Second
	TEST_BASE_WORLD_LATENCY_Refactor = 10 * time.Millisecond // same as score.DefaultLatencyNum
)

func setupProviderOptimizer_Refactor(maxProvidersCount uint) *ProviderOptimizer_Refactor {
	averageBlockTIme := TEST_AVERAGE_BLOCK_TIME_Refactor
	return NewProviderOptimizer_Refactor(StrategyBalanced_Refactor, averageBlockTIme, maxProvidersCount)
}

type providersGenerator_Refactor struct {
	providersAddresses []string
}

func (pg *providersGenerator_Refactor) setupProvidersForTest_Refactor(count int) *providersGenerator_Refactor {
	pg.providersAddresses = make([]string, count)
	for i := range pg.providersAddresses {
		pg.providersAddresses[i] = "lava@test_" + strconv.Itoa(i)
	}
	return pg
}

// TestProviderOptimizerProviderDataSetGet_Refactor tests that the providerData
// Get and Set methods work as expected
func TestProviderOptimizerProviderDataSetGet_Refactor(t *testing.T) {
	providerOptimizer := setupProviderOptimizer_Refactor(1)
	providersGen := (&providersGenerator_Refactor{}).setupProvidersForTest_Refactor(1)
	providerAddress := providersGen.providersAddresses[0]
	for i := 0; i < 100; i++ {
		providerData := ProviderData_Refactor{SyncBlock: uint64(i)}
		address := providerAddress + strconv.Itoa(i)
		set := providerOptimizer.providersStorage.Set(address, providerData, 1)
		if set == false {
			utils.LavaFormatWarning("set in cache dropped", nil)
		}
	}
	time.Sleep(4 * time.Millisecond)
	for i := 0; i < 100; i++ {
		address := providerAddress + strconv.Itoa(i)
		providerData, found := providerOptimizer.getProviderData_Refactor(address)
		require.Equal(t, uint64(i), providerData.SyncBlock, "failed getting entry %s", address)
		require.True(t, found)
	}
}

// TestProviderOptimizerBasicProbeData_Refactor tests the basic provider optimizer operation
// when it is updated with probe relays. Providers with bad scores should have a worse chance
// to be picked (and vice versa).
// Scenario:
//  0. There are 10 providers, the optimizer is configured to pick a single provider
//  1. Choose between 10 identical providers -> none should be in the worst tier
//  2. Append bad probe relay data for providers 5-7 and pick providers -> should not be 6-8
//  3. Append good probe relay data for providers 0-2 and pick providers -> should often be 0-2
func TestProviderOptimizerBasicProbeData_Refactor(t *testing.T) {
	providerOptimizer := setupProviderOptimizer_Refactor(1)
	providersGen := (&providersGenerator_Refactor{}).setupProvidersForTest_Refactor(10)
	rand.InitRandomSeed()
	cu := uint64(10)
	requestBlock := int64(1000)

	// choose between 10 identical providers, none should be in the worst tier
	returnedProviders, tier := providerOptimizer.ChooseProvider_Refactor(providersGen.providersAddresses, nil, cu, requestBlock)
	require.Equal(t, 1, len(returnedProviders))
	require.NotEqual(t, 4, tier)

	// damage providers 5-7 scores with bad latency probes relays
	// they should not be selected by the optimizer and should be in the worst tier
	badLatency := TEST_BASE_WORLD_LATENCY_Refactor * 3
	providerOptimizer.AppendProbeRelayData_Refactor(providersGen.providersAddresses[5], badLatency, true)
	providerOptimizer.AppendProbeRelayData_Refactor(providersGen.providersAddresses[6], badLatency, true)
	providerOptimizer.AppendProbeRelayData_Refactor(providersGen.providersAddresses[7], badLatency, true)
	time.Sleep(4 * time.Millisecond)
	returnedProviders, _ = providerOptimizer.ChooseProvider_Refactor(providersGen.providersAddresses, nil, cu, requestBlock)
	require.Equal(t, 1, len(returnedProviders))
	require.NotEqual(t, 4, tier)
	require.NotEqual(t, returnedProviders[0], providersGen.providersAddresses[5]) // we shouldn't pick the worst provider
	require.NotEqual(t, returnedProviders[0], providersGen.providersAddresses[6]) // we shouldn't pick the worst provider
	require.NotEqual(t, returnedProviders[0], providersGen.providersAddresses[7]) // we shouldn't pick the worst provider

	// improve providers 0-2 scores with good latency probes relays
	// they should be selected by the optimizer more often and should be in the best tier
	goodLatency := TEST_BASE_WORLD_LATENCY_Refactor / 2
	providerOptimizer.AppendProbeRelayData_Refactor(providersGen.providersAddresses[0], goodLatency, true)
	providerOptimizer.AppendProbeRelayData_Refactor(providersGen.providersAddresses[1], goodLatency, true)
	providerOptimizer.AppendProbeRelayData_Refactor(providersGen.providersAddresses[2], goodLatency, true)
	time.Sleep(4 * time.Millisecond)
	results, tierResults := runChooseManyTimesAndReturnResults_Refactor(t, providerOptimizer, providersGen.providersAddresses, nil, 1000, cu, requestBlock)
	require.Greater(t, tierResults[0], 600, tierResults) // we should pick the best tier most often

	// out of 10 providers, and with 3 providers in the top tier we should pick
	// tier-0 providers around a third of that
	require.Greater(t, results[providersGen.providersAddresses[0]], 200, results) // we should pick the best tier most often
}

// runChooseManyTimesAndReturnResults_Refactor uses the given optimizer and providers addresses
// to pick providers <times> times and return two results maps:
//  1. results: map of provider address to the number of times it was picked
//  2. tierResults: map of tier and the number of times a provider from the specific tier was picked
func runChooseManyTimesAndReturnResults_Refactor(t *testing.T, providerOptimizer *ProviderOptimizer_Refactor, providers []string, ignoredProviders map[string]struct{}, times int, cu uint64, requestBlock int64) (map[string]int, map[int]int) {
	tierResults := make(map[int]int)
	results := make(map[string]int)
	for i := 0; i < times; i++ {
		returnedProviders, tier := providerOptimizer.ChooseProvider_Refactor(providers, ignoredProviders, cu, requestBlock)
		require.Equal(t, 1, len(returnedProviders))
		results[returnedProviders[0]]++
		tierResults[tier]++
	}
	return results, tierResults
}

// TestProviderOptimizerBasicRelayData_Refactor tests the basic provider optimizer operation
// when it is updated with regular relays. Providers with bad scores should have a worse chance
// to be picked (and vice versa).
// Scenario:
//  0. There are 10 providers, the optimizer is configured to pick a single provider
//  1. Choose between 10 identical providers -> none should be in the worst tier
//  2. Append bad relay data for providers 5-7 and pick providers -> should not be 6-8
//  3. Append good relay data for providers 0-2 and pick providers -> should often be 0-2
func TestProviderOptimizerBasicRelayData_Refactor(t *testing.T) {
	providerOptimizer := setupProviderOptimizer_Refactor(1)
	providersGen := (&providersGenerator_Refactor{}).setupProvidersForTest_Refactor(10)
	rand.InitRandomSeed()
	cu := uint64(1)
	requestBlock := int64(1000)
	syncBlock := uint64(requestBlock)

	// choose between 10 identical providers, none should be in the worst tier
	returnedProviders, tier := providerOptimizer.ChooseProvider_Refactor(providersGen.providersAddresses, nil, cu, requestBlock)
	require.Equal(t, 1, len(returnedProviders))
	require.NotEqual(t, 4, tier)

	// damage providers 5-7 scores with bad latency relays
	// they should not be selected by the optimizer and should be in the worst tier
	badLatency := TEST_BASE_WORLD_LATENCY_Refactor * 3
	providerOptimizer.AppendRelayData_Refactor(providersGen.providersAddresses[5], badLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData_Refactor(providersGen.providersAddresses[6], badLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData_Refactor(providersGen.providersAddresses[7], badLatency, cu, syncBlock)
	time.Sleep(4 * time.Millisecond)
	returnedProviders, tier = providerOptimizer.ChooseProvider_Refactor(providersGen.providersAddresses, nil, cu, requestBlock)
	require.Equal(t, 1, len(returnedProviders))
	require.NotEqual(t, tier, 3) // we shouldn't pick the low tier providers
	require.NotEqual(t, returnedProviders[0], providersGen.providersAddresses[5], tier)
	require.NotEqual(t, returnedProviders[0], providersGen.providersAddresses[6], tier)
	require.NotEqual(t, returnedProviders[0], providersGen.providersAddresses[7], tier)

	// improve providers 0-2 scores with good latency probes relays
	// they should be selected by the optimizer more often and should be in the best tier
	goodLatency := TEST_BASE_WORLD_LATENCY_Refactor / 2
	providerOptimizer.AppendRelayData_Refactor(providersGen.providersAddresses[0], goodLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData_Refactor(providersGen.providersAddresses[1], goodLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData_Refactor(providersGen.providersAddresses[2], goodLatency, cu, syncBlock)
	time.Sleep(4 * time.Millisecond)
	results, tierResults := runChooseManyTimesAndReturnResults_Refactor(t, providerOptimizer, providersGen.providersAddresses, nil, 1000, cu, requestBlock)
	require.Greater(t, tierResults[0], 600, tierResults) // we should pick the best tier most often

	// out of 10 providers, and with 3 in the top tier we should pick 0 around a third of that
	require.Greater(t, results[providersGen.providersAddresses[0]], 200, results)

	// the bad providers shouldn't have been picked even once
	require.Zero(t, results[providersGen.providersAddresses[5]])
	require.Zero(t, results[providersGen.providersAddresses[6]])
	require.Zero(t, results[providersGen.providersAddresses[7]])
}

// TestProviderOptimizerAvailabilityProbeData_Refactor tests the availability update when
// the optimizer is updated with failed probe relays. Providers with bad scores should have
// a worse chance to be picked (and vice versa).
// Scenario:
//  0. There are 100 providers, the optimizer is configured to pick a single provider
//  1. Append bad probe relay data for all provider but random three
//  2. Pick providers and check they're picked most often
func TestProviderOptimizerAvailabilityProbeData_Refactor(t *testing.T) {
	providerOptimizer := setupProviderOptimizer_Refactor(1)
	providersCount := 100
	cu := uint64(1)
	requestBlock := int64(1000)
	providersGen := (&providersGenerator_Refactor{}).setupProvidersForTest_Refactor(providersCount)
	rand.InitRandomSeed()

	// damage all the providers scores with failed probe relays but three random ones
	skipIndex := rand.Intn(providersCount - 3)
	providerOptimizer.OptimizerNumTiers = 33 // set many tiers so good providers can stand out in the test
	for i := range providersGen.providersAddresses {
		// give all providers a worse availability score except these 3
		if i == skipIndex || i == skipIndex+1 || i == skipIndex+2 {
			// skip 0
			continue
		}
		providerOptimizer.AppendProbeRelayData_Refactor(providersGen.providersAddresses[i], TEST_BASE_WORLD_LATENCY_Refactor, false)
	}

	// pick providers, the three random ones should be top-tier and picked more often
	time.Sleep(4 * time.Millisecond)
	results, tierResults := runChooseManyTimesAndReturnResults_Refactor(t, providerOptimizer, providersGen.providersAddresses, nil, 1000, cu, requestBlock)
	require.Greater(t, tierResults[0], 300, tierResults) // 0.42 chance for top tier due to the algorithm to rebalance chances
	require.Greater(t, results[providersGen.providersAddresses[skipIndex]]+results[providersGen.providersAddresses[skipIndex+1]]+results[providersGen.providersAddresses[skipIndex+2]], 300)
	require.InDelta(t, results[providersGen.providersAddresses[skipIndex]], results[providersGen.providersAddresses[skipIndex+1]], 50)

	// pick providers again but this time ignore one of the random providers, it shouldn't be picked
	results, _ = runChooseManyTimesAndReturnResults_Refactor(t, providerOptimizer, providersGen.providersAddresses, map[string]struct{}{providersGen.providersAddresses[skipIndex]: {}}, 1000, cu, requestBlock)
	require.Zero(t, results[providersGen.providersAddresses[skipIndex]])
}

// TestProviderOptimizerAvailabilityProbeData_Refactor tests the availability update when
// the optimizer is updated with failed relays. Providers with bad scores should have
// a worse chance to be picked (and vice versa).
// Scenario:
//  0. There are 100 providers, the optimizer is configured to pick a single provider
//  1. Append bad probe relay data for all provider but random three
//  2. Pick providers and check they're picked most often
func TestProviderOptimizerAvailabilityRelayData_Refactor(t *testing.T) {
	providerOptimizer := setupProviderOptimizer_Refactor(1)
	providersCount := 100
	cu := uint64(10)
	requestBlock := int64(1000)
	providersGen := (&providersGenerator_Refactor{}).setupProvidersForTest_Refactor(providersCount)
	rand.InitRandomSeed()

	// damage all the providers scores with failed probe relays but three random ones
	skipIndex := rand.Intn(providersCount - 3)
	providerOptimizer.OptimizerNumTiers = 33 // set many tiers so good providers can stand out in the test
	for i := range providersGen.providersAddresses {
		// give all providers a worse availability score except these 3
		if i == skipIndex || i == skipIndex+1 || i == skipIndex+2 {
			// skip 0
			continue
		}
		providerOptimizer.AppendRelayFailure_Refactor(providersGen.providersAddresses[i])
	}

	// pick providers, the three random ones should be top-tier and picked more often
	time.Sleep(4 * time.Millisecond)
	results, tierResults := runChooseManyTimesAndReturnResults_Refactor(t, providerOptimizer, providersGen.providersAddresses, nil, 1000, cu, requestBlock)
	require.Greater(t, tierResults[0], 300, tierResults) // 0.42 chance for top tier due to the algorithm to rebalance chances
	require.Greater(t, results[providersGen.providersAddresses[skipIndex]]+results[providersGen.providersAddresses[skipIndex+1]]+results[providersGen.providersAddresses[skipIndex+2]], 270)
	require.InDelta(t, results[providersGen.providersAddresses[skipIndex]], results[providersGen.providersAddresses[skipIndex+1]], 50)

	// pick providers again but this time ignore one of the random providers, it shouldn't be picked
	results, _ = runChooseManyTimesAndReturnResults_Refactor(t, providerOptimizer, providersGen.providersAddresses, map[string]struct{}{providersGen.providersAddresses[skipIndex]: {}}, 1000, cu, requestBlock)
	require.Zero(t, results[providersGen.providersAddresses[skipIndex]])
}

func TestProviderOptimizerAvailabilityBlockError_Refactor(t *testing.T) {
	providerOptimizer := setupProviderOptimizer_Refactor(1)
	providersCount := 10
	providersGen := (&providersGenerator_Refactor{}).setupProvidersForTest_Refactor(providersCount)
	rand.InitRandomSeed()
	cu := uint64(10)
	requestBlock := int64(1000)
	syncBlock := uint64(1000)
	badSyncBlock := syncBlock - 1

	// damage all the providers scores with bad sync relays but three random ones
	// the three random providers also get slightly worse latency
	// bad sync means an update that doesn't have the latest requested block
	chosenIndex := rand.Intn(providersCount - 2)
	for i := range providersGen.providersAddresses {
		time.Sleep(4 * time.Millisecond)
		if i == chosenIndex || i == chosenIndex+1 || i == chosenIndex+2 {
			slightlyBadLatency := TEST_BASE_WORLD_LATENCY_Refactor + 1*time.Millisecond
			providerOptimizer.AppendRelayData_Refactor(providersGen.providersAddresses[i], slightlyBadLatency, cu, syncBlock)
			continue
		}
		providerOptimizer.AppendRelayData_Refactor(providersGen.providersAddresses[i], TEST_BASE_WORLD_LATENCY_Refactor, cu, badSyncBlock)
	}

	// make the top tier chance to be 70%
	time.Sleep(4 * time.Millisecond)
	selectionTier, _ := providerOptimizer.CalculateSelectionTiers_Refactor(providersGen.providersAddresses, nil, cu, requestBlock)
	tierChances := selectionTier.ShiftTierChance(OptimizerNumTiers, map[int]float64{0: ATierChance, OptimizerNumTiers - 1: LastTierChance})
	require.Greater(t, tierChances[0], 0.7, tierChances)

	// pick providers, the top-tier should be picked picked more often (at least half the times)
	results, tierResults := runChooseManyTimesAndReturnResults_Refactor(t, providerOptimizer, providersGen.providersAddresses, nil, 1000, cu, requestBlock)
	require.Greater(t, tierResults[0], 500, tierResults)

	// out of 10 providers, and with 3 in the top tier we should pick 0 around a third of that
	require.Greater(t, results[providersGen.providersAddresses[chosenIndex]], 200, results)
	sumResults := results[providersGen.providersAddresses[chosenIndex]] + results[providersGen.providersAddresses[chosenIndex+1]] + results[providersGen.providersAddresses[chosenIndex+2]]
	require.Greater(t, sumResults, 500, results) // we should pick the best tier most often

	// now try to get a previous block, our chosenIndex should be inferior in latency and blockError chance should be the same
	results, tierResults = runChooseManyTimesAndReturnResults_Refactor(t, providerOptimizer, providersGen.providersAddresses, nil, 1000, cu, requestBlock-1)
	require.Greater(t, tierResults[0], 500, tierResults) // we should pick the best tier most often
	// out of 10 providers, and with 3 in the top tier we should pick 0 around a third of that
	require.Less(t, results[providersGen.providersAddresses[chosenIndex]], 50, results) // chosen indexes shoulnt be in the tier
	sumResults = results[providersGen.providersAddresses[chosenIndex]] + results[providersGen.providersAddresses[chosenIndex+1]] + results[providersGen.providersAddresses[chosenIndex+2]]
	require.Less(t, sumResults, 150, results) // we should pick the best tier most often
}

// TestProviderOptimizerUpdatingLatency_Refactor tests checks that repeatedly adding better results
// (with both probes and relays) makes the latency score improve
func TestProviderOptimizerUpdatingLatency_Refactor(t *testing.T) {
	providerOptimizer := setupProviderOptimizer_Refactor(1)
	providersCount := 2
	providersGen := (&providersGenerator_Refactor{}).setupProvidersForTest_Refactor(providersCount)
	providerAddress := providersGen.providersAddresses[0]
	cu := uint64(10)
	requestBlock := int64(1000)
	syncBlock := uint64(requestBlock)

	// add an average latency probe relay to determine average score
	providerOptimizer.AppendProbeRelayData_Refactor(providerAddress, TEST_BASE_WORLD_LATENCY_Refactor, true)
	time.Sleep(4 * time.Millisecond)

	// add good latency probe relays, score should improve
	for i := 0; i < 10; i++ {
		// get current score
		data, _ := providerOptimizer.getProviderData_Refactor(providerAddress)
		qos, _ := providerOptimizer.GetExcellenceQoSReportForProvider_Refactor(data, providerAddress)
		score, err := qos.ComputeQoSExcellence_Refactor()
		require.NoError(t, err)

		// add good latency probe
		providerOptimizer.AppendProbeRelayData_Refactor(providerAddress, TEST_BASE_WORLD_LATENCY_Refactor/10, true)
		time.Sleep(4 * time.Millisecond)

		// check score again and compare to the last score
		data, _ = providerOptimizer.getProviderData_Refactor(providerAddress)
		qos, _ = providerOptimizer.GetExcellenceQoSReportForProvider_Refactor(data, providerAddress)
		newScore, err := qos.ComputeQoSExcellence_Refactor()
		require.NoError(t, err)
		require.True(t, newScore.LT(score), "newScore: "+newScore.String()+", score: "+score.String())
	}

	// add an average latency probe relay to determine average score
	providerAddress = providersGen.providersAddresses[1]
	providerOptimizer.AppendRelayData_Refactor(providerAddress, TEST_BASE_WORLD_LATENCY_Refactor, cu, syncBlock)
	time.Sleep(4 * time.Millisecond)

	// add good latency relays, score should improve
	for i := 0; i < 10; i++ {
		// get current score
		data, _ := providerOptimizer.getProviderData_Refactor(providerAddress)
		qos, _ := providerOptimizer.GetExcellenceQoSReportForProvider_Refactor(data, providerAddress)
		score, err := qos.ComputeQoSExcellence_Refactor()
		require.NoError(t, err)

		// add good latency relay
		providerOptimizer.AppendRelayData_Refactor(providerAddress, TEST_BASE_WORLD_LATENCY_Refactor/10, cu, syncBlock)
		time.Sleep(4 * time.Millisecond)

		// check score again and compare to the last score
		data, _ = providerOptimizer.getProviderData_Refactor(providerAddress)
		qos, _ = providerOptimizer.GetExcellenceQoSReportForProvider_Refactor(data, providerAddress)
		newScore, err := qos.ComputeQoSExcellence_Refactor()
		require.NoError(t, err)
		require.True(t, newScore.LT(score), "newScore: "+newScore.String()+", score: "+score.String())
	}
}

func TestProviderOptimizerExploration_Refactor(t *testing.T) {
	providerOptimizer := setupProviderOptimizer_Refactor(2)
	providersCount := 10
	providersGen := (&providersGenerator_Refactor{}).setupProvidersForTest_Refactor(providersCount)
	cu := uint64(10)
	requestBlock := int64(1000)
	syncBlock := uint64(requestBlock)

	rand.InitRandomSeed()
	// start with a disabled chosen index
	chosenIndex := -1
	testProvidersExploration := func(iterations int) float64 {
		exploration := 0.0
		for i := 0; i < iterations; i++ {
			returnedProviders, _ := providerOptimizer.ChooseProvider_Refactor(providersGen.providersAddresses, nil, cu, requestBlock)
			if len(returnedProviders) > 1 {
				exploration++
				// check if we have a specific chosen index
				if chosenIndex >= 0 {
					// there's only one provider eligible for exploration it must be him
					require.Equal(t, providersGen.providersAddresses[chosenIndex], returnedProviders[1])
				}
			}
		}
		return exploration
	}

	// make sure exploration works when providers are defaulted (no data at all)
	exploration := testProvidersExploration(1000)
	require.Greater(t, exploration, float64(10))

	chosenIndex = rand.Intn(providersCount - 2)
	// set chosen index with a value in the past so it can be selected for exploration
	providerOptimizer.appendRelayData_Refactor(providersGen.providersAddresses[chosenIndex], TEST_BASE_WORLD_LATENCY_Refactor*2, true, cu, syncBlock, time.Now().Add(-35*time.Second))
	// set a basic state for all other provider, with a recent time (so they can't be selected for exploration)
	for i := 0; i < 10; i++ {
		for index, address := range providersGen.providersAddresses {
			if index == chosenIndex {
				// we set chosenIndex with a past time so it can be selected for exploration
				continue
			}
			// set samples in the future so they are never a candidate for exploration
			providerOptimizer.appendRelayData_Refactor(address, TEST_BASE_WORLD_LATENCY_Refactor*2, true, cu, syncBlock, time.Now().Add(1*time.Second))
		}
		time.Sleep(4 * time.Millisecond)
	}

	// with a cost strategy we expect exploration at a 10% rate
	providerOptimizer.strategy = StrategyBalanced_Refactor // that's the default but to be explicit
	providerOptimizer.wantedNumProvidersInConcurrency = 2  // that's in the constructor but to be explicit
	iterations := 10000
	exploration = testProvidersExploration(iterations)
	require.Less(t, exploration, float64(1.4)*float64(iterations)*DefaultExplorationChance_Refactor)    // allow mistake buffer of 40% because of randomness
	require.Greater(t, exploration, float64(0.6)*float64(iterations)*DefaultExplorationChance_Refactor) // allow mistake buffer of 40% because of randomness

	// with a cost strategy we expect exploration to happen once in 100 samples
	providerOptimizer.strategy = StrategyCost_Refactor
	exploration = testProvidersExploration(iterations)
	require.Less(t, exploration, float64(1.4)*float64(iterations)*CostExplorationChance_Refactor)    // allow mistake buffer of 40% because of randomness
	require.Greater(t, exploration, float64(0.6)*float64(iterations)*CostExplorationChance_Refactor) // allow mistake buffer of 40% because of randomness

	// privacy disables exploration
	providerOptimizer.strategy = StrategyPrivacy_Refactor
	exploration = testProvidersExploration(iterations)
	require.Equal(t, exploration, float64(0))
}

func TestProviderOptimizerSyncScore_Refactor(t *testing.T) {
	providerOptimizer := setupProviderOptimizer_Refactor(1)
	providersGen := (&providersGenerator_Refactor{}).setupProvidersForTest_Refactor(10)
	rand.InitRandomSeed()
	cu := uint64(10)
	requestBlock := spectypes.LATEST_BLOCK

	syncBlock := uint64(1000)

	chosenIndex := rand.Intn(len(providersGen.providersAddresses))
	sampleTime := time.Now()
	for j := 0; j < 3; j++ { // repeat several times because a sync score is only correct after all providers sent their first block otherwise its giving favor to the first one
		for i := range providersGen.providersAddresses {
			time.Sleep(4 * time.Millisecond)
			if i == chosenIndex {
				// give better syncBlock, latency is a tiny bit worse for the second check
				providerOptimizer.appendRelayData_Refactor(providersGen.providersAddresses[i], TEST_BASE_WORLD_LATENCY_Refactor*2+1*time.Microsecond, true, cu, syncBlock+5, sampleTime)
				continue
			}
			providerOptimizer.appendRelayData_Refactor(providersGen.providersAddresses[i], TEST_BASE_WORLD_LATENCY_Refactor*2, true, cu, syncBlock, sampleTime) // update that he doesn't have the latest requested block
		}
		sampleTime = sampleTime.Add(time.Millisecond * 5)
	}
	time.Sleep(4 * time.Millisecond)
	selectionTier, _ := providerOptimizer.CalculateSelectionTiers_Refactor(providersGen.providersAddresses, nil, cu, requestBlock)
	tier0 := selectionTier.GetTier(0, 4, 3)
	require.Greater(t, len(tier0), 0) // shouldn't be empty
	// we have the best score on the top tier and it's sorted
	require.Equal(t, providersGen.providersAddresses[chosenIndex], tier0[0].Address)

	// now choose with a specific block that all providers have
	selectionTier, _ = providerOptimizer.CalculateSelectionTiers_Refactor(providersGen.providersAddresses, nil, cu, int64(syncBlock))
	tier0 = selectionTier.GetTier(0, 4, 3)
	for idx := range tier0 {
		// sync score doesn't matter now so the tier0 is recalculated and chosenIndex has worst latency
		require.NotEqual(t, providersGen.providersAddresses[chosenIndex], tier0[idx].Address)
	}
}

func TestProviderOptimizerStrategiesScoring_Refactor(t *testing.T) {
	rand.InitRandomSeed()
	providerOptimizer := setupProviderOptimizer_Refactor(1)
	providersCount := 10
	providersGen := (&providersGenerator_Refactor{}).setupProvidersForTest_Refactor(providersCount)
	cu := uint64(10)
	requestBlock := spectypes.LATEST_BLOCK
	syncBlock := uint64(1000)

	// set a basic state for all providers
	sampleTime := time.Now()
	for i := 0; i < 10; i++ {
		for _, address := range providersGen.providersAddresses {
			providerOptimizer.appendRelayData_Refactor(address, TEST_BASE_WORLD_LATENCY_Refactor*2, true, cu, syncBlock, sampleTime)
		}
		time.Sleep(4 * time.Millisecond)
	}
	// provider 2 doesn't get a probe availability hit, this is the most meaningful factor
	for idx, address := range providersGen.providersAddresses {
		if idx != 2 {
			providerOptimizer.AppendProbeRelayData_Refactor(address, TEST_BASE_WORLD_LATENCY_Refactor*2, false)
			time.Sleep(4 * time.Millisecond)
		}
		providerOptimizer.AppendProbeRelayData_Refactor(address, TEST_BASE_WORLD_LATENCY_Refactor*2, true)
		time.Sleep(4 * time.Millisecond)
		providerOptimizer.AppendProbeRelayData_Refactor(address, TEST_BASE_WORLD_LATENCY_Refactor*2, false)
		time.Sleep(4 * time.Millisecond)
		providerOptimizer.AppendProbeRelayData_Refactor(address, TEST_BASE_WORLD_LATENCY_Refactor*2, true)
		time.Sleep(4 * time.Millisecond)
		providerOptimizer.AppendProbeRelayData_Refactor(address, TEST_BASE_WORLD_LATENCY_Refactor*2, true)
		time.Sleep(4 * time.Millisecond)
	}

	sampleTime = time.Now()
	improvedLatency := TEST_BASE_WORLD_LATENCY_Refactor / 2
	normalLatency := TEST_BASE_WORLD_LATENCY_Refactor * 2
	improvedBlock := syncBlock + 1
	// provider 0 gets a good latency
	providerOptimizer.appendRelayData_Refactor(providersGen.providersAddresses[0], improvedLatency, true, cu, syncBlock, sampleTime)

	// providers 3,4 get a regular entry
	providerOptimizer.appendRelayData_Refactor(providersGen.providersAddresses[3], normalLatency, true, cu, syncBlock, sampleTime)
	providerOptimizer.appendRelayData_Refactor(providersGen.providersAddresses[4], normalLatency, true, cu, syncBlock, sampleTime)

	// provider 1 gets a good sync
	providerOptimizer.appendRelayData_Refactor(providersGen.providersAddresses[1], normalLatency, true, cu, improvedBlock, sampleTime)

	sampleTime = sampleTime.Add(10 * time.Millisecond)
	// now repeat to modify all providers scores across sync calculation
	providerOptimizer.appendRelayData_Refactor(providersGen.providersAddresses[0], improvedLatency, true, cu, syncBlock, sampleTime)
	providerOptimizer.appendRelayData_Refactor(providersGen.providersAddresses[3], normalLatency, true, cu, syncBlock, sampleTime)
	providerOptimizer.appendRelayData_Refactor(providersGen.providersAddresses[4], normalLatency, true, cu, syncBlock, sampleTime)
	providerOptimizer.appendRelayData_Refactor(providersGen.providersAddresses[1], normalLatency, true, cu, improvedBlock, sampleTime)

	time.Sleep(4 * time.Millisecond)
	providerOptimizer.strategy = StrategyBalanced_Refactor
	// a balanced strategy should pick provider 2 because of it's high availability
	selectionTier, _ := providerOptimizer.CalculateSelectionTiers_Refactor(providersGen.providersAddresses, nil, cu, requestBlock)
	tier0 := selectionTier.GetTier(0, 4, 3)
	require.Greater(t, len(tier0), 0) // shouldn't be empty
	// we have the best score on the top tier and it's sorted
	require.Equal(t, providersGen.providersAddresses[2], tier0[0].Address)

	providerOptimizer.strategy = StrategyCost_Refactor
	// with a cost strategy we expect the same as balanced
	selectionTier, _ = providerOptimizer.CalculateSelectionTiers_Refactor(providersGen.providersAddresses, nil, cu, requestBlock)
	tier0 = selectionTier.GetTier(0, 4, 3)
	require.Greater(t, len(tier0), 0) // shouldn't be empty
	// we have the best score on the top tier and it's sorted
	require.Equal(t, providersGen.providersAddresses[2], tier0[0].Address)

	providerOptimizer.strategy = StrategyLatency_Refactor
	// latency strategy should pick the best latency
	selectionTier, _ = providerOptimizer.CalculateSelectionTiers_Refactor(providersGen.providersAddresses, map[string]struct{}{providersGen.providersAddresses[2]: {}}, cu, requestBlock)
	tier0 = selectionTier.GetTier(0, 4, 3)
	require.Greater(t, len(tier0), 0) // shouldn't be empty
	require.Equal(t, providersGen.providersAddresses[0], tier0[0].Address)

	providerOptimizer.strategy = StrategySyncFreshness_Refactor
	// freshness strategy should pick the most advanced provider
	selectionTier, _ = providerOptimizer.CalculateSelectionTiers_Refactor(providersGen.providersAddresses, map[string]struct{}{providersGen.providersAddresses[2]: {}}, cu, requestBlock)
	tier0 = selectionTier.GetTier(0, 4, 3)
	require.Greater(t, len(tier0), 0) // shouldn't be empty
	require.Equal(t, providersGen.providersAddresses[1], tier0[0].Address)

	// but if we request a past block, then it doesnt matter and we choose by latency:
	selectionTier, _ = providerOptimizer.CalculateSelectionTiers_Refactor(providersGen.providersAddresses, map[string]struct{}{providersGen.providersAddresses[2]: {}}, cu, int64(syncBlock))
	tier0 = selectionTier.GetTier(0, 4, 3)
	require.Greater(t, len(tier0), 0) // shouldn't be empty
	require.Equal(t, providersGen.providersAddresses[0], tier0[0].Address)
}

func TestExcellence_Refactor(t *testing.T) {
	providerOptimizer := setupProviderOptimizer_Refactor(1)
	providersCount := 5
	providersGen := (&providersGenerator_Refactor{}).setupProvidersForTest_Refactor(providersCount)
	cu := uint64(10)
	syncBlock := uint64(1000)
	// set a basic state for all of them
	sampleTime := time.Now()
	for i := 0; i < 10; i++ {
		for _, address := range providersGen.providersAddresses {
			providerOptimizer.appendRelayData_Refactor(address, TEST_BASE_WORLD_LATENCY_Refactor*2, true, cu, syncBlock, sampleTime)
		}
		time.Sleep(4 * time.Millisecond)
	}
	data, _ := providerOptimizer.getProviderData_Refactor(providersGen.providersAddresses[0])
	report, sampleTime1 := providerOptimizer.GetExcellenceQoSReportForProvider_Refactor(data, providersGen.providersAddresses[0])
	require.NotNil(t, report)
	require.True(t, sampleTime.Equal(sampleTime1))
	data, _ = providerOptimizer.getProviderData_Refactor(providersGen.providersAddresses[1])
	report2, sampleTime2 := providerOptimizer.GetExcellenceQoSReportForProvider_Refactor(data, providersGen.providersAddresses[1])
	require.NotNil(t, report2)
	require.Equal(t, report, report2)
	require.True(t, sampleTime.Equal(sampleTime2))
}

// test low providers count 0-9
func TestProviderOptimizerProvidersCount_Refactor(t *testing.T) {
	rand.InitRandomSeed()
	providerOptimizer := setupProviderOptimizer_Refactor(1)
	providersCount := 10
	providersGen := (&providersGenerator_Refactor{}).setupProvidersForTest_Refactor(providersCount)
	cu := uint64(10)
	requestBlock := int64(1000)
	syncBlock := uint64(1000)
	sampleTime := time.Now()
	for i := 0; i < 10; i++ {
		for _, address := range providersGen.providersAddresses {
			providerOptimizer.appendRelayData_Refactor(address, TEST_BASE_WORLD_LATENCY_Refactor*2, true, cu, syncBlock, sampleTime)
		}
		time.Sleep(4 * time.Millisecond)
	}
	playbook := []struct {
		name      string
		providers int
	}{
		{name: "one", providers: 1},
		{name: "two", providers: 2},
		{name: "three", providers: 3},
		{name: "four", providers: 4},
		{name: "five", providers: 5},
		{name: "six", providers: 6},
		{name: "seven", providers: 7},
		{name: "eight", providers: 8},
		{name: "nine", providers: 9},
	}
	for _, play := range playbook {
		t.Run(play.name, func(t *testing.T) {
			for i := 0; i < 10; i++ {
				returnedProviders, _ := providerOptimizer.ChooseProvider_Refactor(providersGen.providersAddresses[:play.providers], nil, cu, requestBlock)
				require.Greater(t, len(returnedProviders), 0)
			}
		})
	}
}

func TestProviderOptimizerWeights_Refactor(t *testing.T) {
	rand.InitRandomSeed()
	providerOptimizer := setupProviderOptimizer_Refactor(1)
	providersCount := 10
	providersGen := (&providersGenerator_Refactor{}).setupProvidersForTest_Refactor(providersCount)
	cu := uint64(10)
	requestBlock := spectypes.LATEST_BLOCK
	syncBlock := uint64(1000)
	sampleTime := time.Now()
	weights := map[string]int64{
		providersGen.providersAddresses[0]: 10000000000000, // simulating 10m tokens
	}
	for i := 1; i < 10; i++ {
		weights[providersGen.providersAddresses[i]] = 50000000000
	}

	normalLatency := TEST_BASE_WORLD_LATENCY_Refactor * 2
	improvedLatency := normalLatency - 5*time.Millisecond
	improvedBlock := syncBlock + 2

	providerOptimizer.UpdateWeights_Refactor(weights)
	for i := 0; i < 10; i++ {
		for idx, address := range providersGen.providersAddresses {
			if idx == 0 {
				providerOptimizer.appendRelayData_Refactor(address, normalLatency, true, cu, improvedBlock, sampleTime)
			} else {
				providerOptimizer.appendRelayData_Refactor(address, improvedLatency, true, cu, syncBlock, sampleTime)
			}
			sampleTime = sampleTime.Add(5 * time.Millisecond)
			time.Sleep(4 * time.Millisecond)
		}
	}

	// verify 0 has the best score
	selectionTier, _ := providerOptimizer.CalculateSelectionTiers_Refactor(providersGen.providersAddresses, nil, cu, requestBlock)
	tier0 := selectionTier.GetTier(0, 4, 3)
	require.Greater(t, len(tier0), 0) // shouldn't be empty
	require.Equal(t, providersGen.providersAddresses[0], tier0[0].Address)

	// if we pick by sync, provider 0 is in the top tier and should be selected very often
	results, tierResults := runChooseManyTimesAndReturnResults_Refactor(t, providerOptimizer, providersGen.providersAddresses, nil, 1000, cu, requestBlock)
	require.Greater(t, tierResults[0], 600, tierResults) // we should pick the best tier most often
	// out of 10 providers, and with 3 in the top tier we should pick 0 around a third of that
	require.Greater(t, results[providersGen.providersAddresses[0]], 550, results) // we should pick the top provider in tier 0 most times due to weight

	// if we pick by latency only, provider 0 is in the worst tier and can't be selected at all
	results, tierResults = runChooseManyTimesAndReturnResults_Refactor(t, providerOptimizer, providersGen.providersAddresses, nil, 1000, cu, int64(syncBlock))
	require.Greater(t, tierResults[0], 500, tierResults) // we should pick the best tier most often
	// out of 10 providers, and with 3 in the top tier we should pick 0 around a third of that
	require.Zero(t, results[providersGen.providersAddresses[0]])
}

func TestProviderOptimizerTiers_Refactor(t *testing.T) {
	rand.InitRandomSeed()
	cu := uint64(10)
	requestBlock := int64(1000)
	providersCountList := []int{9, 10}
	for why, providersCount := range providersCountList {
		providerOptimizer := setupProviderOptimizer_Refactor(1)
		providersGen := (&providersGenerator_Refactor{}).setupProvidersForTest_Refactor(providersCount)
		syncBlock := uint64(1000)
		sampleTime := time.Now()
		normalLatency := TEST_BASE_WORLD_LATENCY_Refactor * 2
		for i := 0; i < 10; i++ {
			for _, address := range providersGen.providersAddresses {
				modifierLatency := rand.Int63n(3) - 1
				modifierSync := rand.Int63n(3) - 1
				providerOptimizer.appendRelayData_Refactor(address, normalLatency+time.Duration(modifierLatency)*time.Millisecond, true, cu, syncBlock+uint64(modifierSync), sampleTime)
				sampleTime = sampleTime.Add(5 * time.Millisecond)
				time.Sleep(4 * time.Millisecond)
			}
		}
		selectionTier, _ := providerOptimizer.CalculateSelectionTiers_Refactor(providersGen.providersAddresses, nil, cu, requestBlock)
		shiftedChances := selectionTier.ShiftTierChance(4, map[int]float64{0: 0.75})
		require.NotZero(t, shiftedChances[3])
		// if we pick by sync, provider 0 is in the top tier and should be selected very often
		_, tierResults := runChooseManyTimesAndReturnResults_Refactor(t, providerOptimizer, providersGen.providersAddresses, nil, 1000, cu, requestBlock)
		for index := 0; index < OptimizerNumTiers; index++ {
			if providersCount >= 2*MinimumEntries && index == OptimizerNumTiers-1 {
				// skip last tier if there's insufficient providers
				continue
			}
			require.NotZero(t, tierResults[index], "tierResults %v providersCount %s index %d why: %d", tierResults, providersCount, index, why)
		}
	}
}

// TestProviderOptimizerChooseProvider checks that the follwing occurs:
// 0. Assume 6 providers: 2 with great score, 2 with mid score but one has a great stake, and 2 with low score (benchmark).
// We choose 2 providers in each choice. We choose many times.
// 1. ~80% of the times, the great score providers are picked (no preference between the two)
// 2. high stake mid score is picked more than 0 times and picked more than mid score with average stake
// 3. low score are not selected
// TODO: for some reason, provider 0 is picked half the times compared to provider 1 (should be equal)
func TestProviderOptimizerChooseProvider(t *testing.T) {
	rand.InitRandomSeed()
	providerOptimizer := setupProviderOptimizer_Refactor(1)
	providersCount := 6
	providerOptimizer.OptimizerNumTiers = providersCount / 2 // make each tier contain 2 providers
	providerOptimizer.OptimizerMinTierEntries = 2
	providersGen := (&providersGenerator_Refactor{}).setupProvidersForTest_Refactor(providersCount)
	cu := uint64(10)
	requestBlock := spectypes.LATEST_BLOCK
	syncBlock := uint64(1000)
	sampleTime := time.Now()

	// apply high stake for provider 2
	normalStake := int64(50000000000)
	highStake := 5 * normalStake
	highStakeProviderIndex := 2
	weights := map[string]int64{}
	for i := 0; i < providersCount; i++ {
		if i == highStakeProviderIndex {
			weights[providersGen.providersAddresses[i]] = highStake
		} else {
			weights[providersGen.providersAddresses[i]] = normalStake
		}
	}
	providerOptimizer.UpdateWeights_Refactor(weights)

	// setup scores to all providers
	improvedLatency := TEST_BASE_WORLD_LATENCY_Refactor / 2
	normalLatency := TEST_BASE_WORLD_LATENCY_Refactor * 2
	improvedBlock := syncBlock + 1

	// provider 0 and 1 gets a good latency and good sync
	providerOptimizer.appendRelayData_Refactor(providersGen.providersAddresses[0], improvedLatency, true, cu, improvedBlock, sampleTime)
	providerOptimizer.appendRelayData_Refactor(providersGen.providersAddresses[1], improvedLatency, true, cu, improvedBlock, sampleTime)

	// providers 2 and 3 get a good latency only
	providerOptimizer.appendRelayData_Refactor(providersGen.providersAddresses[2], improvedLatency, true, cu, syncBlock, sampleTime)
	providerOptimizer.appendRelayData_Refactor(providersGen.providersAddresses[3], improvedLatency, true, cu, syncBlock, sampleTime)

	// provider 4 and 5 gets a normal latency and sync
	providerOptimizer.appendRelayData_Refactor(providersGen.providersAddresses[4], normalLatency, true, cu, syncBlock, sampleTime)
	providerOptimizer.appendRelayData_Refactor(providersGen.providersAddresses[5], normalLatency, true, cu, syncBlock, sampleTime)

	// now repeat to modify all providers scores across sync calculation
	sampleTime = sampleTime.Add(10 * time.Millisecond)
	time.Sleep(10 * time.Millisecond)
	providerOptimizer.appendRelayData_Refactor(providersGen.providersAddresses[5], normalLatency, true, cu, syncBlock, sampleTime)
	providerOptimizer.appendRelayData_Refactor(providersGen.providersAddresses[4], normalLatency, true, cu, syncBlock, sampleTime)
	providerOptimizer.appendRelayData_Refactor(providersGen.providersAddresses[3], improvedLatency, true, cu, syncBlock, sampleTime)
	providerOptimizer.appendRelayData_Refactor(providersGen.providersAddresses[2], improvedLatency, true, cu, syncBlock, sampleTime)
	providerOptimizer.appendRelayData_Refactor(providersGen.providersAddresses[1], improvedLatency, true, cu, improvedBlock, sampleTime)
	providerOptimizer.appendRelayData_Refactor(providersGen.providersAddresses[0], improvedLatency, true, cu, improvedBlock, sampleTime)
	time.Sleep(4 * time.Millisecond)

	// choose many times and check results
	iterations := 10000
	results, tierResults := runChooseManyTimesAndReturnResults_Refactor(t, providerOptimizer, providersGen.providersAddresses, nil, iterations, cu, requestBlock)
	require.InDelta(t, float64(iterations)*0.7, tierResults[0], float64(iterations)*0.1) // high score are picked 60%-80% of the times
	require.InDelta(t, results[providersGen.providersAddresses[0]],
		results[providersGen.providersAddresses[1]], float64(results[providersGen.providersAddresses[0]])*0.1) // no difference between high score providers (max 10% diff)
	require.Greater(t, results[providersGen.providersAddresses[2]], 0)                                           // high stake mid score provider picked at least once
	require.Greater(t, results[providersGen.providersAddresses[2]], results[providersGen.providersAddresses[3]]) // high stake mid score provider picked more than normal stake mid score provider
	require.Equal(t, 0, results[providersGen.providersAddresses[3]])
	require.Equal(t, 0, results[providersGen.providersAddresses[4]])
}

// TestProviderOptimizerRetriesWithReducedProvidersSet checks that when having a set of providers, the amount of
// providers doesn't matter and the choice is deterministic. The test does the following:
// 0. Assume a set of providers (great/mid/low score with high/low stake, all combinations)
// 1. Run ChooseProvider() <providers_amount> number of times. Each iteration, the chosen provider from the
// last iteration is removed from the providers set. We check the ranking of providers stays the same.
// 2. Do step 1 many times.
// Expected: the ranking of providers stays the same, providers with high stake are picked more often,
// providers from the lowest tier are not picked
// TODO: make odd providers high stake
// TODO: add checks on tier results
func TestProviderOptimizerRetriesWithReducedProvidersSet(t *testing.T) {
	rand.InitRandomSeed()
	providerOptimizer := setupProviderOptimizer_Refactor(1)
	providersCount := 6
	providersGen := (&providersGenerator_Refactor{}).setupProvidersForTest_Refactor(providersCount)

	// create 3 tiers with 2 providers each
	providerOptimizer.OptimizerNumTiers = providersCount / 2
	providerOptimizer.OptimizerMinTierEntries = 2

	// apply high stake for providers 0, 2, 4
	normalStake := int64(50000000000)
	highStake := 5 * normalStake
	highStakeProviderIndexes := []int{0, 2, 4}
	weights := map[string]int64{}
	for i := 0; i < providersCount; i++ {
		if slices.Contains(highStakeProviderIndexes, i) {
			weights[providersGen.providersAddresses[i]] = highStake
		} else {
			weights[providersGen.providersAddresses[i]] = normalStake
		}
	}
	providerOptimizer.UpdateWeights_Refactor(weights)
	cu := uint64(10)
	requestBlock := int64(1000)
	syncBlock := uint64(1000)
	sampleTime := time.Now()
	baseLatency := TEST_BASE_WORLD_LATENCY_Refactor.Seconds()

	// append relay data for each provider depending on its index in the providers array
	// the latency gets worse for increasing index so we assume the best provider is the 1st
	// address, after it the 2nd and so on
	for i := 0; i < 50; i++ {
		for j, address := range providersGen.providersAddresses {
			latency := time.Duration(baseLatency * float64(2*j+1) * float64(time.Millisecond))
			providerOptimizer.appendRelayData_Refactor(address, latency, true, cu, syncBlock, sampleTime)
		}
		sampleTime = sampleTime.Add(5 * time.Millisecond)
		time.Sleep(5 * time.Millisecond)
	}

	// choose many times with different sets of providers and check the ranking stays the same
	// Expected: providers with high stake are picked more often, providers from the lowest tier are not picked
	// Note, on the last two iterations, providers 4,5 are picked and provider 4 is picked more than provider 5
	// since there is only one tier and provider 4 has higher stake than provider 5
	for i := 0; i < providersCount; i++ {
		// run and choose many times and keep a map of provider address -> number of times it was picked
		iterations := 1000
		res, _ := runChooseManyTimesAndReturnResults_Refactor(t, providerOptimizer, providersGen.providersAddresses[i:], nil, iterations, cu, requestBlock)

		switch i {
		case 0:
			// 6 providers, 3 tiers, last one not picked so only
			// providers 0,1,2,3 are picked. tier 0: providers 0,1
			// tier 1: providers 2,3
			// provider 0,2 have higher stake and should be picked more often within their tier
			require.Equal(t, 4, len(res))
			require.Greater(t, res[providersGen.providersAddresses[0]], res[providersGen.providersAddresses[1]])
			require.Greater(t, res[providersGen.providersAddresses[2]], res[providersGen.providersAddresses[3]])
		case 1:
			// 5 providers, 3 tiers, last one not picked so only
			// providers 1,2,3 are picked. tier 0: providers 1,2
			// tier 1: provider 3
			// provider 2 has higher stake and should be picked more often within their tier
			require.Equal(t, 3, len(res))
			require.Greater(t, res[providersGen.providersAddresses[2]], res[providersGen.providersAddresses[1]])
		case 2:
			// 4 providers, 3 tiers, last one not picked so only
			// providers 2,3,4 are picked. tier 0: providers 2,3
			// tier 1: provider 4
			// provider 2 has higher stake and should be picked more often within their tier
			require.Equal(t, 3, len(res))
			require.Greater(t, res[providersGen.providersAddresses[2]], res[providersGen.providersAddresses[3]])
		case 3:
			// less than OptimizerMinTierEntries*2 so last tier chance isn't 0
			// 3 providers, 3 tiers, last one not picked so only
			// providers 3,4,5 are picked. tier 0: providers 3
			// tier 1: providers 4,5
			// provider 4 has higher stake and should be picked more often within their tier
			// TODO: make sure all can be selected
			selectionTiers, _ := providerOptimizer.CalculateSelectionTiers_Refactor(providersGen.providersAddresses[i:], nil, cu, requestBlock)
			require.NotNil(t, selectionTiers)
			for ii := 0; ii < 3; ii++ {
				tier := selectionTiers.GetTier(ii, 3, 2)
				require.NotEmpty(t, tier)
			}
			require.Equal(t, 3, len(res))
			require.Greater(t, res[providersGen.providersAddresses[4]], res[providersGen.providersAddresses[5]])
		case 4:
			// 2 providers, 3 tiers
			// providers 4,5 are picked. tier 0: providers 4,5
			// provider 4 has higher stake and should be picked more often within their tier
			// TODO: let's make sure Tier0 doesn't have 5
			require.Equal(t, 2, len(res))
			require.Greater(t, res[providersGen.providersAddresses[4]], res[providersGen.providersAddresses[5]])
		}
	}
}

// TestProviderOptimizerChoiceSimulation checks that the overall choice mechanism acts as expected,
// For each of the following metrics: latency, sync, availability and stake we do the following:
// 0. Assume 2 providers
// 1. Append relay data for both providers with random samples. The "better" provider will have a randomized
// sample with a better range (for example, the better one gets latency of 10-30ms and the bad one gets 25-40ms)
// 2. Choose between them and verify the better one is chosen more.
// TODO: not passing - the providers are picked pretty evenly
// let's do 2 tiers minimumper tier 1, after fixing tiering function we expect a clear preference for the better scored provider
func TestProviderOptimizerChoiceSimulation(t *testing.T) {
	rand.InitRandomSeed()
	providerOptimizer := setupProviderOptimizer_Refactor(1)
	providersCount := 2
	providersGen := (&providersGenerator_Refactor{}).setupProvidersForTest_Refactor(providersCount)
	cu := uint64(10)
	requestBlock := int64(1000)
	syncBlock := uint64(1000)
	sampleTime := time.Now()
	baseLatency := TEST_BASE_WORLD_LATENCY_Refactor.Seconds()

	// initial values
	p1Latency := baseLatency * float64(time.Millisecond)
	p2Latency := baseLatency * float64(time.Millisecond)
	p1SyncBlock := syncBlock
	p2SyncBlock := syncBlock
	p1Availability := true
	p2Availability := true

	// append relay data for each provider depending on its index in the providers array
	// the latency gets worse for increasing index so we assume the best provider is the 1st
	// address, after it the 2nd and so on
	for i := 0; i < 1000; i++ {
		// randomize latency, provider 0 gets a better latency than provider 1
		p1Latency += float64(rand.Int63n(21)+10) * float64(time.Millisecond) // Random number between 10-30
		p2Latency += float64(rand.Int63n(16)+25) * float64(time.Millisecond) // Random number between 30-40 // TODO: fix

		// randomize sync, provider 0 gets a better sync than provider 1
		// TODO: increment both or only p1 10/5
		if rand.Float64() < 0.8 { // 80% chance to increment
			p1SyncBlock++
		}
		if rand.Float64() < 0.4 { // 40% chance to increment
			p2SyncBlock++
		}

		// randomize availability, provider 0 gets a better availability than provider 1
		// TODO same here
		if rand.Float64() < 0.8 { // 80% chance to true
			p1Availability = true
		} else {
			p1Availability = false
		}
		if rand.Float64() < 0.4 { // 40% chance to true
			p2Availability = true
		} else {
			p2Availability = false
		}

		providerOptimizer.appendRelayData_Refactor(providersGen.providersAddresses[0], time.Duration(p1Latency), p1Availability, cu, p1SyncBlock, sampleTime)
		providerOptimizer.appendRelayData_Refactor(providersGen.providersAddresses[1], time.Duration(p2Latency), p2Availability, cu, p2SyncBlock, sampleTime)

		sampleTime = sampleTime.Add(5 * time.Millisecond)
		time.Sleep(5 * time.Millisecond)
	}

	// choose many times and check the better provider is chosen more often (provider 0)
	iterations := 1000
	res, _ := runChooseManyTimesAndReturnResults_Refactor(t, providerOptimizer, providersGen.providersAddresses, nil, iterations, cu, requestBlock)
	require.Greater(t, res[providersGen.providersAddresses[0]], res[providersGen.providersAddresses[1]])
}

// TestProviderOptimizerLatencySyncScore tests that a provider with 100ms latency and x sync block
// has the same score as a provider with 1100ms latency but x+1 sync block
// This is true since the average block time is 10sec and the default sync factor is 0.1. So
// score_good_latency = latency + sync_factor * sync_lag + ... = 100 + 0.1 * 10000 + ... = 1100 + ...
// score_good_sync = latency + sync_factor * sync_lag + ... = 1100 + 0.1 * 0 + ... = 1100 + ...
// TODO: not working - it seems that the sync affects less than expected
// TODO: add block time to sample time for worse sync block
func TestProviderOptimizerLatencySyncScore(t *testing.T) {
	rand.InitRandomSeed()
	providerOptimizer := setupProviderOptimizer_Refactor(1)
	providersCount := 2
	providersGen := (&providersGenerator_Refactor{}).setupProvidersForTest_Refactor(providersCount)
	cu := uint64(10)
	requestBlock := spectypes.LATEST_BLOCK
	syncBlock := uint64(1000)

	improvedLatency := TEST_BASE_WORLD_LATENCY_Refactor
	badLatency := TEST_BASE_WORLD_LATENCY_Refactor + TEST_AVERAGE_BLOCK_TIME

	// set a basic state for all providers
	sampleTime := time.Now()
	for i := 0; i < 10; i++ {
		for _, address := range providersGen.providersAddresses {
			providerOptimizer.appendRelayData_Refactor(address, TEST_BASE_WORLD_LATENCY_Refactor*2, true, cu, syncBlock, sampleTime)
		}
		time.Sleep(4 * time.Millisecond)
	}

	sampleTime = sampleTime.Add(TEST_AVERAGE_BLOCK_TIME_Refactor)
	// provider 0 gets a good sync with bad latency
	providerOptimizer.appendRelayData_Refactor(providersGen.providersAddresses[0], badLatency, true, cu, syncBlock+1, sampleTime)

	// provider 1 gets a good latency with bad sync
	providerOptimizer.appendRelayData_Refactor(providersGen.providersAddresses[1], improvedLatency, true, cu, syncBlock, sampleTime)

	// verify both providers have the same score
	scores := []math.LegacyDec{}
	for _, provider := range providersGen.providersAddresses {
		data, found := providerOptimizer.getProviderData_Refactor(provider)
		require.True(t, found)
		qos, _ := providerOptimizer.GetExcellenceQoSReportForProvider_Refactor(data, provider)
		score, err := qos.ComputeQoSExcellence_Refactor()
		require.NoError(t, err)
		scores = append(scores, score)
	}
	require.Len(t, scores, 2)
	require.True(t, scores[0].Equal(scores[1]))

	// choose many times - since their scores should be the same, they should be picked in a similar amount
	res, _ := runChooseManyTimesAndReturnResults_Refactor(t, providerOptimizer, providersGen.providersAddresses, nil, 1000, cu, requestBlock)
	require.Equal(t, res[providersGen.providersAddresses[0]], res[providersGen.providersAddresses[1]])
}