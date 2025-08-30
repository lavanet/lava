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
