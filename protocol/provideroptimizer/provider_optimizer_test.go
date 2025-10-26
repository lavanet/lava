package provideroptimizer

import (
	"fmt"
	"slices"
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

const (
	TEST_AVERAGE_BLOCK_TIME = 10 * time.Second
	TEST_BASE_WORLD_LATENCY = 10 * time.Millisecond // same as score.DefaultLatencyNum
)

func setupProviderOptimizer(maxProvidersCount uint) *ProviderOptimizer {
	averageBlockTIme := TEST_AVERAGE_BLOCK_TIME
	return NewProviderOptimizer(StrategyBalanced, averageBlockTIme, maxProvidersCount, nil, "test", false)
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
//  1. Choose between 10 identical providers -> none should be in the worst tier
//  2. Append bad probe relay data for providers 5-7 and pick providers -> should not be 6-8
//  3. Append good probe relay data for providers 0-2 and pick providers -> should often be 0-2
func TestProviderOptimizerBasicProbeData(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providersGen := (&providersGenerator{}).setupProvidersForTest(10)
	rand.InitRandomSeed()
	cu := uint64(10)
	requestBlock := int64(1000)

	// damage providers 5-7 scores with bad latency probes relays
	// they should not be selected by the optimizer and should be in the worst tier
	badLatency := TEST_BASE_WORLD_LATENCY * 3
	providerOptimizer.AppendProbeRelayData(providersGen.providersAddresses[5], badLatency, true)
	providerOptimizer.AppendProbeRelayData(providersGen.providersAddresses[6], badLatency, true)
	providerOptimizer.AppendProbeRelayData(providersGen.providersAddresses[7], badLatency, true)
	time.Sleep(4 * time.Millisecond)
	// returnedProviders, tier = providerOptimizer.ChooseProvider(providersGen.providersAddresses, nil, cu, requestBlock)

	_, tierResults := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, 100, cu, requestBlock)
	require.Less(t, tierResults[3], 10, tierResults)

	// improve providers 0-2 scores with good latency probes relays
	// they should be selected by the optimizer more often and should be in the best tier
	goodLatency := TEST_BASE_WORLD_LATENCY / 2
	providerOptimizer.AppendProbeRelayData(providersGen.providersAddresses[0], goodLatency, true)
	providerOptimizer.AppendProbeRelayData(providersGen.providersAddresses[1], goodLatency, true)
	providerOptimizer.AppendProbeRelayData(providersGen.providersAddresses[2], goodLatency, true)
	time.Sleep(4 * time.Millisecond)
	results, tierResults := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, 1000, cu, requestBlock)
	require.Greater(t, tierResults[0], 499, tierResults) // we should pick the best tier most often

	// out of 10 providers, and with 3 providers in the top tier we should pick
	// tier-0 providers around a third of that
	require.Greater(t, results[providersGen.providersAddresses[0]], 100, results) // we should pick the best tier most often
}

// runChooseManyTimesAndReturnResults uses the given optimizer and providers addresses
// to pick providers <times> times and return two results maps:
//  1. results: map of provider address to the number of times it was picked
//  2. tierResults: map of tier and the number of times a provider from the specific tier was picked
func runChooseManyTimesAndReturnResults(t *testing.T, providerOptimizer *ProviderOptimizer, providers []string, ignoredProviders map[string]struct{}, times int, cu uint64, requestBlock int64) (map[string]int, map[int]int) {
	tierResults := make(map[int]int)
	results := make(map[string]int)
	for i := 0; i < times; i++ {
		returnedProviders, tier := providerOptimizer.ChooseProvider(providers, ignoredProviders, cu, requestBlock)
		require.Equal(t, 1, len(returnedProviders))
		results[returnedProviders[0]]++
		tierResults[tier]++
	}
	return results, tierResults
}

// TestProviderOptimizerBasicRelayData tests the basic provider optimizer operation
// when it is updated with regular relays. Providers with bad scores should have a worse chance
// to be picked (and vice versa).
// Scenario:
//  0. There are 10 providers, the optimizer is configured to pick a single provider
//  1. Choose between 10 identical providers -> none should be in the worst tier
//  2. Append bad relay data for providers 5-7 and pick providers -> should not be 6-8
//  3. Append good relay data for providers 0-2 and pick providers -> should often be 0-2
func TestProviderOptimizerBasicRelayData(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providersGen := (&providersGenerator{}).setupProvidersForTest(10)
	rand.InitRandomSeed()
	cu := uint64(1)
	requestBlock := int64(1000)
	syncBlock := uint64(requestBlock)
	// AutoAdjustTiers = true
	// damage providers 5-7 scores with bad latency relays
	// they should not be selected by the optimizer and should be in the worst tier
	badLatency := TEST_BASE_WORLD_LATENCY * 3
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[5], badLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[6], badLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[7], badLatency, cu, syncBlock)

	// improve providers 0-2 scores with good latency probes relays
	// they should be selected by the optimizer more often and should be in the best tier
	goodLatency := TEST_BASE_WORLD_LATENCY / 3
	averageLatency := TEST_BASE_WORLD_LATENCY / 2
	// add good latency relays for providers 0-2
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[0], goodLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[1], goodLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[2], goodLatency, cu, syncBlock)
	// add average latency relays for providers 3,4,8,9
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[3], averageLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[4], averageLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[8], averageLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[9], averageLatency, cu, syncBlock)
	// add bad latency relays for providers 5-7
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[5], badLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[6], badLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[7], badLatency, cu, syncBlock)
	time.Sleep(4 * time.Millisecond)

	// there's a chance that some of the worst providers will be in part of a higher tier
	// because of a high minimum entries value, so filter the providers that are only in the worst tier
	selectionTier, _, _ := providerOptimizer.CalculateSelectionTiers(providersGen.providersAddresses, nil, cu, requestBlock)
	tier3Entries := selectionTier.GetTier(3, providerOptimizer.OptimizerNumTiers, 1)
	worstTierEntries := map[string]struct{}{
		providersGen.providersAddresses[5]: {},
		providersGen.providersAddresses[6]: {},
		providersGen.providersAddresses[7]: {},
	}
	for _, entry := range tier3Entries {
		// verify that the worst providers are the ones with the bad latency
		if entry.Address != providersGen.providersAddresses[5] &&
			entry.Address != providersGen.providersAddresses[6] &&
			entry.Address != providersGen.providersAddresses[7] {
			t.Fatalf("entry %s is not in the worst tier", entry.Address)
		}
	}

	results, tierResults := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, 1000, cu, requestBlock)
	require.Greater(t, tierResults[0], 400, tierResults) // we should pick the best tier most often

	// Out of 10 providers, and with 3 in the top tier we should pick 0 around a third of that
	require.Greater(t, results[providersGen.providersAddresses[0]], 200, results)

	// Ensure the bad providers (5-7) are in results map even if they were never picked
	for i := 5; i <= 7; i++ {
		if _, exists := results[providersGen.providersAddresses[i]]; !exists {
			results[providersGen.providersAddresses[i]] = 0
		}
	}
	// the bad providers shouldn't have been picked even once
	// Find the two least picked providers
	var leastPicked, secondLeastPicked string
	leastCount, secondLeastCount := 0xffffffff, 0xffffffff
	for addr, count := range results {
		if count < leastCount {
			secondLeastCount = leastCount
			secondLeastPicked = leastPicked
			leastCount = count
			leastPicked = addr
		} else if count < secondLeastCount {
			secondLeastCount = count
			secondLeastPicked = addr
		}
	}
	minimumScores := map[string]int{
		leastPicked:       leastCount,
		secondLeastPicked: secondLeastCount,
	}

	utils.LavaFormatInfo("results", utils.LogAttr("results", results), utils.LogAttr("minimumScores", minimumScores), utils.LogAttr("worstTierEntries", worstTierEntries))
	for address := range minimumScores {
		require.Contains(t, worstTierEntries, address)
	}
}

func TestProviderOptimizerBasicRelayDataAutoAdjustTiers(t *testing.T) {
	// Save original value and defer restore
	originalAutoAdjustTiers := AutoAdjustTiers
	defer func() {
		AutoAdjustTiers = originalAutoAdjustTiers
	}()

	// Set test value
	AutoAdjustTiers = true

	providerOptimizer := setupProviderOptimizer(1)
	providersGen := (&providersGenerator{}).setupProvidersForTest(10)
	weights := make(map[string]int64)
	rand.InitRandomSeed()
	cu := uint64(1)
	requestBlock := int64(1000)
	syncBlock := uint64(1000)
	for _, address := range providersGen.providersAddresses {
		weights[address] = 1000
	}
	providerOptimizer.UpdateWeights(weights, uint64(requestBlock))
	// damage providers 5-7 scores with bad latency relays
	// they should not be selected by the optimizer and should be in the worst tier
	badLatency := TEST_BASE_WORLD_LATENCY * 3
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[5], badLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[6], badLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[7], badLatency, cu, syncBlock)

	// improve providers 0-2 scores with good latency probes relays
	// they should be selected by the optimizer more often and should be in the best tier
	goodLatency := TEST_BASE_WORLD_LATENCY / 3
	bestLatency := TEST_BASE_WORLD_LATENCY / 4
	averageLatency := TEST_BASE_WORLD_LATENCY / 2
	// add good latency relays for providers 0-2
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[0], bestLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[1], goodLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[2], goodLatency, cu, syncBlock)
	// add average latency relays for providers 3,4,8,9
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[3], averageLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[4], averageLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[8], averageLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[9], averageLatency, cu, syncBlock)
	// add bad latency relays for providers 5-7
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[5], badLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[6], badLatency, cu, syncBlock)
	providerOptimizer.AppendRelayData(providersGen.providersAddresses[7], badLatency, cu, syncBlock)
	time.Sleep(4 * time.Millisecond)
	results, tierResults := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, 1000, cu, requestBlock)

	fmt.Println("providersGen.providersAddresses", providersGen.providersAddresses)
	fmt.Println("tierResults", tierResults)
	fmt.Println("results", results)
	selectionTier, _, _ := providerOptimizer.CalculateSelectionTiers(providersGen.providersAddresses, nil, cu, requestBlock)
	selectionTierScoresCount := selectionTier.ScoresCount()
	localMinimumEntries := providerOptimizer.GetMinTierEntries(selectionTier, selectionTierScoresCount)
	tier0Entries := selectionTier.GetTier(0, providerOptimizer.OptimizerNumTiers, localMinimumEntries)
	tier1Entries := selectionTier.GetTier(1, providerOptimizer.OptimizerNumTiers, localMinimumEntries)
	tier2Entries := selectionTier.GetTier(2, providerOptimizer.OptimizerNumTiers, localMinimumEntries)
	tier3Entries := selectionTier.GetTier(3, providerOptimizer.OptimizerNumTiers, localMinimumEntries)

	fmt.Println("tier0Entries", tier0Entries)
	fmt.Println("tier1Entries", tier1Entries)
	fmt.Println("tier2Entries", tier2Entries)
	fmt.Println("tier3Entries", tier3Entries)

	// Add tier validations
	expectedTier0 := []string{
		providersGen.providersAddresses[2],
		providersGen.providersAddresses[1],
		providersGen.providersAddresses[0],
	}
	expectedTier1 := []string{
		providersGen.providersAddresses[1],
		providersGen.providersAddresses[9],
		providersGen.providersAddresses[8],
	}
	expectedTier2 := []string{
		providersGen.providersAddresses[4],
		providersGen.providersAddresses[3],
	}
	expectedTier3 := []string{
		providersGen.providersAddresses[5],
		providersGen.providersAddresses[6],
		providersGen.providersAddresses[7],
	}

	// Validate tier 0 entries
	for i, entry := range tier0Entries {
		require.Contains(t, expectedTier0, entry.Address, "tier 0 entry %d should be %s, got %s", i, expectedTier0[i], entry.Address)
	}

	// Validate tier 1 entries
	for i, entry := range tier1Entries {
		require.Contains(t, expectedTier1, entry.Address, "tier 1 entry %d should be %s, got %s", i, expectedTier1[i], entry.Address)
	}

	// Validate tier 2 entries
	contains := 0
	for _, entry := range tier2Entries {
		// require.Contains(t, expectedTier2, entry.Address, "tier 2 entry %d should be %s, got %s", i, expectedTier2[i], entry.Address)
		if slices.Contains(expectedTier2, entry.Address) {
			contains++
		}
	}
	require.Equal(t, len(expectedTier2), contains, "tier 2 should have correct number of entries")

	// Validate tier 3 entries
	for i, entry := range tier3Entries {
		require.Contains(t, expectedTier3, entry.Address, "tier 3 entry %d should be %s, got %s", i, expectedTier3[i], entry.Address)
	}
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
		providerOptimizer.AppendProbeRelayData(providersGen.providersAddresses[i], TEST_BASE_WORLD_LATENCY, false)
	}

	// pick providers, the three random ones should be top-tier and picked more often
	time.Sleep(4 * time.Millisecond)
	results, tierResults := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, 1000, cu, requestBlock)
	require.Greater(t, tierResults[0], 300, tierResults) // 0.42 chance for top tier due to the algorithm to rebalance chances
	require.Greater(t, results[providersGen.providersAddresses[skipIndex]]+results[providersGen.providersAddresses[skipIndex+1]]+results[providersGen.providersAddresses[skipIndex+2]], 275)
	require.InDelta(t, results[providersGen.providersAddresses[skipIndex]], results[providersGen.providersAddresses[skipIndex+1]], 50)

	// pick providers again but this time ignore one of the random providers, it shouldn't be picked
	results, _ = runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, map[string]struct{}{providersGen.providersAddresses[skipIndex]: {}}, 1000, cu, requestBlock)
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
		providerOptimizer.AppendRelayFailure(providersGen.providersAddresses[i])
	}

	// pick providers, the three random ones should be top-tier and picked more often
	time.Sleep(4 * time.Millisecond)
	results, tierResults := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, 1000, cu, requestBlock)
	require.Greater(t, tierResults[0], 300, tierResults) // 0.42 chance for top tier due to the algorithm to rebalance chances
	require.Greater(t, results[providersGen.providersAddresses[skipIndex]]+results[providersGen.providersAddresses[skipIndex+1]]+results[providersGen.providersAddresses[skipIndex+2]], 270)
	require.InDelta(t, results[providersGen.providersAddresses[skipIndex]], results[providersGen.providersAddresses[skipIndex+1]], 50)

	// pick providers again but this time ignore one of the random providers, it shouldn't be picked
	results, _ = runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, map[string]struct{}{providersGen.providersAddresses[skipIndex]: {}}, 1000, cu, requestBlock)
	require.Zero(t, results[providersGen.providersAddresses[skipIndex]])
}

func TestProviderOptimizerAvailabilityBlockError(t *testing.T) {
	providerOptimizer := setupProviderOptimizer(1)
	providersCount := 10
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)
	rand.InitRandomSeed()
	cu := uint64(10)
	requestBlock := int64(1000)
	syncBlock := uint64(1000)
	badSyncBlock := syncBlock - 1

	// damage all the providers scores with bad sync relays but three random ones
	// the three random providers also get slightly worse latency
	// bad sync means an update that doesn't have the latest requested block
	chosenIndex := rand.Intn(providersCount - 2)
	threeOthers := []int{}
	for i := range providersGen.providersAddresses {
		time.Sleep(4 * time.Millisecond)
		if i == chosenIndex || i == chosenIndex+1 || i == chosenIndex+2 {
			slightlyBadLatency := TEST_BASE_WORLD_LATENCY + 100*time.Millisecond
			providerOptimizer.AppendRelayData(providersGen.providersAddresses[i], slightlyBadLatency, cu, syncBlock)
			continue
		}
		if len(threeOthers) < 3 {
			threeOthers = append(threeOthers, i)
		}
		providerOptimizer.AppendRelayData(providersGen.providersAddresses[i], TEST_BASE_WORLD_LATENCY, cu, badSyncBlock)
	}

	// make the top tier chance to be 70%
	time.Sleep(4 * time.Millisecond)
	selectionTier, _, _ := providerOptimizer.CalculateSelectionTiers(providersGen.providersAddresses, nil, cu, requestBlock)
	tierChances := selectionTier.ShiftTierChance(OptimizerNumTiers, map[int]float64{0: ATierChance, OptimizerNumTiers - 1: LastTierChance})
	require.Greater(t, tierChances[0], 0.7, tierChances)

	// pick providers, the top-tier should be picked picked more often (at least half the times)
	results, tierResults := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, 1000, cu, requestBlock)
	require.Greater(t, tierResults[0], 333, tierResults)

	// out of 10 providers, and with 3 in the top tier we should pick 0 around a third of that
	require.Greater(t, results[providersGen.providersAddresses[chosenIndex]], 200, results)
	sumResults := results[providersGen.providersAddresses[chosenIndex]] + results[providersGen.providersAddresses[chosenIndex+1]] + results[providersGen.providersAddresses[chosenIndex+2]]
	require.Greater(t, sumResults, 333, results) // we should pick the best tier most often

	// now try to get a previous block, our chosenIndex should be inferior in latency and blockError chance should be the same
	results, tierResults = runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, 1000, cu, requestBlock-1)
	require.Greater(t, tierResults[0], 333, tierResults) // we should pick the best tier most often
	// out of 10 providers, and with 3 in the top tier we should pick 0 around a third of that
	require.Less(t, results[providersGen.providersAddresses[chosenIndex]], 500, results) // chosen indexes shoulnt be in the tier

	// we would expect the worst tiers to be less than the other tiers.
	sumResults = results[providersGen.providersAddresses[chosenIndex]] + results[providersGen.providersAddresses[chosenIndex+1]] + results[providersGen.providersAddresses[chosenIndex+2]]
	sumResultsOthers := results[providersGen.providersAddresses[threeOthers[0]]] + results[providersGen.providersAddresses[threeOthers[1]]] + results[providersGen.providersAddresses[threeOthers[2]]]
	require.Less(t, sumResults, sumResultsOthers, results) // we should pick the best tier most often
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

	rand.InitRandomSeed()
	// start with a disabled chosen index
	chosenIndex := -1
	testProvidersExploration := func(iterations int) float64 {
		exploration := 0.0
		for i := 0; i < iterations; i++ {
			returnedProviders, _ := providerOptimizer.ChooseProvider(providersGen.providersAddresses, nil, cu, requestBlock)
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
				providerOptimizer.appendRelayData(providersGen.providersAddresses[i], TEST_BASE_WORLD_LATENCY*2+1*time.Microsecond, true, cu, syncBlock+5, sampleTime)
				continue
			}
			providerOptimizer.appendRelayData(providersGen.providersAddresses[i], TEST_BASE_WORLD_LATENCY*2, true, cu, syncBlock, sampleTime) // update that he doesn't have the latest requested block
		}
		sampleTime = sampleTime.Add(time.Millisecond * 5)
	}
	time.Sleep(4 * time.Millisecond)
	selectionTier, _, _ := providerOptimizer.CalculateSelectionTiers(providersGen.providersAddresses, nil, cu, requestBlock)
	tier0 := selectionTier.GetTier(0, 4, 3)
	require.Greater(t, len(tier0), 0) // shouldn't be empty
	// we have the best score on the top tier and it's sorted
	require.Equal(t, providersGen.providersAddresses[chosenIndex], tier0[0].Address)

	// now choose with a specific block that all providers have
	selectionTier, _, _ = providerOptimizer.CalculateSelectionTiers(providersGen.providersAddresses, nil, cu, int64(syncBlock))
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
	cu := uint64(10)
	requestBlock := spectypes.LATEST_BLOCK
	syncBlock := uint64(1000)

	// set a basic state for all providers
	sampleTime := time.Now()
	for i := 0; i < 10; i++ {
		for _, address := range providersGen.providersAddresses {
			providerOptimizer.appendRelayData(address, TEST_BASE_WORLD_LATENCY*2, true, cu, syncBlock, sampleTime)
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
	improvedLatency := TEST_BASE_WORLD_LATENCY / 2
	normalLatency := TEST_BASE_WORLD_LATENCY * 2
	improvedBlock := syncBlock + 1
	// provider 0 gets a good latency
	providerOptimizer.appendRelayData(providersGen.providersAddresses[0], improvedLatency, true, cu, syncBlock, sampleTime)

	// providers 3,4 get a regular entry
	providerOptimizer.appendRelayData(providersGen.providersAddresses[3], normalLatency, true, cu, syncBlock, sampleTime)
	providerOptimizer.appendRelayData(providersGen.providersAddresses[4], normalLatency, true, cu, syncBlock, sampleTime)

	// provider 1 gets a good sync
	providerOptimizer.appendRelayData(providersGen.providersAddresses[1], normalLatency, true, cu, improvedBlock, sampleTime)

	sampleTime = sampleTime.Add(10 * time.Millisecond)
	// now repeat to modify all providers scores across sync calculation
	providerOptimizer.appendRelayData(providersGen.providersAddresses[0], improvedLatency, true, cu, syncBlock, sampleTime)
	providerOptimizer.appendRelayData(providersGen.providersAddresses[3], normalLatency, true, cu, syncBlock, sampleTime)
	providerOptimizer.appendRelayData(providersGen.providersAddresses[4], normalLatency, true, cu, syncBlock, sampleTime)
	providerOptimizer.appendRelayData(providersGen.providersAddresses[1], normalLatency, true, cu, improvedBlock, sampleTime)

	time.Sleep(4 * time.Millisecond)
	providerOptimizer.strategy = StrategyBalanced
	// a balanced strategy should pick provider 2 because of it's high availability
	selectionTier, _, _ := providerOptimizer.CalculateSelectionTiers(providersGen.providersAddresses, nil, cu, requestBlock)
	tier0 := selectionTier.GetTier(0, 4, 3)
	require.Greater(t, len(tier0), 0) // shouldn't be empty
	// we have the best score on the top tier and it's sorted
	require.Equal(t, providersGen.providersAddresses[2], tier0[0].Address)

	providerOptimizer.strategy = StrategyCost
	// with a cost strategy we expect the same as balanced
	selectionTier, _, _ = providerOptimizer.CalculateSelectionTiers(providersGen.providersAddresses, nil, cu, requestBlock)
	tier0 = selectionTier.GetTier(0, 4, 3)
	require.Greater(t, len(tier0), 0) // shouldn't be empty
	// we have the best score on the top tier and it's sorted
	require.Equal(t, providersGen.providersAddresses[2], tier0[0].Address)

	providerOptimizer.strategy = StrategyLatency
	// latency strategy should pick the best latency
	selectionTier, _, _ = providerOptimizer.CalculateSelectionTiers(providersGen.providersAddresses, map[string]struct{}{providersGen.providersAddresses[2]: {}}, cu, requestBlock)
	tier0 = selectionTier.GetTier(0, 4, 3)
	require.Greater(t, len(tier0), 0) // shouldn't be empty
	require.Equal(t, providersGen.providersAddresses[0], tier0[0].Address)

	providerOptimizer.strategy = StrategySyncFreshness
	// freshness strategy should pick the most advanced provider
	selectionTier, _, _ = providerOptimizer.CalculateSelectionTiers(providersGen.providersAddresses, map[string]struct{}{providersGen.providersAddresses[2]: {}}, cu, requestBlock)
	tier0 = selectionTier.GetTier(0, 4, 3)
	require.Greater(t, len(tier0), 0) // shouldn't be empty
	require.Equal(t, providersGen.providersAddresses[1], tier0[0].Address)

	// but if we request a past block, then it doesnt matter and we choose by latency:
	selectionTier, _, _ = providerOptimizer.CalculateSelectionTiers(providersGen.providersAddresses, map[string]struct{}{providersGen.providersAddresses[2]: {}}, cu, int64(syncBlock))
	tier0 = selectionTier.GetTier(0, 4, 3)
	require.Greater(t, len(tier0), 0) // shouldn't be empty
	require.Equal(t, providersGen.providersAddresses[0], tier0[0].Address)
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
	rand.InitRandomSeed()
	providerOptimizer := setupProviderOptimizer(1)
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
				returnedProviders, _ := providerOptimizer.ChooseProvider(providersGen.providersAddresses[:play.providers], nil, cu, requestBlock)
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

	// verify 0 has the best score
	selectionTier, _, _ := providerOptimizer.CalculateSelectionTiers(providersGen.providersAddresses, nil, cu, requestBlock)
	tier0 := selectionTier.GetTier(0, 4, 3)
	require.Greater(t, len(tier0), 0) // shouldn't be empty
	require.Equal(t, providersGen.providersAddresses[0], tier0[0].Address)

	// if we pick by sync, provider 0 is in the top tier and should be selected very often
	results, tierResults := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, 1000, cu, requestBlock)
	maxTier := tierResults[0]
	for i := 1; i < len(tierResults); i++ {
		require.GreaterOrEqual(t, maxTier, tierResults[i], "tier 0 should have highest selection count")
	}
	// Check that provider 0 has the highest selection count
	maxCount := results[providersGen.providersAddresses[0]]
	for addr, count := range results {
		if addr != providersGen.providersAddresses[0] {
			require.GreaterOrEqual(t, maxCount, count, "provider 0 should have highest selection count")
		}
	}

	// if we pick by sync only, provider 0 is in the worst tier and should be selected less
	_, tierResults = runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, 1000, cu, int64(syncBlock))
	maxTier = tierResults[0]
	for i := 1; i < len(tierResults); i++ {
		require.GreaterOrEqual(t, maxTier, tierResults[i], "tier 0 should have highest selection count")
	}
}

func TestProviderOptimizerTiers(t *testing.T) {
	rand.InitRandomSeed()
	cu := uint64(10)
	requestBlock := int64(1000)
	providersCountList := []int{9, 10}
	for why, providersCount := range providersCountList {
		providerOptimizer := setupProviderOptimizer(1)
		providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)
		syncBlock := uint64(1000)
		sampleTime := time.Now()
		normalLatency := TEST_BASE_WORLD_LATENCY * 2
		for i := 0; i < 10; i++ {
			for _, address := range providersGen.providersAddresses {
				modifierLatency := rand.Int63n(3) - 1
				modifierSync := rand.Int63n(3) - 1
				providerOptimizer.appendRelayData(address, normalLatency+time.Duration(modifierLatency)*time.Millisecond, true, cu, syncBlock+uint64(modifierSync), sampleTime)
				sampleTime = sampleTime.Add(5 * time.Millisecond)
				time.Sleep(4 * time.Millisecond)
			}
		}
		selectionTier, _, _ := providerOptimizer.CalculateSelectionTiers(providersGen.providersAddresses, nil, cu, requestBlock)
		shiftedChances := selectionTier.ShiftTierChance(4, map[int]float64{0: 0.75})
		require.NotZero(t, shiftedChances[3])
		// if we pick by sync, provider 0 is in the top tier and should be selected very often
		_, tierResults := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, 1000, cu, requestBlock)
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
func TestProviderOptimizerChooseProvider(t *testing.T) {
	rand.InitRandomSeed()
	providerOptimizer := setupProviderOptimizer(1)
	providersCount := 6
	providerOptimizer.OptimizerNumTiers = providersCount / 2
	providerOptimizer.OptimizerMinTierEntries = 2 // make each tier contain 2 providers
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

	// setup scores to all providers
	improvedLatency := TEST_BASE_WORLD_LATENCY / 2
	normalLatency := TEST_BASE_WORLD_LATENCY * 2
	improvedBlock := syncBlock + 1

	// provider 0 and 1 gets a good latency and good sync
	providerOptimizer.appendRelayData(providersGen.providersAddresses[0], improvedLatency, true, cu, improvedBlock, sampleTime)
	providerOptimizer.appendRelayData(providersGen.providersAddresses[1], improvedLatency, true, cu, improvedBlock, sampleTime)

	// providers 2 and 3 get a good latency only
	providerOptimizer.appendRelayData(providersGen.providersAddresses[2], improvedLatency, true, cu, syncBlock, sampleTime)
	providerOptimizer.appendRelayData(providersGen.providersAddresses[3], improvedLatency, true, cu, syncBlock, sampleTime)

	// provider 4 and 5 gets a normal latency and sync
	providerOptimizer.appendRelayData(providersGen.providersAddresses[4], normalLatency, true, cu, syncBlock, sampleTime)
	providerOptimizer.appendRelayData(providersGen.providersAddresses[5], normalLatency, true, cu, syncBlock, sampleTime)

	// now repeat to modify all providers scores across sync calculation
	sampleTime = sampleTime.Add(10 * time.Millisecond)
	time.Sleep(10 * time.Millisecond)
	providerOptimizer.appendRelayData(providersGen.providersAddresses[5], normalLatency, true, cu, syncBlock, sampleTime)
	providerOptimizer.appendRelayData(providersGen.providersAddresses[4], normalLatency, true, cu, syncBlock, sampleTime)
	providerOptimizer.appendRelayData(providersGen.providersAddresses[3], improvedLatency, true, cu, syncBlock, sampleTime)
	providerOptimizer.appendRelayData(providersGen.providersAddresses[2], improvedLatency, true, cu, syncBlock, sampleTime)
	providerOptimizer.appendRelayData(providersGen.providersAddresses[1], improvedLatency, true, cu, improvedBlock, sampleTime)
	providerOptimizer.appendRelayData(providersGen.providersAddresses[0], improvedLatency, true, cu, improvedBlock, sampleTime)
	time.Sleep(4 * time.Millisecond)

	// choose many times and check results
	iterations := 10000
	results, tierResults := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, iterations, cu, requestBlock)
	require.InDelta(t, float64(iterations)*0.7, tierResults[0], float64(iterations)*0.15) // high score are picked 60%-80% of the times
	require.InDelta(t, results[providersGen.providersAddresses[0]],
		results[providersGen.providersAddresses[1]], float64(results[providersGen.providersAddresses[0]])*0.15) // no difference between high score providers (max 10% diff)
	require.Greater(t, results[providersGen.providersAddresses[2]], 0)                                           // high stake mid score provider picked at least once
	require.Greater(t, results[providersGen.providersAddresses[2]], results[providersGen.providersAddresses[3]]) // high stake mid score provider picked more than normal stake mid score provider
	require.Less(t, results[providersGen.providersAddresses[4]], results[providersGen.providersAddresses[2]])
	require.Less(t, results[providersGen.providersAddresses[5]], results[providersGen.providersAddresses[2]])
}

// TestProviderOptimizerRetriesWithReducedProvidersSet checks that when having a set of providers, the amount of
// providers doesn't matter and the choice is deterministic. The test does the following:
// 0. Assume a set of providers (great/mid/low score with high/low stake, all combinations)
// 1. Run ChooseProvider() <providers_amount> number of times. Each iteration, the chosen provider from the
// last iteration is removed from the providers set. We check the ranking of providers stays the same.
// 2. Do step 1 many times.
// Expected: the ranking of providers stays the same, providers with high stake are picked more often,
// providers from the lowest tier are not picked
func TestProviderOptimizerRetriesWithReducedProvidersSet(t *testing.T) {
	rand.InitRandomSeed()
	providerOptimizer := setupProviderOptimizer(1)
	providersCount := 6
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)

	// create 3 tiers with 2 providers each
	providerOptimizer.OptimizerNumTiers = providersCount / 2
	providerOptimizer.OptimizerMinTierEntries = 2

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
	baseLatency := TEST_BASE_WORLD_LATENCY.Seconds()

	// append relay data for each provider depending on its index in the providers array
	// the latency gets worse for increasing index so we assume the best provider is the 1st
	// address, after it the 2nd and so on
	for i := 0; i < 50; i++ {
		for j, address := range providersGen.providersAddresses {
			latency := time.Duration(baseLatency * float64(2*j+1) * float64(time.Millisecond))
			providerOptimizer.appendRelayData(address, latency, true, cu, syncBlock, sampleTime)
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
		res, tierResults := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses[i:], nil, iterations, cu, requestBlock)

		switch i {
		case 0:
			// 6 providers, 3 tiers, last one not picked so only
			// providers 0,1,2,3 are picked. tier 0: providers 0,1
			// tier 1: providers 2,3
			// provider 1,3 have higher stake and should be picked more often within their tier
			require.Greater(t, tierResults[0], 480)
			require.Greater(t, tierResults[0], tierResults[1])
			require.Greater(t, res[providersGen.providersAddresses[1]], res[providersGen.providersAddresses[0]])
			require.Greater(t, res[providersGen.providersAddresses[3]], res[providersGen.providersAddresses[2]])
		case 1:
			// 5 providers, 3 tiers, last one not picked so only
			// providers 1,2,3 are picked. tier 0: providers 1,2
			// tier 1: providers 2,3,4 (2 and 4 with part)
			// provider 1 has higher stake and should be picked more often within their tier
			// provider 3 has higher stake than provider 4 and 4 is in tier 1 and 2 (worst tier) so
			// provider 3 should be picked more often than provider 4
			require.Greater(t, tierResults[0], 480)
			require.Greater(t, tierResults[0], tierResults[1])
			require.Greater(t, res[providersGen.providersAddresses[1]], res[providersGen.providersAddresses[2]])
			require.Greater(t, res[providersGen.providersAddresses[3]], res[providersGen.providersAddresses[4]])
		case 2:
			// 4 providers, 3 tiers, last one not picked so only
			// providers 2,3,4 are picked. tier 0: providers 2,3
			// tier 1: providers 3,4
			// provider 3 has higher stake and should be picked more often within their tier
			// provider 3 has higher stake than provider 4 and 4 is in tier 1 and 2 (worst tier) so
			// provider 3 should be picked more often than provider 4
			require.Greater(t, tierResults[0], 480)
			require.Greater(t, tierResults[0], tierResults[1])
			require.Greater(t, res[providersGen.providersAddresses[3]], res[providersGen.providersAddresses[2]])
			require.Greater(t, res[providersGen.providersAddresses[3]], res[providersGen.providersAddresses[4]])
		case 3:
			// 3 providers, 3 tiers, last one not picked
			// minimum entries per tier is 2 and there are 1 provider per tier
			// because of this, each tier > 0 will have 2 providers and not 1
			// providers 3,4,5 are picked. tier 0: providers 3
			// tier 1: providers 4,5
			// provider 5 has higher stake and should be picked more often within their tier
			require.Greater(t, tierResults[0], 480)
			require.Greater(t, tierResults[0], tierResults[1])
			require.Greater(t, res[providersGen.providersAddresses[5]], res[providersGen.providersAddresses[4]])
		case 4:
			// 2 providers, 2 tiers
			// there are less providers than tiers, so num tiers is reduced to 2
			// providers 4,5 are picked. tier 0: providers 4
			// tier 1: providers 4,5 (4 with part=0.5, because it's dragged from tier 0)
			// provider 4 is picked more often than provider 5 even though it has less stake
			// because it's the only provider in tier 0
			require.Greater(t, tierResults[0], 480)
			require.Greater(t, tierResults[0], tierResults[1])
			require.Greater(t, res[providersGen.providersAddresses[4]], res[providersGen.providersAddresses[5]])
		}
	}
}

// TestProviderOptimizerChoiceSimulationBasedOnLatency checks that the overall choice mechanism acts as expected,
// For each of the following metrics: latency, sync, availability and stake we do the following:
// 0. Assume 3 providers
// 1. Append relay data for both providers with random samples. The "better" provider will have a randomized
// sample with a better range (for example, the better one gets latency of 10-30ms and the bad one gets 25-40ms)
// 2. Choose between them and verify the better one is chosen more.
func TestProviderOptimizerChoiceSimulationBasedOnLatency(t *testing.T) {
	rand.InitRandomSeed()
	providerOptimizer := setupProviderOptimizer(1)
	providersCount := 3
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)
	cu := uint64(10)
	requestBlock := int64(1000)
	syncBlock := uint64(1000)
	baseLatency := TEST_BASE_WORLD_LATENCY.Seconds()
	providerOptimizer.OptimizerNumTiers = 4
	providerOptimizer.OptimizerMinTierEntries = 1

	// initial values
	p1Latency := baseLatency * float64(time.Millisecond)
	p2Latency := baseLatency * float64(time.Millisecond)
	p3Latency := baseLatency * float64(time.Millisecond)
	p1SyncBlock := syncBlock
	p2SyncBlock := syncBlock
	p3SyncBlock := syncBlock
	p1Availability := true
	p2Availability := true
	p3Availability := true
	// append relay data for each provider depending on its index in the providers array
	// the latency gets worse for increasing index so we assume the best provider is the 1st
	// address, after it the 2nd and so on
	for i := 0; i < 1000; i++ {
		// randomize latency, provider 0 gets a better latency than provider 1
		p1Latency += 10 * float64(time.Millisecond)
		p2Latency += 20 * float64(time.Millisecond)
		p3Latency += 30 * float64(time.Millisecond)

		// randomize sync, provider 0 gets a better sync than provider 1
		p1SyncBlock++
		p2SyncBlock++
		p3SyncBlock++

		time.Sleep(1 * time.Millisecond)
		providerOptimizer.appendRelayData(providersGen.providersAddresses[0], time.Duration(p1Latency), p1Availability, cu, p1SyncBlock, time.Now())
		providerOptimizer.appendRelayData(providersGen.providersAddresses[1], time.Duration(p2Latency), p2Availability, cu, p2SyncBlock, time.Now())
		providerOptimizer.appendRelayData(providersGen.providersAddresses[2], time.Duration(p3Latency), p3Availability, cu, p3SyncBlock, time.Now())
	}

	// choose many times and check the better provider is chosen more often (provider 0)
	iterations := 1000
	res, tierResults := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, iterations, cu, requestBlock)
	utils.LavaFormatInfo("res", utils.LogAttr("res", res), utils.LogAttr("tierResults", tierResults))
	require.Greater(t, res[providersGen.providersAddresses[0]], res[providersGen.providersAddresses[1]])
	require.Greater(t, res[providersGen.providersAddresses[0]], res[providersGen.providersAddresses[2]])
	require.Greater(t, res[providersGen.providersAddresses[1]], res[providersGen.providersAddresses[2]])
}

func TestProviderOptimizerChoiceSimulationBasedOnSync(t *testing.T) {
	rand.InitRandomSeed()
	providerOptimizer := setupProviderOptimizer(1)
	providersCount := 3
	providersGen := (&providersGenerator{}).setupProvidersForTest(providersCount)
	cu := uint64(10)
	syncBlock := uint64(1000)
	baseLatency := TEST_BASE_WORLD_LATENCY.Seconds()
	providerOptimizer.OptimizerNumTiers = 4
	providerOptimizer.OptimizerMinTierEntries = 1

	// initial values
	p1Latency := baseLatency * float64(time.Millisecond)
	p2Latency := baseLatency * float64(time.Millisecond)
	p3Latency := baseLatency * float64(time.Millisecond)
	p1SyncBlock := syncBlock
	p2SyncBlock := syncBlock
	p3SyncBlock := syncBlock
	p1Availability := true
	p2Availability := true
	p3Availability := true
	// append relay data for each provider depending on its index in the providers array
	// the latency gets worse for increasing index so we assume the best provider is the 1st
	// address, after it the 2nd and so on
	sampleTime := time.Now()
	for i := 0; i < 1000; i++ {
		// randomize latency, provider 0 gets a better latency than provider 1
		p1Latency += 10 * float64(time.Millisecond)
		p2Latency += 10 * float64(time.Millisecond)
		p3Latency += 10 * float64(time.Millisecond)

		// randomize sync, provider 0 gets a better sync than provider 1
		p1SyncBlock++
		if i%100 == 1 {
			p1SyncBlock++
		}
		if i%300 == 1 {
			p2SyncBlock++
		}
		p2SyncBlock++
		p3SyncBlock++

		time.Sleep(1 * time.Millisecond)
		providerOptimizer.appendRelayData(providersGen.providersAddresses[0], time.Duration(p1Latency), p1Availability, cu, p1SyncBlock, sampleTime)
		providerOptimizer.appendRelayData(providersGen.providersAddresses[1], time.Duration(p2Latency), p2Availability, cu, p2SyncBlock, sampleTime)
		providerOptimizer.appendRelayData(providersGen.providersAddresses[2], time.Duration(p3Latency), p3Availability, cu, p3SyncBlock, sampleTime)
	}
	// choose many times and check the better provider is chosen more often (provider 0)
	iterations := 1000
	res, tierResults := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, iterations, cu, int64(p1SyncBlock))

	utils.LavaFormatInfo("res", utils.LogAttr("res", res), utils.LogAttr("tierResults", tierResults))
	require.Greater(t, res[providersGen.providersAddresses[0]], res[providersGen.providersAddresses[1]])
	require.Greater(t, res[providersGen.providersAddresses[0]], res[providersGen.providersAddresses[2]])
	require.Greater(t, res[providersGen.providersAddresses[1]], res[providersGen.providersAddresses[2]])
}

// TestProviderOptimizerLatencySyncScore tests that a provider with 100ms latency and x sync block
// has the same score as a provider with 1100ms latency but x+1 sync block
// This is true since the average block time is 10sec and the default sync factor is 0.3. So
// score_good_latency = latency + sync_factor * sync_lag + ... = 0.01 + 0.3 * 10 + ... = 3.01 + ...
// score_good_sync = latency + sync_factor * sync_lag + ... = 3.01 + 0.3 * 0 + ... = 3.01 + ...
func TestProviderOptimizerLatencySyncScore(t *testing.T) {
	rand.InitRandomSeed()
	providerOptimizer := setupProviderOptimizer(1)
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
	res, _ := runChooseManyTimesAndReturnResults(t, providerOptimizer, providersGen.providersAddresses, nil, iterations, cu, requestBlock)
	require.InDelta(t, res[providersGen.providersAddresses[0]], res[providersGen.providersAddresses[1]], float64(iterations)*0.1)
}
