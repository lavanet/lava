package provideroptimizer

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/lavanet/lava/v4/protocol/metrics"
	"github.com/lavanet/lava/v4/utils"
	"github.com/lavanet/lava/v4/utils/rand"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
	"github.com/stretchr/testify/require"
)

const (
	TEST_AVERAGE_BLOCK_TIME = 10 * time.Second
	TEST_BASE_WORLD_LATENCY = 150 * time.Millisecond
)

type providerOptimizerSyncCache struct {
	value map[interface{}]interface{}
	lock  sync.RWMutex
}

func (posc *providerOptimizerSyncCache) Get(key interface{}) (interface{}, bool) {
	posc.lock.RLock()
	defer posc.lock.RUnlock()
	ret, ok := posc.value[key]
	return ret, ok
}

func (posc *providerOptimizerSyncCache) Set(key, value interface{}, cost int64) bool {
	posc.lock.Lock()
	defer posc.lock.Unlock()
	posc.value[key] = value
	return true
}

func setupProviderOptimizer(maxProvidersCount int) *ProviderOptimizer {
	averageBlockTIme := TEST_AVERAGE_BLOCK_TIME
	baseWorldLatency := TEST_BASE_WORLD_LATENCY
	return NewProviderOptimizer(STRATEGY_BALANCED, averageBlockTIme, baseWorldLatency, uint(maxProvidersCount), nil, "dontcare")
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

func TestProbabilitiesCalculations(t *testing.T) {
	value := CumulativeProbabilityFunctionForPoissonDist(1, 10)
	value2 := CumulativeProbabilityFunctionForPoissonDist(10, 10)
	require.Greater(t, value2, value)

	playbook := []struct {
		name                           string
		blockGap                       uint64
		averageBlockTime               time.Duration
		timeHas                        time.Duration
		expectedProbabilityHigherLimit float64
		expectedProbabilityLowerLimit  float64
	}{
		{
			name:                           "one",
			blockGap:                       1,
			averageBlockTime:               6 * time.Second,
			timeHas:                        25 * time.Second,
			expectedProbabilityHigherLimit: 0.3,
			expectedProbabilityLowerLimit:  0,
		},
		{
			name:                           "five",
			blockGap:                       5,
			averageBlockTime:               6 * time.Second,
			timeHas:                        6 * time.Second,
			expectedProbabilityHigherLimit: 1,
			expectedProbabilityLowerLimit:  0.7,
		},
		{
			name:                           "tight",
			blockGap:                       5,
			averageBlockTime:               6 * time.Second,
			timeHas:                        30 * time.Second,
			expectedProbabilityHigherLimit: 0.5,
			expectedProbabilityLowerLimit:  0.4,
		},
		{
			name:                           "almost there",
			blockGap:                       1,
			averageBlockTime:               6 * time.Second,
			timeHas:                        6 * time.Second,
			expectedProbabilityHigherLimit: 0.4,
			expectedProbabilityLowerLimit:  0.3,
		},
	}
	for _, tt := range playbook {
		t.Run(tt.name, func(t *testing.T) {
			eventRate := tt.timeHas.Seconds() / tt.averageBlockTime.Seconds()
			probabilityBlockError := CumulativeProbabilityFunctionForPoissonDist(tt.blockGap-1, eventRate)
			require.LessOrEqual(t, probabilityBlockError, tt.expectedProbabilityHigherLimit)
			require.GreaterOrEqual(t, probabilityBlockError, tt.expectedProbabilityLowerLimit)
		})
	}
}

func TestProviderOptimizerSetGet(t *testing.T) {
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

func TestProviderOptimizerBasic(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providersGen := (&providersGenerator{}).setupProvidersForTest(10)
	rand.InitRandomSeed()

	requestCU := uint64(10)
	requestBlock := int64(1000)

	returnedProviders, tier := providerOptimizer.ChooseProvider(providersGen.providersAddresses, nil, requestCU, requestBlock)
	require.Equal(t, 1, len(returnedProviders))
	require.NotEqual(t, 4, tier)
	// damage their chance to be selected by placing them in the worst tier
	providerOptimizer.AppendProbeRelayData(providersGen.providersAddresses[5], TEST_BASE_WORLD_LATENCY*3, true)
	providerOptimizer.AppendProbeRelayData(providersGen.providersAddresses[6], TEST_BASE_WORLD_LATENCY*3, true)
	providerOptimizer.AppendProbeRelayData(providersGen.providersAddresses[7], TEST_BASE_WORLD_LATENCY*3, true)
	time.Sleep(4 * time.Millisecond)
	returnedProviders, _ = providerOptimizer.ChooseProvider(providersGen.providersAddresses, nil, requestCU, requestBlock)
	require.Equal(t, 1, len(returnedProviders))
	require.NotEqual(t, 4, tier)
	require.NotEqual(t, returnedProviders[0], providersGen.providersAddresses[5]) // we shouldn't pick the worst provider
	require.NotEqual(t, returnedProviders[0], providersGen.providersAddresses[6]) // we shouldn't pick the worst provider
	require.NotEqual(t, returnedProviders[0], providersGen.providersAddresses[7]) // we shouldn't pick the worst provider
	// improve selection chance by placing them in the top tier
	providerOptimizer.AppendProbeRelayData(providersGen.providersAddresses[0], TEST_BASE_WORLD_LATENCY/2, true)
	providerOptimizer.AppendProbeRelayData(providersGen.providersAddresses[1], TEST_BASE_WORLD_LATENCY/2, true)
	providerOptimizer.AppendProbeRelayData(providersGen.providersAddresses[2], TEST_BASE_WORLD_LATENCY/2, true)
	time.Sleep(4 * time.Millisecond)
	results, tierResults := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, requestCU, requestBlock, 1000)
	require.Greater(t, tierResults[0], 650, tierResults) // we should pick the best tier most often
	// out of 10 providers, and with 3 in the top tier we should pick 0 around a third of that
	require.Greater(t, results[providersGen.providersAddresses[0]], 250, results) // we should pick the best tier most often
}

func runChooseManyTimesAndReturnResults(t *testing.T, providerOptimizer *ProviderOptimizer, providers []string, ignoredProviders map[string]struct{}, requestCU uint64, requestBlock int64, times int) (map[string]int, map[int]int) {
	tierResults := make(map[int]int)
	results := make(map[string]int)
	for i := 0; i < times; i++ {
		returnedProviders, tier := providerOptimizer.ChooseProvider(providers, ignoredProviders, requestCU, requestBlock)
		require.Equal(t, 1, len(returnedProviders))
		results[returnedProviders[0]]++
		tierResults[tier]++
	}
	return results, tierResults
}

func TestProviderOptimizerBasicRelayData(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providersGen := (&providersGenerator{}).setupProvidersForTest(10)
	rand.InitRandomSeed()
	requestCU := uint64(1)
	requestBlock := int64(1000)

	syncBlock := uint64(requestBlock)

	providerOptimizer.AppendRelayData(providersGen.providersAddresses[5], TEST_BASE_WORLD_LATENCY*4, false, requestCU, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[6], TEST_BASE_WORLD_LATENCY*4, false, requestCU, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[7], TEST_BASE_WORLD_LATENCY*4, false, requestCU, syncBlock)
	time.Sleep(4 * time.Millisecond)
	returnedProviders, tier := providerOptimizer.ChooseProvider(providersGen.providersAddresses, nil, requestCU, requestBlock)
	require.Equal(t, 1, len(returnedProviders))
	// we shouldn't pick the low tier providers
	require.NotEqual(t, tier, 3)
	require.NotEqual(t, returnedProviders[0], providersGen.providersAddresses[5], tier)
	require.NotEqual(t, returnedProviders[0], providersGen.providersAddresses[6], tier)
	require.NotEqual(t, returnedProviders[0], providersGen.providersAddresses[7], tier)

	providerOptimizer.AppendRelayData(providersGen.providersAddresses[0], TEST_BASE_WORLD_LATENCY/4, false, requestCU, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[1], TEST_BASE_WORLD_LATENCY/4, false, requestCU, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[2], TEST_BASE_WORLD_LATENCY/4, false, requestCU, syncBlock)
	time.Sleep(4 * time.Millisecond)
	results, tierResults := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, requestCU, requestBlock, 1000)

	require.Zero(t, results[providersGen.providersAddresses[5]])
	require.Zero(t, results[providersGen.providersAddresses[6]])
	require.Zero(t, results[providersGen.providersAddresses[7]])

	require.Greater(t, tierResults[0], 650, tierResults) // we should pick the best tier most often
	// out of 10 providers, and with 3 in the top tier we should pick 0 around a third of that
	require.Greater(t, results[providersGen.providersAddresses[0]], 250, results) // we should pick the best tier most often
}

func TestProviderOptimizerAvailability(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providersCount := 100
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)
	rand.InitRandomSeed()
	requestCU := uint64(10)
	requestBlock := int64(1000)

	skipIndex := rand.Intn(providersCount - 3)
	providerOptimizer.OptimizerNumTiers = 33 // set many tiers so good providers can stand out in the test
	for i := range providersGen.providersAddresses {
		// give all providers a worse availability score except these 3
		if i == skipIndex || i == skipIndex+1 || i == skipIndex+2 {
			// skip 0
			continue
		}
		providerOptimizer.AppendProbeRelayData(providersGen.providersAddresses[i], TEST_BASE_WORLD_LATENCY, false)
	}
	time.Sleep(4 * time.Millisecond)
	results, tierResults := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, requestCU, requestBlock, 1000)
	require.Greater(t, tierResults[0], 300, tierResults) // 0.42 chance for top tier due to the algorithm to rebalance chances
	require.Greater(t, results[providersGen.providersAddresses[skipIndex]]+results[providersGen.providersAddresses[skipIndex+1]]+results[providersGen.providersAddresses[skipIndex+2]], 280)
	require.InDelta(t, results[providersGen.providersAddresses[skipIndex]], results[providersGen.providersAddresses[skipIndex+1]], 50)
	results, _ = runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, map[string]struct{}{providersGen.providersAddresses[skipIndex]: {}}, requestCU, requestBlock, 1000)
	require.Zero(t, results[providersGen.providersAddresses[skipIndex]])
}

func TestProviderOptimizerAvailabilityRelayData(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providersCount := 100
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)
	rand.InitRandomSeed()
	requestCU := uint64(10)
	requestBlock := int64(1000)

	skipIndex := rand.Intn(providersCount - 3)
	providerOptimizer.OptimizerNumTiers = 33 // set many tiers so good providers can stand out in the test
	for i := range providersGen.providersAddresses {
		// give all providers a worse availability score except these 3
		if i == skipIndex || i == skipIndex+1 || i == skipIndex+2 {
			// skip 0
			continue
		}
		providerOptimizer.AppendRelayFailure(providersGen.providersAddresses[i])
	}
	time.Sleep(4 * time.Millisecond)
	results, tierResults := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, requestCU, requestBlock, 1000)
	require.Greater(t, tierResults[0], 300, tierResults) // 0.42 chance for top tier due to the algorithm to rebalance chances
	require.Greater(t, results[providersGen.providersAddresses[skipIndex]]+results[providersGen.providersAddresses[skipIndex+1]]+results[providersGen.providersAddresses[skipIndex+2]], 270)
	require.InDelta(t, results[providersGen.providersAddresses[skipIndex]], results[providersGen.providersAddresses[skipIndex+1]], 50)
	results, _ = runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, map[string]struct{}{providersGen.providersAddresses[skipIndex]: {}}, requestCU, requestBlock, 1000)
	require.Zero(t, results[providersGen.providersAddresses[skipIndex]])
}

func TestProviderOptimizerAvailabilityBlockError(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providersCount := 10
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)
	rand.InitRandomSeed()
	requestCU := uint64(10)
	requestBlock := int64(1000)

	syncBlock := uint64(requestBlock)
	chosenIndex := rand.Intn(providersCount - 2)

	for i := range providersGen.providersAddresses {
		time.Sleep(4 * time.Millisecond)
		if i == chosenIndex || i == chosenIndex+1 || i == chosenIndex+2 {
			// give better syncBlock, worse latency by a little
			providerOptimizer.AppendRelayData(providersGen.providersAddresses[i], TEST_BASE_WORLD_LATENCY+10*time.Millisecond, false, requestCU, syncBlock)
			continue
		}
		providerOptimizer.AppendRelayData(providersGen.providersAddresses[i], TEST_BASE_WORLD_LATENCY, false, requestCU, syncBlock-1) // update that he doesn't have the latest requested block
	}
	time.Sleep(4 * time.Millisecond)
	selectionTier, _, _ := providerOptimizer.CalculateSelectionTiers(providersGen.providersAddresses, nil, requestCU, requestBlock)
	tierChances := selectionTier.ShiftTierChance(OptimizerNumTiers, map[int]float64{0: ATierChance, OptimizerNumTiers - 1: LastTierChance})
	require.Greater(t, tierChances[0], 0.7, tierChances)
	results, tierResults := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, requestCU, requestBlock, 1000)
	require.Greater(t, tierResults[0], 500, tierResults) // we should pick the best tier most often
	// out of 10 providers, and with 3 in the top tier we should pick 0 around a third of that
	require.Greater(t, results[providersGen.providersAddresses[chosenIndex]], 200, results) // we should pick the best tier most often
	sumResults := results[providersGen.providersAddresses[chosenIndex]] + results[providersGen.providersAddresses[chosenIndex+1]] + results[providersGen.providersAddresses[chosenIndex+2]]
	require.Greater(t, sumResults, 500, results) // we should pick the best tier most often
	// now try to get a previous block, our chosenIndex should be inferior in latency and blockError chance should be the same

	results, tierResults = runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, requestCU, requestBlock-1, 1000)
	require.Greater(t, tierResults[0], 500, tierResults) // we should pick the best tier most often
	// out of 10 providers, and with 3 in the top tier we should pick 0 around a third of that
	require.Less(t, results[providersGen.providersAddresses[chosenIndex]], 50, results) // chosen indexes shoulnt be in the tier
	sumResults = results[providersGen.providersAddresses[chosenIndex]] + results[providersGen.providersAddresses[chosenIndex+1]] + results[providersGen.providersAddresses[chosenIndex+2]]
	require.Less(t, sumResults, 150, results) // we should pick the best tier most often
}

// TODO::PRT-1114 This needs to be fixed asap. currently commented out as it prevents pushing unrelated code
// Also on typescript sdk
// func TestProviderOptimizerUpdatingLatency(t *testing.T) {
// 	providerOptimizer := setupProviderOptimizer(1)
// 	providersCount := 2
// 	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)
// 	providerAddress := providersGen.providersAddresses[0]
// 	requestCU := uint64(10)
// 	requestBlock := int64(1000)
// 	syncBlock := uint64(requestBlock)
// 	providerOptimizer.providersStorage = &providerOptimizerSyncCache{value: map[interface{}]interface{}{}}
// 	// in this test we are repeatedly adding better results, and latency score should improve
// 	for i := 0; i < 10; i++ {
// 		providerData, _ := providerOptimizer.getProviderData(providerAddress)
// 		currentLatencyScore := providerOptimizer.calculateLatencyScore(providerData, requestCU, requestBlock)
// 		providerOptimizer.AppendProbeRelayData(providerAddress, TEST_BASE_WORLD_LATENCY, true)
// 		providerData, found := providerOptimizer.getProviderData(providerAddress)
// 		require.True(t, found)
// 		newLatencyScore := providerOptimizer.calculateLatencyScore(providerData, requestCU, requestBlock)
// 		require.Greater(t, currentLatencyScore, newLatencyScore, i)
// 	}
// 	providerAddress = providersGen.providersAddresses[1]
// 	for i := 0; i < 10; i++ {
// 		providerData, _ := providerOptimizer.getProviderData(providerAddress)
// 		currentLatencyScore := providerOptimizer.calculateLatencyScore(providerData, requestCU, requestBlock)
// 		providerOptimizer.AppendRelayData(providerAddress, TEST_BASE_WORLD_LATENCY, false, requestCU, syncBlock)
// 		providerData, found := providerOptimizer.getProviderData(providerAddress)
// 		require.True(t, found)
// 		newLatencyScore := providerOptimizer.calculateLatencyScore(providerData, requestCU, requestBlock)
// 		require.Greater(t, currentLatencyScore, newLatencyScore, i)
// 	}
// }

func TestProviderOptimizerExploration(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(2)
	providersCount := 10
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)
	requestCU := uint64(10)
	requestBlock := int64(1000)
	syncBlock := uint64(requestBlock)

	rand.InitRandomSeed()
	// start with a disabled chosen index
	chosenIndex := -1
	testProvidersExploration := func(iterations int) float64 {
		exploration := 0.0
		for i := 0; i < iterations; i++ {
			returnedProviders, _ := providerOptimizer.ChooseProvider(providersGen.providersAddresses, nil, requestCU, requestBlock)
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
	providerOptimizer.appendRelayData(providersGen.providersAddresses[chosenIndex], TEST_BASE_WORLD_LATENCY*2, false, true, requestCU, syncBlock, time.Now().Add(-35*time.Second))
	// set a basic state for all other provider, with a recent time (so they can't be selected for exploration)
	for i := 0; i < 10; i++ {
		for index, address := range providersGen.providersAddresses {
			if index == chosenIndex {
				// we set chosenIndex with a past time so it can be selected for exploration
				continue
			}
			// set samples in the future so they are never a candidate for exploration
			providerOptimizer.appendRelayData(address, TEST_BASE_WORLD_LATENCY*2, false, true, requestCU, syncBlock, time.Now().Add(1*time.Second))
		}
		time.Sleep(4 * time.Millisecond)
	}

	// with a cost strategy we expect exploration at a 10% rate
	providerOptimizer.strategy = STRATEGY_BALANCED        // that's the default but to be explicit
	providerOptimizer.wantedNumProvidersInConcurrency = 2 // that's in the constructor but to be explicit
	iterations := 10000
	exploration = testProvidersExploration(iterations)
	require.Less(t, exploration, float64(1.4)*float64(iterations)*DEFAULT_EXPLORATION_CHANCE)    // allow mistake buffer of 40% because of randomness
	require.Greater(t, exploration, float64(0.6)*float64(iterations)*DEFAULT_EXPLORATION_CHANCE) // allow mistake buffer of 40% because of randomness

	// with a cost strategy we expect exploration to happen once in 100 samples
	providerOptimizer.strategy = STRATEGY_COST
	exploration = testProvidersExploration(iterations)
	require.Less(t, exploration, float64(1.4)*float64(iterations)*COST_EXPLORATION_CHANCE)    // allow mistake buffer of 40% because of randomness
	require.Greater(t, exploration, float64(0.6)*float64(iterations)*COST_EXPLORATION_CHANCE) // allow mistake buffer of 40% because of randomness

	// privacy disables exploration
	providerOptimizer.strategy = STRATEGY_PRIVACY
	exploration = testProvidersExploration(iterations)
	require.Equal(t, exploration, float64(0))
}

func TestProviderOptimizerSyncScore(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providersGen := (&providersGenerator{}).setupProvidersForTest(10)
	rand.InitRandomSeed()
	requestCU := uint64(10)
	requestBlock := spectypes.LATEST_BLOCK

	syncBlock := uint64(1000)

	chosenIndex := rand.Intn(len(providersGen.providersAddresses))
	sampleTime := time.Now()
	for j := 0; j < 3; j++ { // repeat several times because a sync score is only correct after all providers sent their first block otherwise its giving favor to the first one
		for i := range providersGen.providersAddresses {
			time.Sleep(4 * time.Millisecond)
			if i == chosenIndex {
				// give better syncBlock, latency is a tiny bit worse for the second check
				providerOptimizer.appendRelayData(providersGen.providersAddresses[i], TEST_BASE_WORLD_LATENCY*2+1*time.Microsecond, false, true, requestCU, syncBlock+5, sampleTime)
				continue
			}
			providerOptimizer.appendRelayData(providersGen.providersAddresses[i], TEST_BASE_WORLD_LATENCY*2, false, true, requestCU, syncBlock, sampleTime) // update that he doesn't have the latest requested block
		}
		sampleTime = sampleTime.Add(time.Millisecond * 5)
	}
	time.Sleep(4 * time.Millisecond)
	selectionTier, _, _ := providerOptimizer.CalculateSelectionTiers(providersGen.providersAddresses, nil, requestCU, requestBlock)
	tier0 := selectionTier.GetTier(0, 4, 3)
	require.Greater(t, len(tier0), 0) // shouldn't be empty
	// we have the best score on the top tier and it's sorted
	require.Equal(t, providersGen.providersAddresses[chosenIndex], tier0[0].Address)

	// now choose with a specific block that all providers have
	selectionTier, _, _ = providerOptimizer.CalculateSelectionTiers(providersGen.providersAddresses, nil, requestCU, int64(syncBlock))
	tier0 = selectionTier.GetTier(0, 4, 3)
	for idx := range tier0 {
		// sync score doesn't matter now so the tier0 is recalculated and chosenIndex has worst latency
		require.NotEqual(t, providersGen.providersAddresses[chosenIndex], tier0[idx].Address)
	}
}

func TestProviderOptimizerStrategiesScoring(t *testing.T) {
	rand.InitRandomSeed()
	providerOptimizer := setupProviderOptimizer(1)
	providersCount := 10
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)
	requestCU := uint64(10)
	requestBlock := spectypes.LATEST_BLOCK
	syncBlock := uint64(1000)

	// set a basic state for all providers
	sampleTime := time.Now()
	for i := 0; i < 10; i++ {
		for _, address := range providersGen.providersAddresses {
			providerOptimizer.appendRelayData(address, TEST_BASE_WORLD_LATENCY*2, false, true, requestCU, syncBlock, sampleTime)
		}
		time.Sleep(4 * time.Millisecond)
	}
	// provider 2 doesn't get a probe availability hit, this is the most meaningful factor
	for idx, address := range providersGen.providersAddresses {
		if idx != 2 {
			providerOptimizer.AppendProbeRelayData(address, TEST_BASE_WORLD_LATENCY*2, false)
			time.Sleep(4 * time.Millisecond)
		}
		providerOptimizer.AppendProbeRelayData(address, TEST_BASE_WORLD_LATENCY*2, true)
		time.Sleep(4 * time.Millisecond)
		providerOptimizer.AppendProbeRelayData(address, TEST_BASE_WORLD_LATENCY*2, false)
		time.Sleep(4 * time.Millisecond)
		providerOptimizer.AppendProbeRelayData(address, TEST_BASE_WORLD_LATENCY*2, true)
		time.Sleep(4 * time.Millisecond)
		providerOptimizer.AppendProbeRelayData(address, TEST_BASE_WORLD_LATENCY*2, true)
		time.Sleep(4 * time.Millisecond)
	}

	sampleTime = time.Now()
	improvedLatency := 280 * time.Millisecond
	normalLatency := TEST_BASE_WORLD_LATENCY * 2
	improvedBlock := syncBlock + 1
	// provider 0 gets a good latency
	providerOptimizer.appendRelayData(providersGen.providersAddresses[0], improvedLatency, false, true, requestCU, syncBlock, sampleTime)

	// providers 3,4 get a regular entry
	providerOptimizer.appendRelayData(providersGen.providersAddresses[3], normalLatency, false, true, requestCU, syncBlock, sampleTime)
	providerOptimizer.appendRelayData(providersGen.providersAddresses[4], normalLatency, false, true, requestCU, syncBlock, sampleTime)

	// provider 1 gets a good sync
	providerOptimizer.appendRelayData(providersGen.providersAddresses[1], normalLatency, false, true, requestCU, improvedBlock, sampleTime)

	sampleTime = sampleTime.Add(10 * time.Millisecond)
	// now repeat to modify all providers scores across sync calculation
	providerOptimizer.appendRelayData(providersGen.providersAddresses[0], improvedLatency, false, true, requestCU, syncBlock, sampleTime)
	providerOptimizer.appendRelayData(providersGen.providersAddresses[3], normalLatency, false, true, requestCU, syncBlock, sampleTime)
	providerOptimizer.appendRelayData(providersGen.providersAddresses[4], normalLatency, false, true, requestCU, syncBlock, sampleTime)
	providerOptimizer.appendRelayData(providersGen.providersAddresses[1], normalLatency, false, true, requestCU, improvedBlock, sampleTime)

	time.Sleep(4 * time.Millisecond)
	providerOptimizer.strategy = STRATEGY_BALANCED
	// a balanced strategy should pick provider 2 because of it's high availability
	selectionTier, _, _ := providerOptimizer.CalculateSelectionTiers(providersGen.providersAddresses, nil, requestCU, requestBlock)
	tier0 := selectionTier.GetTier(0, 4, 3)
	require.Greater(t, len(tier0), 0) // shouldn't be empty
	// we have the best score on the top tier and it's sorted
	require.Equal(t, providersGen.providersAddresses[2], tier0[0].Address)

	providerOptimizer.strategy = STRATEGY_COST
	// with a cost strategy we expect the same as balanced
	selectionTier, _, _ = providerOptimizer.CalculateSelectionTiers(providersGen.providersAddresses, nil, requestCU, requestBlock)
	tier0 = selectionTier.GetTier(0, 4, 3)
	require.Greater(t, len(tier0), 0) // shouldn't be empty
	// we have the best score on the top tier and it's sorted
	require.Equal(t, providersGen.providersAddresses[2], tier0[0].Address)

	providerOptimizer.strategy = STRATEGY_LATENCY
	// latency strategy should pick the best latency
	selectionTier, _, _ = providerOptimizer.CalculateSelectionTiers(providersGen.providersAddresses, map[string]struct{}{providersGen.providersAddresses[2]: {}}, requestCU, requestBlock)
	tier0 = selectionTier.GetTier(0, 4, 3)
	require.Greater(t, len(tier0), 0) // shouldn't be empty
	require.Equal(t, providersGen.providersAddresses[0], tier0[0].Address)

	providerOptimizer.strategy = STRATEGY_SYNC_FRESHNESS
	// freshness strategy should pick the most advanced provider
	selectionTier, _, _ = providerOptimizer.CalculateSelectionTiers(providersGen.providersAddresses, map[string]struct{}{providersGen.providersAddresses[2]: {}}, requestCU, requestBlock)
	tier0 = selectionTier.GetTier(0, 4, 3)
	require.Greater(t, len(tier0), 0) // shouldn't be empty
	require.Equal(t, providersGen.providersAddresses[1], tier0[0].Address)

	// but if we request a past block, then it doesnt matter and we choose by latency:
	selectionTier, _, _ = providerOptimizer.CalculateSelectionTiers(providersGen.providersAddresses, map[string]struct{}{providersGen.providersAddresses[2]: {}}, requestCU, int64(syncBlock))
	tier0 = selectionTier.GetTier(0, 4, 3)
	require.Greater(t, len(tier0), 0) // shouldn't be empty
	require.Equal(t, providersGen.providersAddresses[0], tier0[0].Address)
}

func TestExcellence(t *testing.T) {
	floatVal := 0.25
	dec := turnFloatToDec(floatVal, 8)
	floatNew, err := dec.Float64()
	require.NoError(t, err)
	require.Equal(t, floatVal, floatNew)

	providerOptimizer := setupProviderOptimizer(1)
	providersCount := 5
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)
	requestCU := uint64(10)
	syncBlock := uint64(1000)
	// set a basic state for all of them
	sampleTime := time.Now()
	for i := 0; i < 10; i++ {
		for _, address := range providersGen.providersAddresses {
			providerOptimizer.appendRelayData(address, TEST_BASE_WORLD_LATENCY*2, false, true, requestCU, syncBlock, sampleTime)
		}
		time.Sleep(4 * time.Millisecond)
	}
	report, rawReport := providerOptimizer.GetExcellenceQoSReportForProvider(providersGen.providersAddresses[0])
	require.NotNil(t, report)
	require.NotNil(t, rawReport)
	report2, rawReport2 := providerOptimizer.GetExcellenceQoSReportForProvider(providersGen.providersAddresses[1])
	require.NotNil(t, report2)
	require.Equal(t, report, report2)
	require.NotNil(t, rawReport2)
	require.Equal(t, rawReport, rawReport2)
}

func TestPerturbationWithNormalGaussianOnConcurrentComputation(t *testing.T) {
	// Initialize random seed
	rand.InitRandomSeed()

	// Number of iterations
	iterations := 100000

	// Original value and percentage
	orig := 10.0
	percentage := 0.1

	// Create slices to hold perturbed values
	perturbationValues := make([]float64, iterations)

	// WaitGroup to wait for all Goroutines to finish
	var wg sync.WaitGroup

	// Generate perturbed values concurrently
	wg.Add(iterations)
	for i := 0; i < iterations; i++ {
		go func(index int) {
			defer wg.Done()
			perturbationValues[index] = pertrubWithNormalGaussian(orig, percentage)
		}(i)
	}

	// Wait for all Goroutines to finish, this used to panic before the fix and therefore we have this test
	wg.Wait()
	fmt.Println("Test completed successfully")
}

// test low providers count 0-9
func TestProviderOptimizerProvidersCount(t *testing.T) {
	rand.InitRandomSeed()
	providerOptimizer := setupProviderOptimizer(1)
	providersCount := 10
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)
	requestCU := uint64(10)
	requestBlock := spectypes.LATEST_BLOCK
	syncBlock := uint64(1000)
	sampleTime := time.Now()
	for i := 0; i < 10; i++ {
		for _, address := range providersGen.providersAddresses {
			providerOptimizer.appendRelayData(address, TEST_BASE_WORLD_LATENCY*2, false, true, requestCU, syncBlock, sampleTime)
		}
		time.Sleep(4 * time.Millisecond)
	}
	playbook := []struct {
		name      string
		providers int
	}{
		{
			name:      "one",
			providers: 1,
		},
		{
			name:      "two",
			providers: 2,
		},
		{
			name:      "three",
			providers: 3,
		},
		{
			name:      "four",
			providers: 4,
		},
		{
			name:      "five",
			providers: 5,
		},
		{
			name:      "six",
			providers: 6,
		},
		{
			name:      "seven",
			providers: 7,
		},
		{
			name:      "eight",
			providers: 8,
		},
		{
			name:      "nine",
			providers: 9,
		},
	}
	for _, play := range playbook {
		t.Run(play.name, func(t *testing.T) {
			for i := 0; i < 10; i++ {
				returnedProviders, _ := providerOptimizer.ChooseProvider(providersGen.providersAddresses[:play.providers], nil, requestCU, requestBlock)
				require.Greater(t, len(returnedProviders), 0)
			}
		})
	}
}

func TestProviderOptimizerWeights(t *testing.T) {
	rand.InitRandomSeed()
	providerOptimizer := setupProviderOptimizer(1)
	providersCount := 10
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)
	requestCU := uint64(10)
	requestBlock := spectypes.LATEST_BLOCK
	syncBlock := uint64(1000)
	sampleTime := time.Now()
	weights := map[string]int64{
		providersGen.providersAddresses[0]: 10000000000000, // simulating 10m tokens
	}
	for i := 1; i < 10; i++ {
		weights[providersGen.providersAddresses[i]] = 50000000000
	}

	normalLatency := TEST_BASE_WORLD_LATENCY * 2
	improvedLatency := normalLatency - 5*time.Millisecond
	improvedBlock := syncBlock + 2

	providerOptimizer.UpdateWeights(weights, syncBlock)
	for i := 0; i < 10; i++ {
		for idx, address := range providersGen.providersAddresses {
			if idx == 0 {
				providerOptimizer.appendRelayData(address, normalLatency, false, true, requestCU, improvedBlock, sampleTime)
			} else {
				providerOptimizer.appendRelayData(address, improvedLatency, false, true, requestCU, syncBlock, sampleTime)
			}
			sampleTime = sampleTime.Add(5 * time.Millisecond)
			time.Sleep(4 * time.Millisecond)
		}
	}

	// verify 0 has the best score
	selectionTier, _, _ := providerOptimizer.CalculateSelectionTiers(providersGen.providersAddresses, nil, requestCU, requestBlock)
	tier0 := selectionTier.GetTier(0, 4, 3)
	require.Greater(t, len(tier0), 0) // shouldn't be empty
	require.Equal(t, providersGen.providersAddresses[0], tier0[0].Address)

	// if we pick by sync, provider 0 is in the top tier and should be selected very often
	results, tierResults := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, requestCU, requestBlock, 1000)
	require.Greater(t, tierResults[0], 600, tierResults) // we should pick the best tier most often
	// out of 10 providers, and with 3 in the top tier we should pick 0 around a third of that
	require.Greater(t, results[providersGen.providersAddresses[0]], 550, results) // we should pick the top provider in tier 0 most times due to weight

	// if we pick by latency only, provider 0 is in the worst tier and can't be selected at all
	results, tierResults = runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, requestCU, int64(syncBlock), 1000)
	require.Greater(t, tierResults[0], 500, tierResults) // we should pick the best tier most often
	// out of 10 providers, and with 3 in the top tier we should pick 0 around a third of that
	require.Zero(t, results[providersGen.providersAddresses[0]])
}

func TestProviderOptimizerTiers(t *testing.T) {
	rand.InitRandomSeed()

	providersCountList := []int{9, 10}
	for why, providersCount := range providersCountList {
		providerOptimizer := setupProviderOptimizer(1)
		providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)
		requestCU := uint64(10)
		requestBlock := spectypes.LATEST_BLOCK
		syncBlock := uint64(1000)
		sampleTime := time.Now()
		normalLatency := TEST_BASE_WORLD_LATENCY * 2
		for i := 0; i < 10; i++ {
			for _, address := range providersGen.providersAddresses {
				modifierLatency := rand.Int63n(3) - 1
				modifierSync := rand.Int63n(3) - 1
				providerOptimizer.appendRelayData(address, normalLatency+time.Duration(modifierLatency)*time.Millisecond, false, true, requestCU, syncBlock+uint64(modifierSync), sampleTime)
				sampleTime = sampleTime.Add(5 * time.Millisecond)
				time.Sleep(4 * time.Millisecond)
			}
		}
		selectionTier, _, _ := providerOptimizer.CalculateSelectionTiers(providersGen.providersAddresses, nil, requestCU, requestBlock)
		shiftedChances := selectionTier.ShiftTierChance(4, map[int]float64{0: 0.75})
		require.NotZero(t, shiftedChances[3])
		// if we pick by sync, provider 0 is in the top tier and should be selected very often
		_, tierResults := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, requestCU, requestBlock, 1000)
		for index := 0; index < OptimizerNumTiers; index++ {
			if providersCount >= 2*MinimumEntries && index == OptimizerNumTiers-1 {
				// skip last tier if there's insufficient providers
				continue
			}
			require.NotZero(t, tierResults[index], "tierResults %v providersCount %s index %d why: %d", tierResults, providersCount, index, why)
		}
	}
}

func TestProviderOptimizerWithOptimizerQoSClient(t *testing.T) {
	rand.InitRandomSeed()

	wg := sync.WaitGroup{}
	wg.Add(1)
	httpServerHandler := func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, r.ContentLength)
		r.Body.Read(data)

		optimizerQoSReport := &[]map[string]interface{}{}
		err := json.Unmarshal(data, optimizerQoSReport)
		require.NoError(t, err)
		require.NotZero(t, len(*optimizerQoSReport))
		w.WriteHeader(http.StatusOK)
		wg.Done()
	}

	mockHttpServer := httptest.NewServer(http.HandlerFunc(httpServerHandler))
	defer mockHttpServer.Close()

	chainId := "dontcare"

	consumerOptimizerQoSClient := metrics.NewConsumerOptimizerQoSClient("lava@test", mockHttpServer.URL, 1*time.Second)
	consumerOptimizerQoSClient.StartOptimizersQoSReportsCollecting(context.Background(), 900*time.Millisecond)

	providerOptimizer := NewProviderOptimizer(STRATEGY_BALANCED, TEST_AVERAGE_BLOCK_TIME, TEST_BASE_WORLD_LATENCY, 10, consumerOptimizerQoSClient, chainId)
	consumerOptimizerQoSClient.RegisterOptimizer(providerOptimizer, chainId)

	syncBlock := uint64(1000)

	providerAddr := "lava@test"

	providerOptimizer.UpdateWeights(map[string]int64{
		providerAddr: 1000000000,
	}, syncBlock)

	requestCU := uint64(10)

	normalLatency := TEST_BASE_WORLD_LATENCY * 2
	providerOptimizer.appendRelayData(providerAddr, normalLatency, false, true, requestCU, syncBlock, time.Now())

	wg.Wait()
}

// TODO: new tests we need:
// check 3 providers, one with great stake one with great score
// retries: groups getting smaller
// no possible selections full
// do a simulation with better and worse providers, make sure it's good
// TODO: Oren - check optimizer selection with defaults (no scores for some of the providers)
