package provideroptimizer

import (
	"testing"
	"time"

	"github.com/lavanet/lava/v5/utils/rand"
	"github.com/stretchr/testify/require"
)

// TestWeightedChoiceByQoS_BasicBehavior verifies that QoS-based selection favors lower scores
func TestWeightedChoiceByQoS_BasicBehavior(t *testing.T) {
	rand.InitRandomSeed()

	selectionWeighter := NewSelectionWeighter()

	// Create entries with different QoS scores (lower = better)
	entries := []Entry{
		{Address: "provider_best", Score: 0.1, Part: 1.0}, // Best QoS
		{Address: "provider_good", Score: 0.2, Part: 1.0}, // Good QoS
		{Address: "provider_fair", Score: 0.4, Part: 1.0}, // Fair QoS
		{Address: "provider_poor", Score: 0.8, Part: 1.0}, // Poor QoS
	}

	// Run many selections to verify statistical distribution
	selections := make(map[string]int)
	totalSelections := 10000

	for i := 0; i < totalSelections; i++ {
		selected := selectionWeighter.WeightedChoiceByQoS(entries)
		selections[selected]++
	}

	// Verify that best QoS provider gets significantly more selections
	bestSelections := selections["provider_best"]
	poorSelections := selections["provider_poor"]

	t.Logf("Selection results over %d iterations:", totalSelections)
	t.Logf("  Best (score 0.1): %d selections (%.1f%%)", bestSelections, float64(bestSelections)/float64(totalSelections)*100)
	t.Logf("  Good (score 0.2): %d selections (%.1f%%)", selections["provider_good"], float64(selections["provider_good"])/float64(totalSelections)*100)
	t.Logf("  Fair (score 0.4): %d selections (%.1f%%)", selections["provider_fair"], float64(selections["provider_fair"])/float64(totalSelections)*100)
	t.Logf("  Poor (score 0.8): %d selections (%.1f%%)", poorSelections, float64(poorSelections)/float64(totalSelections)*100)

	// Best provider should get at least 4x more selections than worst
	require.Greater(t, bestSelections, poorSelections*4,
		"Best QoS provider should get significantly more selections than worst")

	// Best provider should get at least 40% of all selections
	require.Greater(t, bestSelections, totalSelections*40/100,
		"Best QoS provider should get at least 40%% of selections")

	// Poor provider should get less than 15% of selections
	require.Less(t, poorSelections, totalSelections*15/100,
		"Poor QoS provider should get less than 15%% of selections")
}

// TestWeightedChoiceByQoS_EqualStakes verifies behavior with equal stakes
func TestWeightedChoiceByQoS_EqualStakes(t *testing.T) {
	rand.InitRandomSeed()

	selectionWeighter := NewSelectionWeighter()

	// Set equal stakes for all providers
	stakes := map[string]int64{
		"provider_A": 1000000,
		"provider_B": 1000000,
		"provider_C": 1000000,
		"provider_D": 1000000,
	}
	selectionWeighter.SetWeights(stakes)

	// Entries with different QoS scores but equal stakes
	entries := []Entry{
		{Address: "provider_A", Score: 0.10, Part: 1.0}, // 5ms latency
		{Address: "provider_B", Score: 0.15, Part: 1.0}, // 8ms latency
		{Address: "provider_C", Score: 0.25, Part: 1.0}, // 15ms latency
		{Address: "provider_D", Score: 0.40, Part: 1.0}, // 25ms latency
	}

	selections := make(map[string]int)
	totalSelections := 10000

	for i := 0; i < totalSelections; i++ {
		selected := selectionWeighter.WeightedChoiceByQoS(entries)
		selections[selected]++
	}

	// Calculate expected weights
	expectedWeightA := 1.0 / (0.10 + 0.0001)
	expectedWeightB := 1.0 / (0.15 + 0.0001)
	expectedWeightC := 1.0 / (0.25 + 0.0001)
	expectedWeightD := 1.0 / (0.40 + 0.0001)
	totalWeight := expectedWeightA + expectedWeightB + expectedWeightC + expectedWeightD

	expectedProbA := expectedWeightA / totalWeight
	expectedProbB := expectedWeightB / totalWeight
	expectedProbC := expectedWeightC / totalWeight
	expectedProbD := expectedWeightD / totalWeight

	actualProbA := float64(selections["provider_A"]) / float64(totalSelections)
	actualProbB := float64(selections["provider_B"]) / float64(totalSelections)
	actualProbC := float64(selections["provider_C"]) / float64(totalSelections)
	actualProbD := float64(selections["provider_D"]) / float64(totalSelections)

	t.Logf("Expected vs Actual probabilities:")
	t.Logf("  Provider A (0.10): Expected %.1f%%, Actual %.1f%%", expectedProbA*100, actualProbA*100)
	t.Logf("  Provider B (0.15): Expected %.1f%%, Actual %.1f%%", expectedProbB*100, actualProbB*100)
	t.Logf("  Provider C (0.25): Expected %.1f%%, Actual %.1f%%", expectedProbC*100, actualProbC*100)
	t.Logf("  Provider D (0.40): Expected %.1f%%, Actual %.1f%%", expectedProbD*100, actualProbD*100)

	// Verify within 5% tolerance (statistical variance)
	tolerance := 0.05
	require.InDelta(t, expectedProbA, actualProbA, tolerance, "Provider A probability mismatch")
	require.InDelta(t, expectedProbB, actualProbB, tolerance, "Provider B probability mismatch")
	require.InDelta(t, expectedProbC, actualProbC, tolerance, "Provider C probability mismatch")
	require.InDelta(t, expectedProbD, actualProbD, tolerance, "Provider D probability mismatch")
}

// TestWeightedChoiceByQoS_vs_StakeBased compares QoS vs stake-based selection
func TestWeightedChoiceByQoS_vs_StakeBased(t *testing.T) {
	rand.InitRandomSeed()

	selectionWeighter := NewSelectionWeighter()

	// Set equal stakes
	stakes := map[string]int64{
		"provider_1": 1000000,
		"provider_2": 1000000,
		"provider_3": 1000000,
		"provider_4": 1000000,
	}
	selectionWeighter.SetWeights(stakes)

	entries := []Entry{
		{Address: "provider_1", Score: 0.10, Part: 1.0}, // Best QoS
		{Address: "provider_2", Score: 0.15, Part: 1.0},
		{Address: "provider_3", Score: 0.25, Part: 1.0},
		{Address: "provider_4", Score: 0.40, Part: 1.0}, // Worst QoS
	}

	totalSelections := 10000

	// Test stake-based selection (should be ~25% each)
	stakeSelections := make(map[string]int)
	for i := 0; i < totalSelections; i++ {
		selected := selectionWeighter.WeightedChoice(entries)
		stakeSelections[selected]++
	}

	// Test QoS-based selection (should favor best QoS)
	qosSelections := make(map[string]int)
	for i := 0; i < totalSelections; i++ {
		selected := selectionWeighter.WeightedChoiceByQoS(entries)
		qosSelections[selected]++
	}

	t.Logf("Stake-based selection (equal stakes):")
	for i := 1; i <= 4; i++ {
		provider := "provider_" + string(rune('0'+i))
		pct := float64(stakeSelections[provider]) / float64(totalSelections) * 100
		t.Logf("  %s: %.1f%%", provider, pct)
	}

	t.Logf("\nQoS-based selection:")
	for i := 1; i <= 4; i++ {
		provider := "provider_" + string(rune('0'+i))
		pct := float64(qosSelections[provider]) / float64(totalSelections) * 100
		t.Logf("  %s: %.1f%%", provider, pct)
	}

	// Stake-based: all should be roughly equal (~25% each)
	for i := 1; i <= 4; i++ {
		provider := "provider_" + string(rune('0'+i))
		pct := float64(stakeSelections[provider]) / float64(totalSelections)
		require.InDelta(t, 0.25, pct, 0.05, "Stake-based should be ~25% for "+provider)
	}

	// QoS-based: best should have significantly more than worst
	best := float64(qosSelections["provider_1"]) / float64(totalSelections)
	worst := float64(qosSelections["provider_4"]) / float64(totalSelections)
	require.Greater(t, best, worst*3, "QoS-based: best should get 3x more than worst")
	require.Greater(t, best, 0.35, "QoS-based: best should get >35%")
	require.Less(t, worst, 0.15, "QoS-based: worst should get <15%")
}

// TestProviderOptimizer_FlagEnabled tests the optimizer with QoS flag enabled
func TestProviderOptimizer_FlagEnabled(t *testing.T) {
	rand.InitRandomSeed()

	// Create optimizer with QoS selection ENABLED
	optimizerQoS := NewProviderOptimizer(
		StrategyBalanced,
		10*time.Second,
		3,
		nil,
		"test",
		true, // ← QoS selection enabled
	)

	// Create optimizer with QoS selection DISABLED (default)
	optimizerStake := NewProviderOptimizer(
		StrategyBalanced,
		10*time.Second,
		3,
		nil,
		"test",
		false, // ← QoS selection disabled
	)

	// Setup providers with equal stakes
	providersGen := (&providersGenerator{}).setupProvidersForTest(4)

	stakes := make(map[string]int64)
	for _, addr := range providersGen.providersAddresses {
		stakes[addr] = 1000000 // Equal stakes
	}

	optimizerQoS.UpdateWeights(stakes, 1)
	optimizerStake.UpdateWeights(stakes, 1)

	// Append different QoS data for each provider
	baseLatency := 5 * time.Millisecond
	for i, provider := range providersGen.providersAddresses {
		// Provider 0: best (5ms), Provider 3: worst (20ms)
		latency := baseLatency * time.Duration(i+1)
		optimizerQoS.AppendProbeRelayData(provider, latency, true)
		optimizerStake.AppendProbeRelayData(provider, latency, true)
	}
	time.Sleep(4 * time.Millisecond) // Allow QoS data to be processed

	// Run selections
	totalSelections := 5000
	qosSelections := make(map[string]int)
	stakeSelections := make(map[string]int)

	for i := 0; i < totalSelections; i++ {
		// QoS-based optimizer
		qosProviders, _ := optimizerQoS.ChooseProvider(
			providersGen.providersAddresses,
			nil,
			1000,
			-1,
		)
		if len(qosProviders) > 0 {
			qosSelections[qosProviders[0]]++
		}

		// Stake-based optimizer
		stakeProviders, _ := optimizerStake.ChooseProvider(
			providersGen.providersAddresses,
			nil,
			1000,
			-1,
		)
		if len(stakeProviders) > 0 {
			stakeSelections[stakeProviders[0]]++
		}
	}

	bestProvider := providersGen.providersAddresses[0]
	worstProvider := providersGen.providersAddresses[3]

	t.Logf("QoS-enabled optimizer:")
	t.Logf("  Best provider (5ms):  %d selections (%.1f%%)",
		qosSelections[bestProvider],
		float64(qosSelections[bestProvider])/float64(totalSelections)*100)
	t.Logf("  Worst provider (20ms): %d selections (%.1f%%)",
		qosSelections[worstProvider],
		float64(qosSelections[worstProvider])/float64(totalSelections)*100)

	t.Logf("\nStake-based optimizer (flag=false):")
	t.Logf("  Best provider (5ms):  %d selections (%.1f%%)",
		stakeSelections[bestProvider],
		float64(stakeSelections[bestProvider])/float64(totalSelections)*100)
	t.Logf("  Worst provider (20ms): %d selections (%.1f%%)",
		stakeSelections[worstProvider],
		float64(stakeSelections[worstProvider])/float64(totalSelections)*100)

	// Verify QoS-enabled optimizer favors best provider
	qosBestRatio := float64(qosSelections[bestProvider]) / float64(qosSelections[worstProvider])
	require.Greater(t, qosBestRatio, 1.3,
		"QoS-enabled: best provider should get at least 1.3x more selections than worst")

	// Verify stake-based optimizer is more balanced
	stakeBestRatio := float64(stakeSelections[bestProvider]) / float64(stakeSelections[worstProvider])
	require.Less(t, stakeBestRatio, 1.3,
		"Stake-based: selection should be more balanced with equal stakes (ratio < 1.3x)")

	// Verify QoS-based selection shows clear preference compared to stake-based
	require.Greater(t, qosBestRatio, stakeBestRatio,
		"QoS-enabled should show stronger preference for best provider than stake-based")
}

// TestWeightedChoiceByQoS_FractionalPart tests QoS selection with fractional tier participation
func TestWeightedChoiceByQoS_FractionalPart(t *testing.T) {
	rand.InitRandomSeed()

	selectionWeighter := NewSelectionWeighter()

	// Entries with fractional participation (tier boundaries)
	entries := []Entry{
		{Address: "provider_A", Score: 0.10, Part: 0.3}, // Fractional (at boundary)
		{Address: "provider_B", Score: 0.15, Part: 1.0}, // Full
		{Address: "provider_C", Score: 0.25, Part: 1.0}, // Full
		{Address: "provider_D", Score: 0.40, Part: 0.7}, // Fractional (at boundary)
	}

	selections := make(map[string]int)
	totalSelections := 10000

	for i := 0; i < totalSelections; i++ {
		selected := selectionWeighter.WeightedChoiceByQoS(entries)
		selections[selected]++
	}

	t.Logf("Selection with fractional participation:")
	for _, entry := range entries {
		count := selections[entry.Address]
		pct := float64(count) / float64(totalSelections) * 100
		t.Logf("  %s (score %.2f, part %.1f): %d selections (%.1f%%)",
			entry.Address, entry.Score, entry.Part, count, pct)
	}

	// Provider A has best score but limited by Part=0.3
	// Should still get good selections due to excellent QoS
	providerA := selections["provider_A"]
	providerD := selections["provider_D"]

	// Even with 0.3 participation, best QoS should beat worst QoS
	require.Greater(t, providerA, providerD,
		"Best QoS (even with Part=0.3) should beat worst QoS")
}

// TestWeightedChoiceByQoS_EmptyList tests edge case with empty provider list
func TestWeightedChoiceByQoS_EmptyList(t *testing.T) {
	rand.InitRandomSeed()

	selectionWeighter := NewSelectionWeighter()

	entries := []Entry{}
	selected := selectionWeighter.WeightedChoiceByQoS(entries)

	require.Equal(t, "", selected, "Empty list should return empty string")
}

// TestWeightedChoiceByQoS_SingleProvider tests with only one provider
func TestWeightedChoiceByQoS_SingleProvider(t *testing.T) {
	rand.InitRandomSeed()

	selectionWeighter := NewSelectionWeighter()

	entries := []Entry{
		{Address: "provider_only", Score: 0.5, Part: 1.0},
	}

	// Should always select the only provider
	for i := 0; i < 100; i++ {
		selected := selectionWeighter.WeightedChoiceByQoS(entries)
		require.Equal(t, "provider_only", selected)
	}
}
