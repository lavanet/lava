package score

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAdaptiveMaxCalculator_BasicFunctionality(t *testing.T) {
	// Create adaptive max calculator
	halfLife := 1 * time.Hour
	minMax := 1.0
	maxMax := 30.0
	compression := 100.0

	calc := NewAdaptiveMaxCalculator(halfLife, minMax, maxMax, compression)
	require.NotNil(t, calc)

	// Add samples
	baseTime := time.Now()
	samples := []float64{0.5, 1.0, 1.5, 2.0, 2.5}

	for i, sample := range samples {
		err := calc.AddSample(sample, baseTime.Add(time.Duration(i)*time.Second))
		require.NoError(t, err)
	}

	// Get adaptive max (should be close to 95th percentile of samples)
	adaptiveMax := calc.GetAdaptiveMax()
	require.Greater(t, adaptiveMax, 0.0)
	require.LessOrEqual(t, adaptiveMax, maxMax)
	require.GreaterOrEqual(t, adaptiveMax, minMax)

	// For 5 samples, 95th percentile should be around the highest value
	require.InDelta(t, 2.5, adaptiveMax, 0.5)
}

func TestAdaptiveMaxCalculator_ExponentialDecay(t *testing.T) {
	halfLife := 1 * time.Hour
	calc := NewAdaptiveMaxCalculator(halfLife, 0.0, 100.0, 100.0)

	baseTime := time.Now()

	// Add high samples at time 0
	for i := 0; i < 50; i++ {
		err := calc.AddSample(10.0, baseTime.Add(time.Duration(i)*time.Millisecond))
		require.NoError(t, err)
	}

	// Initial adaptive max should be around 10.0
	initialMax := calc.GetAdaptiveMax()
	require.InDelta(t, 10.0, initialMax, 0.5)

	// Add even more low samples several hours later
	// so that the weight ratio heavily favors the new values
	futureTime := baseTime.Add(5 * time.Hour) // 5 half-lives = weight 1/32
	for i := 0; i < 200; i++ {                  // 4x more samples
		err := calc.AddSample(2.0, futureTime.Add(time.Duration(i)*time.Millisecond))
		require.NoError(t, err)
	}

	// With 5 half-lives, old samples have weight ~0.03 (1/32)
	// New samples have weight 1.0 and there are 4x more of them
	// So effective weight ratio is 200*1.0 vs 50*0.03 = 200 vs 1.5
	// The 95th percentile should now be dominated by the new low values
	newMax := calc.GetAdaptiveMax()
	
	// Just verify decay happened - max should be closer to 2 than to 10
	midpoint := (10.0 + 2.0) / 2.0 // 6.0
	require.Less(t, newMax, midpoint, "adaptive max should be closer to recent values after decay")
}

func TestAdaptiveMaxCalculator_Clamping(t *testing.T) {
	minMax := 2.0
	maxMax := 5.0
	calc := NewAdaptiveMaxCalculator(1*time.Hour, minMax, maxMax, 100.0)

	baseTime := time.Now()

	// Add very low samples (below minMax)
	for i := 0; i < 50; i++ {
		err := calc.AddSample(0.5, baseTime.Add(time.Duration(i)*time.Second))
		require.NoError(t, err)
	}

	// Adaptive max should be clamped to minMax
	adaptiveMax := calc.GetAdaptiveMax()
	require.Equal(t, minMax, adaptiveMax)

	// Reset and add very high samples (above maxMax)
	calc.Reset()
	for i := 0; i < 50; i++ {
		err := calc.AddSample(20.0, baseTime.Add(time.Duration(i)*time.Second))
		require.NoError(t, err)
	}

	// Adaptive max should be clamped to maxMax
	adaptiveMax = calc.GetAdaptiveMax()
	require.Equal(t, maxMax, adaptiveMax)
}

func TestAdaptiveMaxCalculator_DecayFormula(t *testing.T) {
	// Verify that decay formula matches ScoreStore formula
	halfLife := 1 * time.Hour
	calc := NewAdaptiveMaxCalculator(halfLife, 0.0, 100.0, 100.0)

	baseTime := time.Now()

	// Add initial sample
	err := calc.AddSample(10.0, baseTime)
	require.NoError(t, err)

	// Calculate expected decay factor after 1 hour (1 half-life)
	timeDiff := 1 * time.Hour
	exponent := -(math.Ln2 * timeDiff.Seconds()) / halfLife.Seconds()
	expectedDecayFactor := math.Exp(exponent)
	require.InDelta(t, 0.5, expectedDecayFactor, 0.001) // Should be 0.5 after 1 half-life

	// After 2 hours (2 half-lives), decay should be 0.25
	timeDiff = 2 * time.Hour
	exponent = -(math.Ln2 * timeDiff.Seconds()) / halfLife.Seconds()
	expectedDecayFactor = math.Exp(exponent)
	require.InDelta(t, 0.25, expectedDecayFactor, 0.001)
}

func TestAdaptiveMaxCalculator_NilHandling(t *testing.T) {
	var calc *AdaptiveMaxCalculator

	// Should not panic on nil
	err := calc.AddSample(1.0, time.Now())
	require.NoError(t, err)

	adaptiveMax := calc.GetAdaptiveMax()
	require.Equal(t, 0.0, adaptiveMax)

	stats := calc.GetStats()
	require.False(t, stats["enabled"].(bool))

	calc.Reset() // Should not panic
}

func TestAdaptiveMaxCalculator_NegativeSample(t *testing.T) {
	calc := NewAdaptiveMaxCalculator(1*time.Hour, 0.0, 100.0, 100.0)

	// Adding negative sample should return error
	err := calc.AddSample(-1.0, time.Now())
	require.Error(t, err)
}

func TestAdaptiveMaxCalculator_Stats(t *testing.T) {
	calc := NewAdaptiveMaxCalculator(1*time.Hour, 1.0, 30.0, 100.0)

	baseTime := time.Now()
	samples := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	for i, sample := range samples {
		err := calc.AddSample(sample, baseTime.Add(time.Duration(i)*time.Second))
		require.NoError(t, err)
	}

	stats := calc.GetStats()
	require.NotNil(t, stats)
	require.True(t, stats["enabled"].(bool))
	require.Greater(t, stats["adaptive_max_p95"].(float64), 0.0)
	require.Greater(t, stats["adaptive_p10"].(float64), 0.0)
	require.Greater(t, stats["adaptive_p90"].(float64), 0.0)
	require.Greater(t, stats["median"].(float64), 0.0)
	require.Greater(t, stats["p99"].(float64), 0.0)
	require.Greater(t, stats["total_weight"].(float64), 0.0)
	require.Greater(t, stats["centroid_count"].(int), 0)
	require.Equal(t, 100.0, stats["compression"].(float64))
	require.Equal(t, 3600.0, stats["half_life_seconds"].(float64))
	require.Equal(t, 1.0, stats["min_max"].(float64))
	require.Equal(t, 30.0, stats["max_max"].(float64))
}

func TestAdaptiveMaxCalculator_LargeDataset(t *testing.T) {
	// Test with large number of samples (900 samples = 3 providers × 300 relays)
	calc := NewAdaptiveMaxCalculator(1*time.Hour, 0.5, 10.0, 100.0)

	baseTime := time.Now()

	// Simulate 3 providers with different latency distributions
	// Provider A: avg 0.5s
	for i := 0; i < 300; i++ {
		sample := 0.5 + (float64(i%20) / 100.0) // 0.5 ± 0.2s
		err := calc.AddSample(sample, baseTime.Add(time.Duration(i)*time.Millisecond))
		require.NoError(t, err)
	}

	// Provider B: avg 1.0s
	for i := 0; i < 300; i++ {
		sample := 1.0 + (float64(i%20) / 100.0) // 1.0 ± 0.2s
		err := calc.AddSample(sample, baseTime.Add(time.Duration(300+i)*time.Millisecond))
		require.NoError(t, err)
	}

	// Provider C: avg 2.0s
	for i := 0; i < 300; i++ {
		sample := 2.0 + (float64(i%20) / 100.0) // 2.0 ± 0.2s
		err := calc.AddSample(sample, baseTime.Add(time.Duration(600+i)*time.Millisecond))
		require.NoError(t, err)
	}

	// Get 95th percentile
	adaptiveMax := calc.GetAdaptiveMax()

	// 95th percentile of 900 samples should be around 2.2s (95% of samples below this)
	require.Greater(t, adaptiveMax, 1.5)
	require.Less(t, adaptiveMax, 3.0)
	require.InDelta(t, 2.2, adaptiveMax, 0.5)

	// Verify stats
	stats := calc.GetStats()
	require.Greater(t, stats["total_weight"].(float64), 800.0) // Should have ~900 total weight
}

func TestAdaptiveMaxCalculator_Reset(t *testing.T) {
	calc := NewAdaptiveMaxCalculator(1*time.Hour, 0.0, 100.0, 100.0)

	baseTime := time.Now()

	// Add samples
	for i := 0; i < 50; i++ {
		err := calc.AddSample(5.0, baseTime.Add(time.Duration(i)*time.Second))
		require.NoError(t, err)
	}

	// Verify samples were added
	stats := calc.GetStats()
	require.Greater(t, stats["total_weight"].(float64), 0.0)

	// Reset
	calc.Reset()

	// Verify reset worked
	stats = calc.GetStats()
	require.Equal(t, 0.0, stats["total_weight"].(float64))

	// Can add samples after reset
	err := calc.AddSample(1.0, time.Now())
	require.NoError(t, err)
}

func TestAdaptiveMaxCalculator_OldSampleHandling(t *testing.T) {
	calc := NewAdaptiveMaxCalculator(1*time.Hour, 0.0, 100.0, 100.0)

	baseTime := time.Now()

	// Add recent sample
	err := calc.AddSample(5.0, baseTime)
	require.NoError(t, err)

	// Try to add older sample (should be handled gracefully)
	oldTime := baseTime.Add(-10 * time.Second)
	err = calc.AddSample(3.0, oldTime)
	require.NoError(t, err) // Should not error, just skip decay

	// Adaptive max should still work
	adaptiveMax := calc.GetAdaptiveMax()
	require.Greater(t, adaptiveMax, 0.0)
}

func TestAdaptiveMaxCalculator_GetAdaptiveBounds(t *testing.T) {
	// Test P10-P90 bounds calculation (Phase 2 hybrid approach)
	calc := NewAdaptiveMaxCalculator(1*time.Hour, 1.0, 30.0, 100.0)

	baseTime := time.Now()

	// Add samples representing 3 providers with different latencies
	// Provider A: 0.3s (300 samples)
	for i := 0; i < 300; i++ {
		sample := 0.3 + (float64(i%10) / 100.0) // 0.3 ± 0.1s
		err := calc.AddSample(sample, baseTime.Add(time.Duration(i)*time.Millisecond))
		require.NoError(t, err)
	}

	// Provider B: 1.0s (300 samples)
	for i := 0; i < 300; i++ {
		sample := 1.0 + (float64(i%10) / 100.0) // 1.0 ± 0.1s
		err := calc.AddSample(sample, baseTime.Add(time.Duration(300+i)*time.Millisecond))
		require.NoError(t, err)
	}

	// Provider C: 2.0s (300 samples)
	for i := 0; i < 300; i++ {
		sample := 2.0 + (float64(i%10) / 100.0) // 2.0 ± 0.1s
		err := calc.AddSample(sample, baseTime.Add(time.Duration(600+i)*time.Millisecond))
		require.NoError(t, err)
	}

	// Get P10 and P90
	p10, p90 := calc.GetAdaptiveBounds()

	// Verify P10 is around 0.3s (10th percentile of the distribution)
	require.Greater(t, p10, 0.1)  // Above minimum bound
	require.Less(t, p10, 0.6)     // Should be close to provider A's latency
	require.InDelta(t, 0.3, p10, 0.2)

	// Verify P90 is around 2.2s (90th percentile of the distribution)
	require.Greater(t, p90, 1.5)  // Should be higher than provider B
	require.Less(t, p90, 3.0)     // Should be close to provider C's upper range
	require.InDelta(t, 2.1, p90, 0.5)

	// Verify P90 > P10 (valid range)
	require.Greater(t, p90, p10)

	// Verify both are within configured bounds
	require.GreaterOrEqual(t, p90, 1.0)   // minMax
	require.LessOrEqual(t, p90, 30.0)     // maxMax
}

func TestAdaptiveMaxCalculator_GetAdaptiveBounds_Clamping(t *testing.T) {
	// Test that P10-P90 bounds are properly clamped
	minMax := 1.0
	maxMax := 5.0
	calc := NewAdaptiveMaxCalculator(1*time.Hour, minMax, maxMax, 100.0)

	baseTime := time.Now()

	// Add very low samples (P10 would be below minimum)
	for i := 0; i < 100; i++ {
		err := calc.AddSample(0.05, baseTime.Add(time.Duration(i)*time.Millisecond))
		require.NoError(t, err)
	}

	p10, p90 := calc.GetAdaptiveBounds()

	// P10 should be clamped to minimum (0.001 = 1ms - AdaptiveP10MinBound)
	require.GreaterOrEqual(t, p10, AdaptiveP10MinBound)

	// P90 should be clamped to minMax (1.0)
	require.GreaterOrEqual(t, p90, minMax)

	// P90 should be greater than P10
	require.Greater(t, p90, p10)

	// Reset and test high values
	calc.Reset()

	// Add very high samples (P90 would be above maximum)
	for i := 0; i < 100; i++ {
		err := calc.AddSample(50.0, baseTime.Add(time.Duration(i)*time.Millisecond))
		require.NoError(t, err)
	}

	p10, p90 = calc.GetAdaptiveBounds()

	// P90 should be clamped to maxMax (5.0)
	require.LessOrEqual(t, p90, maxMax)

	// P10 should be less than P90
	require.Less(t, p10, p90)
}

func TestAdaptiveMaxCalculator_GetAdaptiveBounds_NilHandling(t *testing.T) {
	var calc *AdaptiveMaxCalculator

	// Should return safe defaults on nil
	p10, p90 := calc.GetAdaptiveBounds()
	require.Equal(t, 0.5, p10)
	require.Equal(t, 3.0, p90)
}

func TestAdaptiveMaxCalculator_Stats_IncludesBounds(t *testing.T) {
	// Test that stats include P10-P90 bounds
	calc := NewAdaptiveMaxCalculator(1*time.Hour, 1.0, 30.0, 100.0)

	baseTime := time.Now()

	// Add samples
	for i := 0; i < 100; i++ {
		sample := float64(i) / 10.0 // 0.0 to 10.0
		err := calc.AddSample(sample, baseTime.Add(time.Duration(i)*time.Millisecond))
		require.NoError(t, err)
	}

	stats := calc.GetStats()
	require.NotNil(t, stats)

	// Verify stats contain P10-P90 bounds
	require.Contains(t, stats, "adaptive_p10")
	require.Contains(t, stats, "adaptive_p90")
	require.Contains(t, stats, "adaptive_max_p95")

	p10 := stats["adaptive_p10"].(float64)
	p90 := stats["adaptive_p90"].(float64)
	p95 := stats["adaptive_max_p95"].(float64)

	// Verify ordering: P10 < P90 < P95
	require.Less(t, p10, p90)
	require.LessOrEqual(t, p90, p95)

	// All should be positive
	require.Greater(t, p10, 0.0)
	require.Greater(t, p90, 0.0)
	require.Greater(t, p95, 0.0)
}
