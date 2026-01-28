package score

import (
	"fmt"
	"time"
)

// Config defines a collection of parameters that can be used by ScoreStore. ScoreStore is a
// decaying weighted average object that is used to collect providers performance metrics samples.
// These are used to calculate the providers QoS excellence score, used by the provider optimizer
// when choosing providers to be paired with a consumer.
//
// Config parameters:
//   1. Weight: sample weight that multiplies the sample when updating the ScoreStore.
//   2. HalfLife: defines the half life time of the decaying exponent used in the ScoreStore.
//   3. LatencyCuFactor: an additional multiplier to latency samples that is determined by
//      the amount of CU used by the relay that the provider serviced.
//
// Additional info:
// CU latency factors are used to scale down high CU latencies when updating the latency ScoreStore
// so it can be safely added to the score average without bias (for example, a high CU
// latency sample from a TX might be 10sec while a low CU latency sample from a basic query might
// be 10ms and they're both considered good response time from the provider)
//
// TODO: high latency can be due to archive requests, addons, etc. This implementation
// is only partial since it considers only the CU amount

const (
	DefaultHalfLifeTime = time.Hour
	MaxHalfTime         = 3 * time.Hour

	DefaultWeight     float64 = 1
	ProbeUpdateWeight float64 = 0.25
	RelayUpdateWeight float64 = 1

	// TODO: find actual numbers from info of latencies of high/mid/low CU from "stats.lavanet.xyz".
	// Do a distribution and find average factor to multiply the failure cost by.
	DefaultCuLatencyFactor = LowCuLatencyFactor
	HighCuLatencyFactor    = 0.01       // for cu > HighCuThreshold
	MidCuLatencyFactor     = 0.1        // for MidCuThreshold < cu < HighCuThreshold
	LowCuLatencyFactor     = float64(1) // for cu < MidCuThreshold

	HighCuThreshold = uint64(100)
	MidCuThreshold  = uint64(50)

	// Adaptive Normalization Constants (Phase 2)
	// P10 bounds: Used to clamp the 10th percentile (adaptive minimum)
	AdaptiveP10MinBound float64 = 0.001 // 1ms - extremely fast local/optimized providers
	AdaptiveP10MaxBound float64 = 10.0  // 10s - maximum reasonable P10

	// P90 bounds: Use existing score bounds as fallback
	// AdaptiveP90MinBound = minMax parameter (typically 1.0s)
	// AdaptiveP90MaxBound = maxMax parameter (typically 30.0s = WorstLatencyScore)

	// T-Digest compression for adaptive max calculator
	DefaultTDigestCompression float64 = 100.0 // Good balance of accuracy vs memory

	// Default bounds for latency adaptive max calculator
	DefaultLatencyAdaptiveMinMax float64 = 1.0  // 1 second minimum for P90
	DefaultLatencyAdaptiveMaxMax float64 = 30.0 // 30 seconds maximum for P90 (WorstLatencyScore)

	// Sync-specific adaptive normalization constants (Phase 2)
	// P10 bounds for sync: Used to clamp the 10th percentile (adaptive minimum)
	AdaptiveSyncP10MinBound float64 = 0.1  // 100ms - very fast sync (perfect or near-perfect)
	AdaptiveSyncP10MaxBound float64 = 60.0 // 60s - maximum reasonable P10 for sync

	// Default bounds for sync adaptive max calculator
	DefaultSyncAdaptiveMinMax float64 = 30.0   // 30 seconds minimum for P90
	DefaultSyncAdaptiveMaxMax float64 = 1200.0 // 1200 seconds (20 min) maximum for P90 (WorstSyncScore)

	// Availability-specific normalization constants (Phase 1: Simple Rescaling)
	// Availability is already in [0,1] range but has poor distribution (typically 0.95-1.0 = only 5% of range)
	// Rescaling [MIN_ACCEPTABLE, 1.0] → [0.0, 1.0] achieves 100% range utilization
	MinAcceptableAvailability float64 = 0.80 // Below 80% availability = score of 0 (reasonable minimum)
)

type Config struct {
	Weight          float64
	HalfLife        time.Duration
	LatencyCuFactor float64 // should only be used for latency samples
}

var defaultConfig = Config{
	Weight:          DefaultWeight,
	HalfLife:        DefaultHalfLifeTime,
	LatencyCuFactor: DefaultCuLatencyFactor,
}

// Validate validates the Config's fields hold valid values
func (c Config) Validate() error {
	if c.Weight <= 0 {
		return fmt.Errorf("invalid config: weight must be strictly positive, weight: %f", c.Weight)
	}
	if c.HalfLife.Seconds() <= 0 {
		return fmt.Errorf("invalid config: half life time must be strictly positive, half life: %f", c.HalfLife.Seconds())
	}
	if c.LatencyCuFactor <= 0 || c.LatencyCuFactor > 1 {
		return fmt.Errorf("invalid config: latency cu factor must be between (0,1], latency cu factor: %f", c.LatencyCuFactor)
	}
	return nil
}

// String prints a Config's fields
func (c Config) String() string {
	return fmt.Sprintf("weight: %f, decay_half_life_time_sec: %f, latency_cu_factor: %f", c.Weight, c.HalfLife.Seconds(), c.LatencyCuFactor)
}

// Option is used as a generic and elegant way to configure a new ScoreStore
type Option func(*Config)

func WithWeight(weight float64) Option {
	return func(c *Config) {
		c.Weight = weight
	}
}

func WithDecayHalfLife(halfLife time.Duration) Option {
	return func(c *Config) {
		c.HalfLife = halfLife
	}
}

func WithLatencyCuFactor(factor float64) Option {
	return func(c *Config) {
		c.LatencyCuFactor = factor
	}
}

// GetLatencyFactor returns the appropriate latency factor by the CU amount
func GetLatencyFactor(cu uint64) float64 {
	if cu > HighCuThreshold {
		return HighCuLatencyFactor
	} else if cu < MidCuThreshold {
		return LowCuLatencyFactor
	}

	return MidCuLatencyFactor
}
