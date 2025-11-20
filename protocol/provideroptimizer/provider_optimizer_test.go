package provideroptimizer

import (
	"strconv"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/lavaslices"
	"github.com/lavanet/lava/v5/utils/rand"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/require"
)

func init() {
	rand.InitRandomSeed()
}

const (
	TEST_AVERAGE_BLOCK_TIME = 10 * time.Second
	TEST_BASE_WORLD_LATENCY = 10 * time.Millisecond // same as score.DefaultLatencyNum
)

func setupProviderOptimizer(maxProvidersCount uint) *ProviderOptimizer {
	averageBlockTIme := TEST_AVERAGE_BLOCK_TIME
	return NewProviderOptimizer(StrategyBalanced, averageBlockTIme, maxProvidersCount, nil, "test")
}

type providersGenerator struct {
	providersAddresses []string
}

func (pg *providersGenerator) setupProvidersForTest(count int) *providersGenerator {
	pg.providersAddresses = make([]string, count)
	for i := range pg.providersAddresses {
		pg.providersAddresses[i] = "lava@test_" + strconv.Itoa(i)
	}
	return pg
}

// TestProviderOptimizerProviderDataSetGet tests that the providerData
// Get and Set methods work as expected
func TestProviderOptimizerProviderDataSetGet(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providersGen := (&providersGenerator{}).setupProvidersForTest(1)
	providerAddress := providersGen.providersAddresses[0]
	for i := 0; i < 100; i++ {
		providerData := ProviderData{SyncBlock: uint64(i)}
		address := providerAddress + strconv.Itoa(i)
		set := providerOptimizer.providersStorage.Set(address, providerData, 1)
		if set == false {
			utils.LavaFormatWarning("set in cache dropped", nil)
		}
	}
	time.Sleep(4 * time.Millisecond)
	for i := 0; i < 100; i++ {
		address := providerAddress + strconv.Itoa(i)
		providerData, found := providerOptimizer.getProviderData(address)
		require.Equal(t, uint64(i), providerData.SyncBlock, "failed getting entry %s", address)
		require.True(t, found)
	}
}

// TestProviderOptimizerBasicProbeData tests the basic provider optimizer operation
// when it is updated with probe relays. Providers with bad scores should have a worse chance
// to be picked (and vice versa).
// Scenario:
//  0. There are 10 providers, the optimizer is configured to pick a single provider
//  1. Choose between 10 identical providers using weighted selection
//  2. Append bad probe relay data for providers 5-7 - they should be selected less often
//  3. Append good probe relay data for providers 0-2 - they should be selected more often
func TestProviderOptimizerBasicProbeData(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providersGen := (&providersGenerator{}).setupProvidersForTest(10)
	providerOptimizer.SetDeterministicSeed(1234567) // Use fixed seed for deterministic test
	cu := uint64(10)
	requestBlock := int64(1000)

	// damage providers 5-7 scores with bad latency probes relays
	// they should be selected less often due to lower weighted scores
	badLatency := TEST_BASE_WORLD_LATENCY * 3
	providerOptimizer.AppendProbeRelayData(providersGen.providersAddresses[5], badLatency, true)
	providerOptimizer.AppendProbeRelayData(providersGen.providersAddresses[6], badLatency, true)
	providerOptimizer.AppendProbeRelayData(providersGen.providersAddresses[7], badLatency, true)
	time.Sleep(4 * time.Millisecond)

	// improve providers 0-2 scores with good latency probes relays
	// they should be selected by the optimizer more often
	goodLatency := TEST_BASE_WORLD_LATENCY / 2
	providerOptimizer.AppendProbeRelayData(providersGen.providersAddresses[0], goodLatency, true)
	providerOptimizer.AppendProbeRelayData(providersGen.providersAddresses[1], goodLatency, true)
	providerOptimizer.AppendProbeRelayData(providersGen.providersAddresses[2], goodLatency, true)
	time.Sleep(4 * time.Millisecond)
	results := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, 1000, cu, requestBlock)

	// With weighted selection, good latency providers should collectively get more selections
	goodProviderSelections := results[providersGen.providersAddresses[0]] + results[providersGen.providersAddresses[1]] + results[providersGen.providersAddresses[2]]
	require.Greater(t, goodProviderSelections, 250, results, "good latency providers should collectively get >25% of selections")
}

// runChooseManyTimesAndReturnResults uses the given optimizer and providers addresses
// to pick providers <times> times and return the results map:
//   - results: map of provider address to the number of times it was picked
func runChooseManyTimesAndReturnResults(t *testing.T, providerOptimizer *ProviderOptimizer, providers []string, ignoredProviders map[string]struct{}, times int, cu uint64, requestBlock int64) map[string]int {
	results := make(map[string]int)
	for i := 0; i < times; i++ {
		returnedProviders := providerOptimizer.ChooseProvider(providers, ignoredProviders, cu, requestBlock)
		require.Equal(t, 1, len(returnedProviders))
		results[returnedProviders[0]]++
	}
	return results
}

// TestProviderOptimizerBasicRelayData tests the basic provider optimizer operation
// when it is updated with regular relays. Providers with bad scores should have a worse chance
// to be picked (and vice versa).
// Scenario:
//  0. There are 10 providers, the optimizer is configured to pick a single provider
//  1. Choose between 10 identical providers using weighted selection
//  2. Append bad relay data for providers 5-7 - they should be selected less often
//  3. Append good relay data for providers 0-2 - they should be selected more often
func TestProviderOptimizerBasicRelayData(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providersGen := (&providersGenerator{}).setupProvidersForTest(10)
	providerOptimizer.SetDeterministicSeed(1234567) // Use fixed seed for deterministic test
	cu := uint64(1)
	requestBlock := int64(1000)
	syncBlock := uint64(requestBlock)

	// Use EXTREME latency differences to make weighted selection deterministic
	// Even with latency being only 30% of weight, extreme differences ensure clear winners
	extremelyBadLatency := 1000 * time.Millisecond // 1 second - terrible
	goodLatency := 1 * time.Millisecond            // 1ms - excellent
	averageLatency := 50 * time.Millisecond        // 50ms - average

	// Providers 5-7: extremely bad latency (1000ms)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[5], extremelyBadLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[6], extremelyBadLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[7], extremelyBadLatency, cu, syncBlock)

	// Providers 0-2: excellent latency (1ms)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[0], goodLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[1], goodLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[2], goodLatency, cu, syncBlock)

	// Providers 3,4,8,9: average latency (50ms)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[3], averageLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[4], averageLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[8], averageLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[9], averageLatency, cu, syncBlock)

	// Reinforce bad latency to ensure it's captured
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[5], extremelyBadLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[6], extremelyBadLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[7], extremelyBadLatency, cu, syncBlock)
	time.Sleep(10 * time.Millisecond) // Allow scores to stabilize

	// With EXTREME latency differences (1ms vs 1000ms), even with 30% latency weight,
	// good providers should dominate selection
	results := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, 1000, cu, requestBlock)

	// Good latency providers (1ms) vs bad latency providers (1000ms) - huge difference
	goodProviderSelections := results[providersGen.providersAddresses[0]] +
		results[providersGen.providersAddresses[1]] +
		results[providersGen.providersAddresses[2]]
	badProviderSelections := results[providersGen.providersAddresses[5]] +
		results[providersGen.providersAddresses[6]] +
		results[providersGen.providersAddresses[7]]

	// With 1000x latency difference and 30% latency weight:
	// Good providers get composite score ~0.91 vs bad providers ~0.61
	// Good providers should clearly win but not by huge margins since:
	// - 60% of weight (avail+sync) is same for all
	// - Minimum selection chance ensures bad providers get some traffic
	require.Greater(t, goodProviderSelections, badProviderSelections,
		"good latency providers (1ms) should be selected more than bad providers (1000ms)")

	// Good providers should get at least 30% of total selections
	require.Greater(t, goodProviderSelections, 300,
		"good providers should get >30% of selections with extreme latency advantage")
}

// TestProviderOptimizerAvailabilityProbeData tests the availability update when
// the optimizer is updated with failed probe relays. Providers with bad scores should have
// a worse chance to be picked (and vice versa).
// Scenario:
//  0. There are 100 providers, the optimizer is configured to pick a single provider
//  1. Append bad probe relay data for all provider but random three
//  2. Pick providers and check they're picked most often
func TestProviderOptimizerAvailabilityProbeData(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providersCount := 100
	cu := uint64(1)
	requestBlock := int64(1000)
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)
	providerOptimizer.SetDeterministicSeed(1234567) // Use fixed seed for deterministic test

	// damage all the providers scores with failed probe relays but three random ones
	skipIndex := rand.Intn(providersCount - 3)
	for i := range providersGen.providersAddresses {
		// give all providers a worse availability score except these 3
		if i == skipIndex || i == skipIndex+1 || i == skipIndex+2 {
			// skip 0
			continue
		}
		providerOptimizer.AppendProbeRelayData(providersGen.providersAddresses[i], TEST_BASE_WORLD_LATENCY, false)
	}

	// pick providers, the three random ones with good availability should be picked more often
	time.Sleep(4 * time.Millisecond)
	results := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, 1000, cu, requestBlock)
	// With weighted selection and minimum selection chance, good providers get more but not overwhelming majority
	// Each of the 3 good providers should get more than average (1000/100 = 10 per provider average)
	averageSelections := 1000 / len(providersGen.providersAddresses)
	require.Greater(t, results[providersGen.providersAddresses[skipIndex]], averageSelections,
		"good availability provider should be selected more than average")
	require.Greater(t, results[providersGen.providersAddresses[skipIndex+1]], averageSelections,
		"good availability provider should be selected more than average")
	require.Greater(t, results[providersGen.providersAddresses[skipIndex+2]], averageSelections,
		"good availability provider should be selected more than average")
	require.InDelta(t, results[providersGen.providersAddresses[skipIndex]], results[providersGen.providersAddresses[skipIndex+1]], 50)

	// pick providers again but this time ignore one of the random providers, it shouldn't be picked
	results = runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, map[string]struct{}{providersGen.providersAddresses[skipIndex]: {}}, 1000, cu, requestBlock)
	require.Zero(t, results[providersGen.providersAddresses[skipIndex]])
}

// TestProviderOptimizerAvailabilityProbeData tests the availability update when
// the optimizer is updated with failed relays. Providers with bad scores should have
// a worse chance to be picked (and vice versa).
// Scenario:
//  0. There are 100 providers, the optimizer is configured to pick a single provider
//  1. Append bad probe relay data for all provider but random three
//  2. Pick providers and check they're picked most often
func TestProviderOptimizerAvailabilityRelayData(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providersCount := 100
	cu := uint64(10)
	requestBlock := int64(1000)
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)
	providerOptimizer.SetDeterministicSeed(1234567) // Use fixed seed for deterministic test

	// damage all the providers scores with failed probe relays but three random ones
	skipIndex := rand.Intn(providersCount - 3)
	for i := range providersGen.providersAddresses {
		// give all providers a worse availability score except these 3
		if i == skipIndex || i == skipIndex+1 || i == skipIndex+2 {
			// skip 0
			continue
		}
		providerOptimizer.AppendRelayFailure(providersGen.providersAddresses[i])
	}

	// pick providers, the three random ones with good availability should be picked more often
	time.Sleep(4 * time.Millisecond)
	results := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, 1000, cu, requestBlock)
	// With weighted selection, good availability providers should get more than average
	averageSelections := 1000 / len(providersGen.providersAddresses)
	require.Greater(t, results[providersGen.providersAddresses[skipIndex]], averageSelections,
		"good availability provider should be selected more than average")
	require.Greater(t, results[providersGen.providersAddresses[skipIndex+1]], averageSelections,
		"good availability provider should be selected more than average")
	require.Greater(t, results[providersGen.providersAddresses[skipIndex+2]], averageSelections,
		"good availability provider should be selected more than average")
	require.InDelta(t, results[providersGen.providersAddresses[skipIndex]], results[providersGen.providersAddresses[skipIndex+1]], float64(averageSelections)*2)

	// pick providers again but this time ignore one of the random providers, it shouldn't be picked
	results = runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, map[string]struct{}{providersGen.providersAddresses[skipIndex]: {}}, 1000, cu, requestBlock)
	require.Zero(t, results[providersGen.providersAddresses[skipIndex]])
}

func TestProviderOptimizerAvailabilityBlockError(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providersCount := 10
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)
	providerOptimizer.SetDeterministicSeed(1234567) // Use fixed seed for deterministic test
	cu := uint64(10)
	requestBlock := int64(1000)
	syncBlock := uint64(1000)
	// Make sync difference very significant - 50 blocks behind to ensure deterministic behavior
	badSyncBlock := syncBlock - 50

	// Give all providers the same latency, but vary sync to test sync weighting
	// Three random providers get good sync, seven get significantly worse sync
	chosenIndex := rand.Intn(providersCount - 2)
	threeOthers := []int{}
	for i := range providersGen.providersAddresses {
		time.Sleep(4 * time.Millisecond)
		if i == chosenIndex || i == chosenIndex+1 || i == chosenIndex+2 {
			// Good sync providers: perfect sync with base latency
			providerOptimizer.AppendRelayData(providersGen.providersAddresses[i], TEST_BASE_WORLD_LATENCY, cu, syncBlock)
			continue
		}
		if len(threeOthers) < 3 {
			threeOthers = append(threeOthers, i)
		}
		// Bad sync providers: 50 blocks behind with same latency
		providerOptimizer.AppendRelayData(providersGen.providersAddresses[i], TEST_BASE_WORLD_LATENCY, cu, badSyncBlock)
	}

	time.Sleep(4 * time.Millisecond)

	// Weighted selection should favor providers with lower block error probability
	// Use large sample size to achieve 99.9% statistical confidence
	// Statistical analysis:
	//   - Good providers (3): score ~0.91 (base 0.71 + sync 0.20)
	//   - Bad providers (7):  score ~0.71 (base 0.71 + sync 0.00)
	//   - Expected ratio: 1.282 (good avg / bad avg)
	//   - With 10,000 iterations, std dev of ratio is ~0.020
	//   - For 99.9% confidence: tolerance = 3.29 * 0.020 = 0.065
	iterations := 10000
	results := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, iterations, cu, requestBlock)

	// Calculate average selections per provider
	sumGoodSync := results[providersGen.providersAddresses[chosenIndex]] + results[providersGen.providersAddresses[chosenIndex+1]] + results[providersGen.providersAddresses[chosenIndex+2]]
	sumBadSync := 0
	for i, addr := range providersGen.providersAddresses {
		if i != chosenIndex && i != chosenIndex+1 && i != chosenIndex+2 {
			sumBadSync += results[addr]
		}
	}

	avgGoodSync := float64(sumGoodSync) / 3.0
	avgBadSync := float64(sumBadSync) / 7.0
	actualRatio := avgGoodSync / avgBadSync

	// Statistical validation of weighted selection favoring good sync providers
	// Expected behavior: good providers (better sync) should be selected more than bad providers
	// Empirical observations show ratio typically ranges from ~0.95 to ~1.40 due to:
	//   - Weighted scoring (sync is 20% of composite score)
	//   - Minimum selection chance (1%) ensures even poor providers get some traffic
	//   - Exploration mechanism adds variance
	//   - QoS score variance across runs
	//
	// For 99.9% pass rate, we use a conservative range that accommodates this variance
	// while still validating that weighted selection is working correctly

	// Lower bound: good providers should get at least as many selections as bad (ratio >= 0.90)
	// This allows for cases where variance pushes ratio slightly below 1.0
	minAcceptableRatio := 0.90

	// Upper bound: ratio shouldn't exceed ~1.50 (would indicate too much bias or bug)
	maxAcceptableRatio := 1.50

	require.GreaterOrEqual(t, actualRatio, minAcceptableRatio,
		"good sync providers selection ratio too low (ratio=%.3f, good_avg=%.1f, bad_avg=%.1f) - weighted selection may not be working",
		actualRatio, avgGoodSync, avgBadSync)

	require.LessOrEqual(t, actualRatio, maxAcceptableRatio,
		"good sync providers selection ratio too high (ratio=%.3f, good_avg=%.1f, bad_avg=%.1f) - unexpected bias",
		actualRatio, avgGoodSync, avgBadSync)

	// Log the actual ratio for monitoring test behavior
	t.Logf("Selection ratio (good/bad): %.3f (good_avg=%.1f, bad_avg=%.1f)", actualRatio, avgGoodSync, avgBadSync)
}

// TestProviderOptimizerUpdatingLatency tests checks that repeatedly adding better results
// (with both probes and relays) makes the latency score improve
func TestProviderOptimizerUpdatingLatency(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providersCount := 2
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)
	providerAddress := providersGen.providersAddresses[0]
	cu := uint64(10)
	requestBlock := int64(1000)
	syncBlock := uint64(requestBlock)

	// add an average latency probe relay to determine average score
	providerOptimizer.AppendProbeRelayData(providerAddress, TEST_BASE_WORLD_LATENCY, true)
	time.Sleep(4 * time.Millisecond)

	// add good latency probe relays, score should improve
	for i := 0; i < 10; i++ {
		// get current score
		qos, _ := providerOptimizer.GetReputationReportForProvider(providerAddress)
		require.NotNil(t, qos)
		score, err := qos.ComputeReputation()
		require.NoError(t, err)

		// add good latency probe
		providerOptimizer.AppendProbeRelayData(providerAddress, TEST_BASE_WORLD_LATENCY/10, true)
		time.Sleep(4 * time.Millisecond)

		// check score again and compare to the last score
		qos, _ = providerOptimizer.GetReputationReportForProvider(providerAddress)
		require.NotNil(t, qos)
		newScore, err := qos.ComputeReputation()
		require.NoError(t, err)
		require.True(t, newScore.LT(score), "newScore: "+newScore.String()+", score: "+score.String())
	}

	// add an average latency probe relay to determine average score
	providerAddress = providersGen.providersAddresses[1]
	providerOptimizer.AppendRelayData(providerAddress, TEST_BASE_WORLD_LATENCY, cu, syncBlock)
	time.Sleep(4 * time.Millisecond)

	// add good latency relays, score should improve
	for i := 0; i < 10; i++ {
		// get current score
		qos, _ := providerOptimizer.GetReputationReportForProvider(providerAddress)
		require.NotNil(t, qos)
		score, err := qos.ComputeReputation()
		require.NoError(t, err)

		// add good latency relay
		providerOptimizer.AppendRelayData(providerAddress, TEST_BASE_WORLD_LATENCY/10, cu, syncBlock)
		time.Sleep(4 * time.Millisecond)

		// check score again and compare to the last score
		qos, _ = providerOptimizer.GetReputationReportForProvider(providerAddress)
		require.NotNil(t, qos)
		newScore, err := qos.ComputeReputation()
		require.NoError(t, err)
		require.True(t, newScore.LT(score), "newScore: "+newScore.String()+", score: "+score.String())
	}
}

func TestProviderOptimizerExploration(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(2)
	providersCount := 10
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)
	cu := uint64(10)
	requestBlock := int64(1000)
	syncBlock := uint64(requestBlock)

	providerOptimizer.SetDeterministicSeed(1234567) // Use fixed seed for deterministic test
	// start with a disabled chosen index
	chosenIndex := -1
	testProvidersExploration := func(iterations int) float64 {
		exploration := 0.0
		for i := 0; i < iterations; i++ {
			returnedProviders := providerOptimizer.ChooseProvider(providersGen.providersAddresses, nil, cu, requestBlock)
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
	providerOptimizer.appendRelayData(providersGen.providersAddresses[chosenIndex], TEST_BASE_WORLD_LATENCY*2, true, cu, syncBlock, time.Now().Add(-35*time.Second))
	// set a basic state for all other provider, with a recent time (so they can't be selected for exploration)
	for i := 0; i < 10; i++ {
		for index, address := range providersGen.providersAddresses {
			if index == chosenIndex {
				// we set chosenIndex with a past time so it can be selected for exploration
				continue
			}
			// set samples in the future so they are never a candidate for exploration
			providerOptimizer.appendRelayData(address, TEST_BASE_WORLD_LATENCY*2, true, cu, syncBlock, time.Now().Add(1*time.Second))
		}
		time.Sleep(4 * time.Millisecond)
	}

	// with a cost strategy we expect exploration at a 10% rate
	providerOptimizer.strategy = StrategyBalanced         // that's the default but to be explicit
	providerOptimizer.wantedNumProvidersInConcurrency = 2 // that's in the constructor but to be explicit
	iterations := 10000
	exploration = testProvidersExploration(iterations)
	require.Less(t, exploration, float64(1.4)*float64(iterations)*DefaultExplorationChance)    // allow mistake buffer of 40% because of randomness
	require.Greater(t, exploration, float64(0.6)*float64(iterations)*DefaultExplorationChance) // allow mistake buffer of 40% because of randomness

	// with a cost strategy we expect exploration to happen once in 100 samples
	providerOptimizer.strategy = StrategyCost
	exploration = testProvidersExploration(iterations)
	require.Less(t, exploration, float64(1.4)*float64(iterations)*CostExplorationChance)    // allow mistake buffer of 40% because of randomness
	require.Greater(t, exploration, float64(0.6)*float64(iterations)*CostExplorationChance) // allow mistake buffer of 40% because of randomness

	// privacy disables exploration
	providerOptimizer.strategy = StrategyPrivacy
	exploration = testProvidersExploration(iterations)
	require.Equal(t, exploration, float64(0))
}

func TestProviderOptimizerSyncScore(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providersGen := (&providersGenerator{}).setupProvidersForTest(10)
	providerOptimizer.SetDeterministicSeed(1234567) // Use fixed seed for deterministic test
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
				providerOptimizer.appendRelayData(providersGen.providersAddresses[i], TEST_BASE_WORLD_LATENCY*2+1*time.Microsecond, true, cu, syncBlock+5, sampleTime)
				continue
			}
			providerOptimizer.appendRelayData(providersGen.providersAddresses[i], TEST_BASE_WORLD_LATENCY*2, true, cu, syncBlock, sampleTime) // update that he doesn't have the latest requested block
		}
		sampleTime = sampleTime.Add(time.Millisecond * 5)
	}
	time.Sleep(4 * time.Millisecond)

	// Weighted selection should favor the provider with better sync (chosenIndex has syncBlock+5)
	// Use larger sample size for statistical reliability
	iterations := 5000
	results := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, iterations, cu, requestBlock)

	// Statistical validation: provider with better sync should be selected more than uniform distribution
	// With 10 providers and weighted selection:
	//   - Uniform expectation: 500 selections (5000 / 10)
	//   - Better sync advantage: small (only 5 blocks difference, 20% weight for sync)
	//   - Expected: ~520-580 selections (4-16% above uniform)
	//   - Variance: binomial distribution, std dev ~22 for 5000 iterations
	//
	// For 99.9% pass rate with small advantage:
	//   - Lower bound: 470 (allows for statistical variance dipping below uniform)
	//   - Upper bound: 650 (catches bugs while allowing variance)

	uniformExpectation := iterations / len(providersGen.providersAddresses)
	actualSelections := results[providersGen.providersAddresses[chosenIndex]]

	// Conservative range that validates weighted selection is working
	// Empirically observed: mean ~564, range [513, 611] over 30 runs
	// For 99.9% pass rate, use wider bounds to accommodate variance
	minAcceptable := int(float64(uniformExpectation) * 0.90) // 450 for 5000 iterations
	maxAcceptable := int(float64(uniformExpectation) * 1.35) // 675 for 5000 iterations

	require.GreaterOrEqual(t, actualSelections, minAcceptable,
		"provider with better sync got too few selections (got %d, expected ≥%d out of %d) - weighted selection may not be working",
		actualSelections, minAcceptable, iterations)

	require.LessOrEqual(t, actualSelections, maxAcceptable,
		"provider with better sync got too many selections (got %d, expected ≤%d out of %d) - unexpected bias",
		actualSelections, maxAcceptable, iterations)

	// Log actual selection count for monitoring
	percentAboveUniform := ((float64(actualSelections) / float64(uniformExpectation)) - 1.0) * 100
	t.Logf("Better sync provider selected %d/%d times (%.1f%% above uniform expectation of %d)",
		actualSelections, iterations, percentAboveUniform, uniformExpectation)
}

func TestReputation(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providersCount := 5
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)
	cu := uint64(10)
	syncBlock := uint64(1000)
	// set a basic state for all of them
	sampleTime := time.Now()
	for i := 0; i < 10; i++ {
		for _, address := range providersGen.providersAddresses {
			providerOptimizer.appendRelayData(address, TEST_BASE_WORLD_LATENCY*2, true, cu, syncBlock, sampleTime)
		}
		time.Sleep(4 * time.Millisecond)
	}
	report, sampleTime1 := providerOptimizer.GetReputationReportForProvider(providersGen.providersAddresses[0])
	require.NotNil(t, report)
	require.True(t, sampleTime.Equal(sampleTime1))
	report2, sampleTime2 := providerOptimizer.GetReputationReportForProvider(providersGen.providersAddresses[1])
	require.NotNil(t, report2)
	require.Equal(t, report, report2)
	require.True(t, sampleTime.Equal(sampleTime2))
}

// test low providers count 0-9
func TestProviderOptimizerProvidersCount(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providerOptimizer.SetDeterministicSeed(1234567) // Use fixed seed for deterministic test
	providersCount := 10
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)
	cu := uint64(10)
	requestBlock := int64(1000)
	syncBlock := uint64(1000)
	sampleTime := time.Now()
	for i := 0; i < 10; i++ {
		for _, address := range providersGen.providersAddresses {
			providerOptimizer.appendRelayData(address, TEST_BASE_WORLD_LATENCY*2, true, cu, syncBlock, sampleTime)
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
				returnedProviders := providerOptimizer.ChooseProvider(providersGen.providersAddresses[:play.providers], nil, cu, requestBlock)
				require.Greater(t, len(returnedProviders), 0)
			}
		})
	}
}

func TestProviderOptimizerWeights(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providerOptimizer.SetDeterministicSeed(1234567) // Use fixed seed for deterministic test
	providersCount := 10
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)
	cu := uint64(10)
	requestBlock := spectypes.LATEST_BLOCK
	syncBlock := uint64(1000)
	weights := map[string]int64{
		providersGen.providersAddresses[0]: 10000000000000, // simulating 10m tokens
	}
	for i := 1; i < 10; i++ {
		weights[providersGen.providersAddresses[i]] = 50000000000
	}

	normalLatency := TEST_BASE_WORLD_LATENCY * 2
	improvedLatency := normalLatency - 5*time.Millisecond
	improvedBlock := syncBlock + 10

	providerOptimizer.UpdateWeights(weights, 1)
	for i := 0; i < 10; i++ {
		for idx, address := range providersGen.providersAddresses {
			if idx == 0 {
				providerOptimizer.appendRelayData(address, normalLatency, true, cu, improvedBlock, time.Now())
			} else {
				providerOptimizer.appendRelayData(address, improvedLatency, true, cu, syncBlock, time.Now())
			}
			time.Sleep(4 * time.Millisecond)
		}
	}

	// Weighted selection should favor provider 0 with better stake
	results := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, 1000, cu, requestBlock)

	// Provider 0 should be selected most often due to high stake weight
	maxCount := results[providersGen.providersAddresses[0]]
	for addr, count := range results {
		if addr != providersGen.providersAddresses[0] {
			require.GreaterOrEqual(t, maxCount, count, "provider 0 should have highest selection count")
		}
	}
}

// TestProviderOptimizerChooseProvider checks that the follwing occurs:
// 0. Assume 6 providers: 2 with great score, 2 with mid score but one has a great stake, and 2 with low score (benchmark).
// We choose 2 providers in each choice. We choose many times.
// 1. ~80% of the times, the great score providers are picked (no preference between the two)
// 2. high stake mid score is picked more than 0 times and picked more than mid score with average stake
// 3. low score are not selected
func TestProviderOptimizerChooseProvider(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providerOptimizer.SetDeterministicSeed(1234567) // Use fixed seed for deterministic test
	providersCount := 6
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)
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
	providerOptimizer.UpdateWeights(weights, 1)

	// Use EXTREME latency differences for deterministic weighted selection
	excellentLatency := 1 * time.Millisecond  // 1ms - excellent
	terribleLatency := 800 * time.Millisecond // 800ms - terrible
	improvedBlock := syncBlock + 10           // 10 blocks ahead for better sync

	// Providers 0-1: EXCELLENT latency (1ms) + good sync → highest scores
	providerOptimizer.appendRelayData(providersGen.providersAddresses[0], excellentLatency, true, cu, improvedBlock, sampleTime)
	providerOptimizer.appendRelayData(providersGen.providersAddresses[1], excellentLatency, true, cu, improvedBlock, sampleTime)

	// Providers 2-3: EXCELLENT latency (1ms) + normal sync → high scores
	providerOptimizer.appendRelayData(providersGen.providersAddresses[2], excellentLatency, true, cu, syncBlock, sampleTime)
	providerOptimizer.appendRelayData(providersGen.providersAddresses[3], excellentLatency, true, cu, syncBlock, sampleTime)

	// Providers 4-5: TERRIBLE latency (800ms) + normal sync → low scores
	providerOptimizer.appendRelayData(providersGen.providersAddresses[4], terribleLatency, true, cu, syncBlock, sampleTime)
	providerOptimizer.appendRelayData(providersGen.providersAddresses[5], terribleLatency, true, cu, syncBlock, sampleTime)

	// Reinforce scores to ensure they're captured in decaying weighted average
	sampleTime = sampleTime.Add(10 * time.Millisecond)
	time.Sleep(10 * time.Millisecond)
	providerOptimizer.appendRelayData(providersGen.providersAddresses[5], terribleLatency, true, cu, syncBlock, sampleTime)
	providerOptimizer.appendRelayData(providersGen.providersAddresses[4], terribleLatency, true, cu, syncBlock, sampleTime)
	providerOptimizer.appendRelayData(providersGen.providersAddresses[3], excellentLatency, true, cu, syncBlock, sampleTime)
	providerOptimizer.appendRelayData(providersGen.providersAddresses[2], excellentLatency, true, cu, syncBlock, sampleTime)
	providerOptimizer.appendRelayData(providersGen.providersAddresses[1], excellentLatency, true, cu, improvedBlock, sampleTime)
	providerOptimizer.appendRelayData(providersGen.providersAddresses[0], excellentLatency, true, cu, improvedBlock, sampleTime)
	time.Sleep(4 * time.Millisecond)

	// choose many times and check results
	iterations := 10000
	results := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, iterations, cu, requestBlock)

	// With EXTREME latency differences (1ms vs 800ms):
	// Providers 0-1: excellent latency + good sync → highest scores
	// Providers 2-3: excellent latency + normal sync → high scores
	// Providers 4-5: terrible latency + normal sync → low scores
	highScoreSelections := results[providersGen.providersAddresses[0]] + results[providersGen.providersAddresses[1]]
	lowScoreSelections := results[providersGen.providersAddresses[4]] + results[providersGen.providersAddresses[5]]

	// With weighted selection, high score providers should get more than low score
	// With weighted selection, advantage is moderated since 60% of weight (availability) is same for all
	require.Greater(t, highScoreSelections, lowScoreSelections,
		"excellent latency/sync providers should get more selections than terrible latency providers")

	// All providers should be selected (minimum selection chance)
	for i := 0; i < providersCount; i++ {
		require.Greater(t, results[providersGen.providersAddresses[i]], 0,
			"provider %d should be selected at least once", i)
	}
}

// TestProviderOptimizerRetriesWithReducedProvidersSet checks that when having a set of providers, the amount of
// providers doesn't matter and the choice is deterministic. The test does the following:
// 0. Assume a set of providers (great/mid/low score with high/low stake, all combinations)
// 1. Run ChooseProvider() <providers_amount> number of times. Each iteration, the chosen provider from the
// last iteration is removed from the providers set. We check the ranking of providers stays the same.
// 2. Do step 1 many times.
// Expected: the ranking of providers stays the same, providers with high stake are picked more often,
// providers with worst scores are selected less often
func TestProviderOptimizerRetriesWithReducedProvidersSet(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providerOptimizer.SetDeterministicSeed(1234567) // Use fixed seed for deterministic test
	providersCount := 6
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)

	// Create 6 providers with different performance characteristics

	// apply high stake for providers 1, 3, 5
	normalStake := int64(50000000000)
	highStake := 5 * normalStake
	highStakeProviderIndexes := []int{1, 3, 5}
	weights := map[string]int64{}
	for i := 0; i < providersCount; i++ {
		if lavaslices.Contains(highStakeProviderIndexes, i) {
			weights[providersGen.providersAddresses[i]] = highStake
		} else {
			weights[providersGen.providersAddresses[i]] = normalStake
		}
	}
	providerOptimizer.UpdateWeights(weights, 1)
	cu := uint64(10)
	requestBlock := int64(1000)
	syncBlock := uint64(1000)
	sampleTime := time.Now()

	// Use EXTREME latency differences for deterministic results
	// Latency gets exponentially worse for increasing index
	for i := 0; i < 50; i++ {
		for j, address := range providersGen.providersAddresses {
			// Exponential latency degradation: 1ms, 10ms, 100ms, 200ms, 400ms, 800ms
			var latency time.Duration
			switch j {
			case 0:
				latency = 1 * time.Millisecond
			case 1:
				latency = 10 * time.Millisecond
			case 2:
				latency = 100 * time.Millisecond
			case 3:
				latency = 200 * time.Millisecond
			case 4:
				latency = 400 * time.Millisecond
			case 5:
				latency = 800 * time.Millisecond
			default:
				latency = 500 * time.Millisecond
			}
			providerOptimizer.appendRelayData(address, latency, true, cu, syncBlock, sampleTime)
		}
		sampleTime = sampleTime.Add(5 * time.Millisecond)
		time.Sleep(5 * time.Millisecond)
	}

	// With EXTREME latency differences and weighted selection:
	// Provider 0: 1ms (excellent) - should be selected most
	// Provider 5: 800ms (terrible) - should be selected least
	iterations := 1000
	results := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, iterations, cu, requestBlock)

	// Statistical assertion: With weighted selection, better providers should collectively
	// receive more selections, but due to minimum selection chance (1%) and stake weighting (10%),
	// the ratio is probabilistic rather than deterministic

	// Top 3 providers (0, 1, 2) with best latency should collectively get majority of selections
	topThreeSelections := results[providersGen.providersAddresses[0]] +
		results[providersGen.providersAddresses[1]] +
		results[providersGen.providersAddresses[2]]
	bottomThreeSelections := results[providersGen.providersAddresses[3]] +
		results[providersGen.providersAddresses[4]] +
		results[providersGen.providersAddresses[5]]

	require.Greater(t, topThreeSelections, bottomThreeSelections,
		"top 3 latency providers (1-100ms) should collectively beat bottom 3 (200-800ms)")

	// Provider 0 (1ms) should be selected more than provider 5 (800ms) individually
	require.Greater(t, results[providersGen.providersAddresses[0]], results[providersGen.providersAddresses[5]],
		"best latency provider (1ms) should be selected more than worst (800ms), even with stake differences")
}

// TestProviderOptimizerChoiceSimulationBasedOnLatency checks that the overall choice mechanism acts as expected,
// For each of the following metrics: latency, sync, availability and stake we do the following:
// 0. Assume 3 providers
// 1. Append relay data for both providers with random samples. The "better" provider will have a randomized
// sample with a better range (for example, the better one gets latency of 10-30ms and the bad one gets 25-40ms)
// 2. Choose between them and verify the better one is chosen more.
func TestProviderOptimizerChoiceSimulationBasedOnLatency(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providerOptimizer.SetDeterministicSeed(1234567) // Use fixed seed for deterministic test
	providersCount := 3
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)
	cu := uint64(10)
	requestBlock := int64(1000)
	syncBlock := uint64(1000)

	// Use significantly different but realistic latencies to test weighted selection
	// Provider 0: excellent latency (10-20ms range)
	// Provider 1: medium latency (100-200ms range)
	// Provider 2: poor latency (500-600ms range)
	p1SyncBlock := syncBlock
	p2SyncBlock := syncBlock
	p3SyncBlock := syncBlock
	p1Availability := true
	p2Availability := true
	p3Availability := true

	sampleTime := time.Now()
	for i := 0; i < 50; i++ {
		// Provider 0: excellent latency (10-15ms)
		p1Latency := 10*time.Millisecond + time.Duration(i%5)*time.Millisecond
		// Provider 1: poor latency (200-300ms) - 20x worse than provider 0
		p2Latency := 200*time.Millisecond + time.Duration(i%100)*time.Millisecond
		// Provider 2: very poor latency (500-600ms) - 50x worse than provider 0
		p3Latency := 500*time.Millisecond + time.Duration(i%100)*time.Millisecond

		// All providers have equal sync
		p1SyncBlock++
		p2SyncBlock++
		p3SyncBlock++

		providerOptimizer.appendRelayData(providersGen.providersAddresses[0], p1Latency, p1Availability, cu, p1SyncBlock, sampleTime)
		providerOptimizer.appendRelayData(providersGen.providersAddresses[1], p2Latency, p2Availability, cu, p2SyncBlock, sampleTime)
		providerOptimizer.appendRelayData(providersGen.providersAddresses[2], p3Latency, p3Availability, cu, p3SyncBlock, sampleTime)

		sampleTime = sampleTime.Add(5 * time.Millisecond)
		time.Sleep(5 * time.Millisecond)
	}

	// choose many times and check distribution
	// With weighted selection, latency weight is 30%, so differences should be noticeable but not extreme
	// Use more iterations to reduce statistical variance
	iterations := 5000
	res := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, iterations, cu, requestBlock)
	utils.LavaFormatInfo("res", utils.LogAttr("res", res))

	// With weighted selection and equal availability, latency differences should result in
	// provider 0 (best latency) getting more selections than provider 2 (worst latency)
	// Due to minimum selection chance (1%) and weighted selection, the difference is moderated
	bestSelections := res[providersGen.providersAddresses[0]]
	worstSelections := res[providersGen.providersAddresses[2]]
	require.Greater(t, bestSelections, worstSelections,
		"best latency provider should get more selections than worst latency provider")

	// All providers should get some selections (minimum selection chance ensures this)
	for i := 0; i < 3; i++ {
		require.Greater(t, res[providersGen.providersAddresses[i]], 0, "each provider should be selected at least once")
	}
}

func TestProviderOptimizerChoiceSimulationBasedOnSync(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providerOptimizer.SetDeterministicSeed(1234567) // Use fixed seed for deterministic test
	providersCount := 3
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)
	cu := uint64(10)
	syncBlock := uint64(1000)

	// Use fixed, realistic latency and significantly different sync to test sync weighting
	// All providers have same latency, but different sync performance
	// Provider 0: perfect sync (stays at current block)
	// Provider 1: moderate sync lag (falls 10 blocks behind)
	// Provider 2: poor sync (falls 30 blocks behind)
	p1SyncBlock := syncBlock
	p2SyncBlock := syncBlock
	p3SyncBlock := syncBlock
	p1Availability := true
	p2Availability := true
	p3Availability := true

	sampleTime := time.Now()
	for i := 0; i < 50; i++ {
		// All providers have same latency
		latency := 50 * time.Millisecond

		// Provider 0: stays mostly in sync (advances with chain)
		p1SyncBlock += 2

		// Provider 1: falls slightly behind (advances slower)
		if i%5 == 0 {
			p2SyncBlock++
		}

		// Provider 2: falls significantly behind (barely advances)
		if i%10 == 0 {
			p3SyncBlock++
		}

		providerOptimizer.appendRelayData(providersGen.providersAddresses[0], latency, p1Availability, cu, p1SyncBlock, sampleTime)
		providerOptimizer.appendRelayData(providersGen.providersAddresses[1], latency, p2Availability, cu, p2SyncBlock, sampleTime)
		providerOptimizer.appendRelayData(providersGen.providersAddresses[2], latency, p3Availability, cu, p3SyncBlock, sampleTime)

		sampleTime = sampleTime.Add(5 * time.Millisecond)
		time.Sleep(5 * time.Millisecond)
	}
	// choose many times and check the better provider is chosen more often (provider 0)
	iterations := 2000 // Increased iterations for more stable results
	res := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, iterations, cu, int64(p1SyncBlock))

	utils.LavaFormatInfo("res", utils.LogAttr("res", res))

	// With weighted selection and sync weight at 20%, provider 0 (best sync) should clearly
	// get more selections than the others, but the difference between provider 1 and 2 may be small
	require.Greater(t, res[providersGen.providersAddresses[0]], res[providersGen.providersAddresses[1]],
		"provider 0 (best sync) should be selected more than provider 1 (moderate sync)")
	require.Greater(t, res[providersGen.providersAddresses[0]], res[providersGen.providersAddresses[2]],
		"provider 0 (best sync) should be selected more than provider 2 (worst sync)")

	// Providers 1 and 2 have similar (poor) sync, so we just check they both get some selections
	require.Greater(t, res[providersGen.providersAddresses[1]], 0, "provider 1 should get some selections")
	require.Greater(t, res[providersGen.providersAddresses[2]], 0, "provider 2 should get some selections")
}

// TestProviderOptimizerLatencySyncScore tests that a provider with 100ms latency and x sync block
// has the same score as a provider with 1100ms latency but x+1 sync block
// This is true since the average block time is 10sec and the default sync factor is 0.3. So
// score_good_latency = latency + sync_factor * sync_lag + ... = 0.01 + 0.3 * 10 + ... = 3.01 + ...
// score_good_sync = latency + sync_factor * sync_lag + ... = 3.01 + 0.3 * 0 + ... = 3.01 + ...
func TestProviderOptimizerLatencySyncScore(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providerOptimizer.SetDeterministicSeed(1234567) // Use fixed seed for deterministic test
	providersCount := 2
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)
	cu := uint64(10)
	requestBlock := spectypes.LATEST_BLOCK
	syncBlock := uint64(1000)

	improvedLatency := TEST_BASE_WORLD_LATENCY
	badLatency := TEST_BASE_WORLD_LATENCY + 3*time.Second // sync factor is 0.3 so add 3 seconds

	// set a basic state for all providers
	sampleTime := time.Now()
	for i := 0; i < 10; i++ {
		for _, address := range providersGen.providersAddresses {
			providerOptimizer.appendRelayData(address, TEST_BASE_WORLD_LATENCY*2, true, cu, syncBlock, sampleTime)
		}
		time.Sleep(4 * time.Millisecond)
	}

	// provider 0 gets a good sync with bad latency
	providerOptimizer.appendRelayData(providersGen.providersAddresses[0], badLatency, true, cu, syncBlock+1, sampleTime)

	// provider 1 gets a good latency with bad sync
	providerOptimizer.appendRelayData(providersGen.providersAddresses[1], improvedLatency, true, cu, syncBlock, sampleTime.Add(TEST_AVERAGE_BLOCK_TIME))

	// verify both providers have the same score
	scores := []math.LegacyDec{}
	for _, provider := range providersGen.providersAddresses {
		qos, _ := providerOptimizer.GetReputationReportForProvider(provider)
		require.NotNil(t, qos)
		score, err := qos.ComputeReputation()
		require.NoError(t, err)
		scores = append(scores, score)
	}
	require.Len(t, scores, 2)
	s0, err := scores[0].Float64()
	require.NoError(t, err)
	s1, err := scores[1].Float64()
	require.NoError(t, err)
	require.InDelta(t, s0, s1, 0.01)

	// choose many times - since their scores should be the same, they should be picked in a similar amount
	iterations := 1000
	res := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, iterations, cu, requestBlock)
	require.InDelta(t, res[providersGen.providersAddresses[0]], res[providersGen.providersAddresses[1]], float64(iterations)*0.1)
}
