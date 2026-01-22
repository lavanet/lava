package score

import (
	"math"
	"sync"
	"time"

	tdigest "github.com/influxdata/tdigest"
	"github.com/lavanet/lava/v5/utils"
)

// AdaptiveMaxCalculator uses T-Digest with exponential decay to calculate
// adaptive maximum values for normalization. This approach uses the 95th percentile
// of all individual samples (not provider averages) with exponential decay weighting
// that aligns mathematically with the ScoreStore's decaying weighted average.
type AdaptiveMaxCalculator struct {
	digest     *tdigest.TDigest
	lastUpdate time.Time
	halfLife   time.Duration
	minMax     float64 // Minimum allowed max value (safety bound)
	maxMax     float64 // Maximum allowed max value (safety bound)
	mu         sync.RWMutex
}

// NewAdaptiveMaxCalculator creates a new adaptive max calculator with T-Digest
// and exponential decay support.
//
// Parameters:
//   - halfLife: The half-life for exponential decay (should match ScoreStore half-life)
//   - minMax: Minimum value to clamp adaptive max to (safety bound)
//   - maxMax: Maximum value to clamp adaptive max to (safety bound)
//   - compression: T-Digest compression parameter (100 is recommended)
func NewAdaptiveMaxCalculator(halfLife time.Duration, minMax, maxMax float64, compression float64) *AdaptiveMaxCalculator {
	if compression <= 0 {
		compression = 100 // Default compression
	}

	return &AdaptiveMaxCalculator{
		digest:     tdigest.NewWithCompression(compression),
		lastUpdate: time.Now(),
		halfLife:   halfLife,
		minMax:     minMax,
		maxMax:     maxMax,
	}
}

// AddSample adds a new sample to the T-Digest with exponential decay.
// This method applies exponential decay to all existing centroids (using the SAME
// formula as ScoreStore) and then adds the new sample with weight = 1.0.
//
// The decay formula is:
//
//	decayFactor = exp(-(ln(2) * timeDiff) / halfLife)
//
// This ensures that:
//   - Sample from 1 hour ago: 50% weight
//   - Sample from 2 hours ago: 25% weight
//   - Sample from 3 hours ago: 12.5% weight
//
// Parameters:
//   - value: The sample value to add
//   - sampleTime: The timestamp of the sample
func (a *AdaptiveMaxCalculator) AddSample(value float64, sampleTime time.Time) error {
	if a == nil {
		return nil // Gracefully handle nil (adaptive max might be disabled)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Validate input
	if value < 0 {
		return utils.LavaFormatError("cannot add negative sample to adaptive max calculator",
			nil,
			utils.LogAttr("value", value),
		)
	}

	// Calculate time difference since last update
	timeDiff := sampleTime.Sub(a.lastUpdate).Seconds()
	if timeDiff < 0 {
		// Sample is older than last update - this shouldn't happen in normal operation
		// but we handle it gracefully by not applying decay
		utils.LavaFormatTrace("adaptive max calculator: sample time is before last update, skipping decay",
			utils.LogAttr("sampleTime", sampleTime),
			utils.LogAttr("lastUpdate", a.lastUpdate),
		)
		timeDiff = 0
	}

	// Apply exponential decay to all existing centroids
	// This uses the SAME formula as ScoreStore.Update()
	if timeDiff > 0 {
		exponent := -(math.Ln2 * timeDiff) / a.halfLife.Seconds()
		decayFactor := math.Exp(exponent)

		if decayFactor > 1 {
			return utils.LavaFormatError("invalid decay factor > 1 in adaptive max calculator",
				nil,
				utils.LogAttr("decayFactor", decayFactor),
				utils.LogAttr("timeDiff", timeDiff),
				utils.LogAttr("halfLife", a.halfLife),
			)
		}

		// Scale all existing centroids by decay factor
		a.applyDecayToDigest(decayFactor)
	}

	// Add new sample with weight = 1.0
	a.digest.Add(value, 1.0)

	// Update last update time
	a.lastUpdate = sampleTime

	return nil
}

// applyDecayToDigest applies exponential decay to all centroids in the T-Digest.
// This is done by scaling all centroid weights by the decay factor.
//
// Note: The influxdata/tdigest library doesn't have a native Scale() method,
// so we implement decay by rebuilding the digest with decayed weights.
func (a *AdaptiveMaxCalculator) applyDecayToDigest(decayFactor float64) {
	// Get all centroids from the current digest
	centroids := a.digest.Centroids()

	if len(centroids) == 0 {
		return // Nothing to decay
	}

	// Create a new digest with the same compression
	compression := a.digest.Compression
	newDigest := tdigest.NewWithCompression(compression)

	// Add all centroids with decayed weights
	for _, centroid := range centroids {
		newWeight := centroid.Weight * decayFactor
		if newWeight > 0 {
			newDigest.Add(centroid.Mean, newWeight)
		}
	}

	// Replace the old digest with the new one
	a.digest = newDigest
}

// GetAdaptiveMax returns the 95th percentile from the T-Digest as the adaptive max,
// clamped to the configured [minMax, maxMax] range.
// DEPRECATED: Use GetAdaptiveBounds() for the P10-P90 approach (Phase 2 hybrid).
func (a *AdaptiveMaxCalculator) GetAdaptiveMax() float64 {
	if a == nil {
		return 0 // Gracefully handle nil (adaptive max might be disabled)
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	// Get 95th percentile from T-Digest
	percentile95 := a.digest.Quantile(0.95)

	// Clamp to reasonable range
	if percentile95 < a.minMax {
		percentile95 = a.minMax
	}
	if percentile95 > a.maxMax {
		percentile95 = a.maxMax
	}

	return percentile95
}

// GetAdaptiveBounds returns both P10 and P90 from the T-Digest for the
// P10-P90 adaptive normalization approach (Phase 2 hybrid).
// This provides better distribution and outlier handling than P95-only.
//
// Returns:
//   - p10: 10th percentile (adaptive minimum), clamped to reasonable bounds
//   - p90: 90th percentile (adaptive maximum), clamped to reasonable bounds
func (a *AdaptiveMaxCalculator) GetAdaptiveBounds() (p10, p90 float64) {
	if a == nil {
		// Return safe defaults if calculator is not initialized
		return 0.5, 3.0
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	// Get both P10 and P90 from T-Digest
	p10 = a.digest.Quantile(0.10)
	p90 = a.digest.Quantile(0.90)

	// Apply safety bounds
	// P10: minimum bound should be lower than P90's minimum
	if p10 < 0.1 {
		p10 = 0.1 // Minimum latency bound (100ms)
	}
	if p10 > 10.0 {
		p10 = 10.0 // Maximum for P10 (10 seconds)
	}

	// P90: use configured bounds
	if p90 < a.minMax {
		p90 = a.minMax
	}
	if p90 > a.maxMax {
		p90 = a.maxMax
	}

	// Ensure valid range: P90 must be greater than P10
	if p90 <= p10 {
		// If they're equal or inverted, adjust P90 to be at least 1 second above P10
		p90 = p10 + 1.0
		// But don't exceed maxMax
		if p90 > a.maxMax {
			p90 = a.maxMax
			p10 = p90 - 1.0
			if p10 < 0.1 {
				p10 = 0.1
			}
		}
	}

	return p10, p90
}

// GetStats returns statistics about the adaptive max calculator for debugging
func (a *AdaptiveMaxCalculator) GetStats() map[string]interface{} {
	if a == nil {
		return map[string]interface{}{"enabled": false}
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	// Get percentiles directly without calling GetAdaptiveBounds (avoid nested locks)
	p10Raw := a.digest.Quantile(0.10)
	p90Raw := a.digest.Quantile(0.90)

	// Apply same bounds logic as GetAdaptiveBounds
	p10 := p10Raw
	if p10 < 0.1 {
		p10 = 0.1
	}
	if p10 > 10.0 {
		p10 = 10.0
	}

	p90 := p90Raw
	if p90 < a.minMax {
		p90 = a.minMax
	}
	if p90 > a.maxMax {
		p90 = a.maxMax
	}

	if p90 <= p10 {
		p90 = p10 + 1.0
		if p90 > a.maxMax {
			p90 = a.maxMax
			p10 = p90 - 1.0
			if p10 < 0.1 {
				p10 = 0.1
			}
		}
	}

	return map[string]interface{}{
		"enabled":           true,
		"adaptive_max_p95":  a.digest.Quantile(0.95),
		"adaptive_p10":      p10,
		"adaptive_p90":      p90,
		"median":            a.digest.Quantile(0.5),
		"p99":               a.digest.Quantile(0.99),
		"total_weight":      a.digest.Count(),
		"centroid_count":    len(a.digest.Centroids()),
		"compression":       a.digest.Compression,
		"last_update":       a.lastUpdate,
		"half_life_seconds": a.halfLife.Seconds(),
		"min_max":           a.minMax,
		"max_max":           a.maxMax,
	}
}

// Reset clears the T-Digest and resets to initial state
func (a *AdaptiveMaxCalculator) Reset() {
	if a == nil {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	compression := a.digest.Compression
	a.digest = tdigest.NewWithCompression(compression)
	a.lastUpdate = time.Now()
}
