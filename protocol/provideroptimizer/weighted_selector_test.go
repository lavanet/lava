package provideroptimizer

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/utils/rand"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	"github.com/stretchr/testify/require"
)

// Helper function to create a QoS report
func createQoSReport(availability, latency, sync float64) *pairingtypes.QualityOfServiceReport {
	return &pairingtypes.QualityOfServiceReport{
		Availability: math.LegacyMustNewDecFromStr(formatFloat(availability)),
		Latency:      math.LegacyMustNewDecFromStr(formatFloat(latency)),
		Sync:         math.LegacyMustNewDecFromStr(formatFloat(sync)),
	}
}

func formatFloat(f float64) string {
	return math.LegacyNewDec(int64(f * 1000000)).QuoInt64(1000000).String()
}

// TestNewWeightedSelector tests the creation of a new WeightedSelector
func TestNewWeightedSelector(t *testing.T) {
	config := DefaultWeightedSelectorConfig()
	ws := NewWeightedSelector(config)

	require.NotNil(t, ws)
	require.Equal(t, 0.4, ws.availabilityWeight)
	require.Equal(t, 0.3, ws.latencyWeight)
	require.Equal(t, 0.2, ws.syncWeight)
	require.Equal(t, 0.1, ws.stakeWeight)
	require.Equal(t, 0.10, ws.minSelectionChance)
}

// TestWeightNormalization tests that weights are normalized if they don't sum to 1.0
func TestWeightNormalization(t *testing.T) {
	config := WeightedSelectorConfig{
		AvailabilityWeight: 0.5,
		LatencyWeight:      0.5,
		SyncWeight:         0.5,
		StakeWeight:        0.5,
		MinSelectionChance: 0.01,
		Strategy:           StrategyBalanced,
	}

	ws := NewWeightedSelector(config)

	// Weights should be normalized to sum to 1.0
	totalWeight := ws.availabilityWeight + ws.latencyWeight + ws.syncWeight + ws.stakeWeight
	require.InDelta(t, 1.0, totalWeight, 0.001)

	// Each weight should be 0.25 (0.5/2.0)
	require.InDelta(t, 0.25, ws.availabilityWeight, 0.001)
	require.InDelta(t, 0.25, ws.latencyWeight, 0.001)
	require.InDelta(t, 0.25, ws.syncWeight, 0.001)
	require.InDelta(t, 0.25, ws.stakeWeight, 0.001)
}

// TestCalculateScorePerfectProvider tests scoring for a perfect provider
func TestCalculateScorePerfectProvider(t *testing.T) {
	config := DefaultWeightedSelectorConfig()
	ws := NewWeightedSelector(config)

	qos := createQoSReport(1.0, 0.0, 0.0) // Perfect availability, latency, sync
	stake := sdk.NewCoin("ulava", math.NewInt(1000))
	totalStake := sdk.NewCoin("ulava", math.NewInt(10000))

	score := ws.CalculateScore(qos, stake, totalStake)

	// Perfect provider should have high score (close to 1.0)
	// availability: 1.0 * 0.4 = 0.4
	// latency: 1.0 * 0.3 = 0.3 (0 latency normalized to 1.0)
	// sync: 1.0 * 0.2 = 0.2 (0 sync normalized to 1.0)
	// stake: 0.1 * 0.1 = 0.01 (1000/10000 = 0.1)
	// total: 0.4 + 0.3 + 0.2 + 0.01 = 0.91
	require.InDelta(t, 0.91, score, 0.02)
}

// TestCalculateScorePoorProvider tests scoring for a poor provider
func TestCalculateScorePoorProvider(t *testing.T) {
	config := DefaultWeightedSelectorConfig()
	ws := NewWeightedSelector(config)

	qos := createQoSReport(0.5, 1.0, 60.0) // Poor availability, high latency, poor sync
	stake := sdk.NewCoin("ulava", math.NewInt(100))
	totalStake := sdk.NewCoin("ulava", math.NewInt(10000))

	score := ws.CalculateScore(qos, stake, totalStake)

	// Poor provider should have lower score
	// availability: 0.5 * 0.4 = 0.2
	// latency: 0.0 * 0.3 = 0.0 (1s latency normalized to 0)
	// sync: 0.0 * 0.2 = 0.0 (60s sync normalized to 0)
	// stake: 0.01 * 0.1 = 0.001 (100/10000 = 0.01)
	// total: 0.2 + 0.0 + 0.0 + 0.001 = 0.201
	require.InDelta(t, 0.20, score, 0.05)
}

// TestCalculateScoreMinimumChance ensures minimum selection chance is enforced
func TestCalculateScoreMinimumChance(t *testing.T) {
	config := DefaultWeightedSelectorConfig()
	ws := NewWeightedSelector(config)

	qos := createQoSReport(0.0, 10.0, 1000.0) // Terrible metrics
	stake := sdk.NewCoin("ulava", math.NewInt(1))
	totalStake := sdk.NewCoin("ulava", math.NewInt(1000000))

	score := ws.CalculateScore(qos, stake, totalStake)

	// Even terrible provider should get minimum chance
	require.GreaterOrEqual(t, score, 0.01)
}

// TestNormalizeLatency tests latency normalization
func TestNormalizeLatency(t *testing.T) {
	config := DefaultWeightedSelectorConfig()
	ws := NewWeightedSelector(config)

	testCases := []struct {
		name     string
		latency  float64
		expected float64
	}{
		{"zero latency", 0.0, 1.0},
		{"low latency", 0.1, 0.9},
		{"medium latency", 0.5, 0.5},
		{"high latency", 1.0, 0.0},
		{"very high latency", 2.0, 0.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			normalized := ws.normalizeLatency(tc.latency)
			require.InDelta(t, tc.expected, normalized, 0.01)
		})
	}
}

// TestNormalizeSync tests sync normalization
func TestNormalizeSync(t *testing.T) {
	config := DefaultWeightedSelectorConfig()
	ws := NewWeightedSelector(config)

	testCases := []struct {
		name     string
		sync     float64
		expected float64
	}{
		{"zero sync lag", 0.0, 1.0},
		{"low sync lag", 6.0, 0.9},
		{"medium sync lag", 30.0, 0.5},
		{"high sync lag", 60.0, 0.0},
		{"very high sync lag", 120.0, 0.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			normalized := ws.normalizeSync(tc.sync)
			require.InDelta(t, tc.expected, normalized, 0.01)
		})
	}
}

// TestNormalizeStake tests stake normalization
func TestNormalizeStake(t *testing.T) {
	config := DefaultWeightedSelectorConfig()
	ws := NewWeightedSelector(config)

	testCases := []struct {
		name       string
		stake      int64
		totalStake int64
		expected   float64
	}{
		{"zero stake", 0, 10000, 0.0},
		{"small stake", 100, 10000, 0.01},
		{"medium stake", 2500, 10000, 0.25},
		{"large stake", 5000, 10000, 0.5},
		{"majority stake", 9000, 10000, 0.9},
		{"full stake", 10000, 10000, 1.0},
		{"exceeds total", 15000, 10000, 1.0}, // Capped at 1.0
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stake := sdk.NewCoin("ulava", math.NewInt(tc.stake))
			totalStake := sdk.NewCoin("ulava", math.NewInt(tc.totalStake))
			normalized := ws.normalizeStake(stake, totalStake)
			require.InDelta(t, tc.expected, normalized, 0.01)
		})
	}
}

// TestSelectProviderSingleProvider tests selection with only one provider
func TestSelectProviderSingleProvider(t *testing.T) {
	config := DefaultWeightedSelectorConfig()
	ws := NewWeightedSelector(config)

	providers := []ProviderScore{
		{Address: "provider1", CompositeScore: 0.8, SelectionWeight: 0.8},
	}

	selected := ws.SelectProvider(providers)
	require.Equal(t, "provider1", selected)
}

// TestSelectProviderEmptyList tests selection with empty provider list
func TestSelectProviderEmptyList(t *testing.T) {
	config := DefaultWeightedSelectorConfig()
	ws := NewWeightedSelector(config)

	providers := []ProviderScore{}

	selected := ws.SelectProvider(providers)
	require.Equal(t, "", selected)
}

// TestSelectProviderDistribution tests that selection follows probability distribution
func TestSelectProviderDistribution(t *testing.T) {
	config := DefaultWeightedSelectorConfig()
	ws := NewWeightedSelector(config)
	ws.SetDeterministicSeed(1234567) // Use fixed seed for deterministic test

	// Create providers with different scores
	providers := []ProviderScore{
		{Address: "high_score", CompositeScore: 0.8, SelectionWeight: 0.8},
		{Address: "medium_score", CompositeScore: 0.4, SelectionWeight: 0.4},
		{Address: "low_score", CompositeScore: 0.2, SelectionWeight: 0.2},
	}

	// Run many selections and count results
	selections := make(map[string]int)
	iterations := 10000

	for i := 0; i < iterations; i++ {
		selected := ws.SelectProvider(providers)
		selections[selected]++
	}

	// Expected probabilities:
	// high_score: 0.8 / (0.8 + 0.4 + 0.2) = 0.8 / 1.4 ≈ 0.571 (57.1%)
	// medium_score: 0.4 / 1.4 ≈ 0.286 (28.6%)
	// low_score: 0.2 / 1.4 ≈ 0.143 (14.3%)

	highScorePct := float64(selections["high_score"]) / float64(iterations)
	mediumScorePct := float64(selections["medium_score"]) / float64(iterations)
	lowScorePct := float64(selections["low_score"]) / float64(iterations)

	// Allow 5% deviation from expected
	require.InDelta(t, 0.571, highScorePct, 0.05, "high_score selection rate")
	require.InDelta(t, 0.286, mediumScorePct, 0.05, "medium_score selection rate")
	require.InDelta(t, 0.143, lowScorePct, 0.05, "low_score selection rate")

	t.Logf("Selection distribution over %d iterations:", iterations)
	t.Logf("  high_score: %.2f%% (expected ~57.1%%)", highScorePct*100)
	t.Logf("  medium_score: %.2f%% (expected ~28.6%%)", mediumScorePct*100)
	t.Logf("  low_score: %.2f%% (expected ~14.3%%)", lowScorePct*100)
}

// TestSelectProviderEqualScores tests selection when all providers have equal scores
func TestSelectProviderEqualScores(t *testing.T) {
	config := DefaultWeightedSelectorConfig()
	ws := NewWeightedSelector(config)
	ws.SetDeterministicSeed(1234567) // Use fixed seed for deterministic test

	providers := []ProviderScore{
		{Address: "provider1", CompositeScore: 0.5, SelectionWeight: 0.5},
		{Address: "provider2", CompositeScore: 0.5, SelectionWeight: 0.5},
		{Address: "provider3", CompositeScore: 0.5, SelectionWeight: 0.5},
	}

	selections := make(map[string]int)
	iterations := 3000

	for i := 0; i < iterations; i++ {
		selected := ws.SelectProvider(providers)
		selections[selected]++
	}

	// Each provider should be selected approximately equally (~33.3%)
	for _, provider := range providers {
		pct := float64(selections[provider.Address]) / float64(iterations)
		require.InDelta(t, 0.333, pct, 0.05)
	}
}

// TestSelectProviderZeroScores tests fallback when all scores are zero
func TestSelectProviderZeroScores(t *testing.T) {
	config := DefaultWeightedSelectorConfig()
	ws := NewWeightedSelector(config)
	ws.SetDeterministicSeed(1234567) // Use fixed seed for deterministic test

	providers := []ProviderScore{
		{Address: "provider1", CompositeScore: 0.0, SelectionWeight: 0.0},
		{Address: "provider2", CompositeScore: 0.0, SelectionWeight: 0.0},
		{Address: "provider3", CompositeScore: 0.0, SelectionWeight: 0.0},
	}

	// Should still select a provider (uniform random)
	selected := ws.SelectProvider(providers)
	require.NotEmpty(t, selected)
	require.Contains(t, []string{"provider1", "provider2", "provider3"}, selected)
}

// TestStrategyLatencyAdjustment tests latency strategy score adjustments
func TestStrategyLatencyAdjustment(t *testing.T) {
	config := DefaultWeightedSelectorConfig()
	config.Strategy = StrategyLatency
	ws := NewWeightedSelector(config)

	// High latency score should be boosted by strategy
	latency, sync := ws.applyStrategyAdjustments(0.8, 0.5)

	// Latency strategy should boost good latency scores
	require.Greater(t, latency, 0.75)
	require.InDelta(t, 0.5, sync, 0.01) // Sync should be unchanged
}

// TestStrategySyncAdjustment tests sync strategy score adjustments
func TestStrategySyncAdjustment(t *testing.T) {
	config := DefaultWeightedSelectorConfig()
	config.Strategy = StrategySyncFreshness
	ws := NewWeightedSelector(config)

	// High sync score should be boosted by strategy
	latency, sync := ws.applyStrategyAdjustments(0.5, 0.8)

	require.InDelta(t, 0.5, latency, 0.01) // Latency should be unchanged
	require.Greater(t, sync, 0.75)         // Sync should be boosted
}

// TestCalculateProviderScores tests the full score calculation pipeline
func TestCalculateProviderScores(t *testing.T) {
	config := DefaultWeightedSelectorConfig()
	ws := NewWeightedSelector(config)

	allAddresses := []string{"provider1", "provider2", "provider3"}
	ignoredProviders := map[string]struct{}{}

	providerData := map[string]*pairingtypes.QualityOfServiceReport{
		"provider1": createQoSReport(1.0, 0.1, 1.0),
		"provider2": createQoSReport(0.8, 0.3, 5.0),
		"provider3": createQoSReport(0.6, 0.5, 10.0),
	}

	stakes := map[string]int64{
		"provider1": 5000,
		"provider2": 3000,
		"provider3": 2000,
	}

	providerDataGetter := func(addr string) (*pairingtypes.QualityOfServiceReport, time.Time, bool) {
		qos, ok := providerData[addr]
		return qos, time.Now(), ok
	}

	stakeGetter := func(addr string) int64 {
		return stakes[addr]
	}

	scores, qosReports := ws.CalculateProviderScores(allAddresses, ignoredProviders, providerDataGetter, stakeGetter)

	require.Len(t, scores, 3)
	require.Len(t, qosReports, 3)

	// provider1 should have highest score (best metrics, highest stake)
	require.Greater(t, scores[0].CompositeScore, scores[1].CompositeScore)
	require.Greater(t, scores[1].CompositeScore, scores[2].CompositeScore)

	// Verify all providers are present in reports
	for _, addr := range allAddresses {
		_, ok := qosReports[addr]
		require.True(t, ok, "QoS report missing for %s", addr)
	}
}

// TestCalculateProviderScoresWithIgnored tests that ignored providers are skipped
func TestCalculateProviderScoresWithIgnored(t *testing.T) {
	config := DefaultWeightedSelectorConfig()
	ws := NewWeightedSelector(config)

	allAddresses := []string{"provider1", "provider2", "provider3"}
	ignoredProviders := map[string]struct{}{
		"provider2": {},
	}

	providerData := map[string]*pairingtypes.QualityOfServiceReport{
		"provider1": createQoSReport(1.0, 0.1, 1.0),
		"provider2": createQoSReport(0.8, 0.3, 5.0),
		"provider3": createQoSReport(0.6, 0.5, 10.0),
	}

	stakes := map[string]int64{
		"provider1": 5000,
		"provider2": 3000,
		"provider3": 2000,
	}

	providerDataGetter := func(addr string) (*pairingtypes.QualityOfServiceReport, time.Time, bool) {
		qos, ok := providerData[addr]
		return qos, time.Now(), ok
	}

	stakeGetter := func(addr string) int64 {
		return stakes[addr]
	}

	scores, qosReports := ws.CalculateProviderScores(allAddresses, ignoredProviders, providerDataGetter, stakeGetter)

	// Should only have 2 providers (provider2 is ignored)
	require.Len(t, scores, 2)
	require.Len(t, qosReports, 2)

	// Verify provider2 is not in results
	for _, score := range scores {
		require.NotEqual(t, "provider2", score.Address)
	}
	_, ok := qosReports["provider2"]
	require.False(t, ok)
}

// TestGetConfig tests retrieving the configuration
func TestGetConfig(t *testing.T) {
	originalConfig := WeightedSelectorConfig{
		AvailabilityWeight: 0.5,
		LatencyWeight:      0.3,
		SyncWeight:         0.15,
		StakeWeight:        0.05,
		MinSelectionChance: 0.02,
		Strategy:           StrategyLatency,
	}

	ws := NewWeightedSelector(originalConfig)
	retrievedConfig := ws.GetConfig()

	// Weights will be normalized, so check they sum to 1.0
	totalWeight := retrievedConfig.AvailabilityWeight +
		retrievedConfig.LatencyWeight +
		retrievedConfig.SyncWeight +
		retrievedConfig.StakeWeight
	require.InDelta(t, 1.0, totalWeight, 0.001)
	require.Equal(t, originalConfig.MinSelectionChance, retrievedConfig.MinSelectionChance)
	require.Equal(t, originalConfig.Strategy, retrievedConfig.Strategy)
}

// TestUpdateStrategy tests changing the strategy
func TestUpdateStrategy(t *testing.T) {
	config := DefaultWeightedSelectorConfig()
	ws := NewWeightedSelector(config)

	require.Equal(t, StrategyBalanced, ws.strategy)

	ws.UpdateStrategy(StrategyLatency)
	require.Equal(t, StrategyLatency, ws.strategy)

	ws.UpdateStrategy(StrategySyncFreshness)
	require.Equal(t, StrategySyncFreshness, ws.strategy)
}

// BenchmarkCalculateScore benchmarks score calculation
func BenchmarkCalculateScore(b *testing.B) {
	config := DefaultWeightedSelectorConfig()
	ws := NewWeightedSelector(config)

	qos := createQoSReport(0.95, 0.15, 2.5)
	stake := sdk.NewCoin("ulava", math.NewInt(5000))
	totalStake := sdk.NewCoin("ulava", math.NewInt(50000))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ws.CalculateScore(qos, stake, totalStake)
	}
}

// BenchmarkSelectProvider benchmarks provider selection
func BenchmarkSelectProvider(b *testing.B) {
	config := DefaultWeightedSelectorConfig()
	ws := NewWeightedSelector(config)
	ws.SetDeterministicSeed(1234567) // Use fixed seed for deterministic benchmark

	providers := make([]ProviderScore, 50)
	for i := 0; i < 50; i++ {
		providers[i] = ProviderScore{
			Address:         "provider" + string(rune(i)),
			CompositeScore:  0.5 + float64(i)*0.01,
			SelectionWeight: 0.5 + float64(i)*0.01,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ws.SelectProvider(providers)
	}
}

// BenchmarkCalculateProviderScores benchmarks full score calculation pipeline
func BenchmarkCalculateProviderScores(b *testing.B) {
	config := DefaultWeightedSelectorConfig()
	ws := NewWeightedSelector(config)

	allAddresses := make([]string, 50)
	providerData := make(map[string]*pairingtypes.QualityOfServiceReport)
	stakes := make(map[string]int64)

	for i := 0; i < 50; i++ {
		addr := "provider" + string(rune(i))
		allAddresses[i] = addr
		providerData[addr] = createQoSReport(0.9, 0.2, 3.0)
		stakes[addr] = 1000 + int64(i)*100
	}

	ignoredProviders := map[string]struct{}{}

	providerDataGetter := func(addr string) (*pairingtypes.QualityOfServiceReport, time.Time, bool) {
		qos, ok := providerData[addr]
		return qos, time.Now(), ok
	}

	stakeGetter := func(addr string) int64 {
		return stakes[addr]
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ws.CalculateProviderScores(allAddresses, ignoredProviders, providerDataGetter, stakeGetter)
	}
}

// TestQoSReportGeneration tests that QoS reports are generated correctly
func TestQoSReportGeneration(t *testing.T) {
	config := DefaultWeightedSelectorConfig()
	ws := NewWeightedSelector(config)

	allAddresses := []string{"provider1"}
	ignoredProviders := map[string]struct{}{}

	expectedLatency := 0.15
	expectedSync := 2.5
	expectedAvailability := 0.95

	providerData := map[string]*pairingtypes.QualityOfServiceReport{
		"provider1": createQoSReport(expectedAvailability, expectedLatency, expectedSync),
	}

	stakes := map[string]int64{
		"provider1": 5000,
	}

	lastUpdateTime := time.Now().Add(-5 * time.Minute)

	providerDataGetter := func(addr string) (*pairingtypes.QualityOfServiceReport, time.Time, bool) {
		qos, ok := providerData[addr]
		return qos, lastUpdateTime, ok
	}

	stakeGetter := func(addr string) int64 {
		return stakes[addr]
	}

	_, qosReports := ws.CalculateProviderScores(allAddresses, ignoredProviders, providerDataGetter, stakeGetter)

	require.Len(t, qosReports, 1)
	report := qosReports["provider1"]

	require.Equal(t, "provider1", report.ProviderAddress)
	require.InDelta(t, expectedAvailability, report.AvailabilityScore, 0.01)
	require.InDelta(t, expectedLatency, report.LatencyScore, 0.01)
	require.InDelta(t, expectedSync, report.SyncScore, 0.01)
	require.Greater(t, report.GenericScore, 0.0)
}

// TestStrategyDistributedFlattening tests that distributed strategy flattens the score curve
func TestStrategyDistributedFlattening(t *testing.T) {
	balancedConfig := DefaultWeightedSelectorConfig()
	balancedConfig.Strategy = StrategyBalanced
	balancedWS := NewWeightedSelector(balancedConfig)

	distributedConfig := DefaultWeightedSelectorConfig()
	distributedConfig.Strategy = StrategyDistributed
	distributedWS := NewWeightedSelector(distributedConfig)

	// Apply strategy adjustments
	balancedLatency, balancedSync := balancedWS.applyStrategyAdjustments(0.8, 0.8)
	distributedLatency, distributedSync := distributedWS.applyStrategyAdjustments(0.8, 0.8)

	// Distributed strategy should flatten scores (make them lower)
	require.Less(t, distributedLatency, balancedLatency)
	require.Less(t, distributedSync, balancedSync)
}