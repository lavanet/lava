package types

import (
	"fmt"
	"math"

	cosmosmath "cosmossdk.io/math"
)

// NewQualityOfServiceReport creates a QualityOfServiceReport with default zero values.
func NewQualityOfServiceReport() *QualityOfServiceReport {
	return &QualityOfServiceReport{
		Latency:      0,
		Availability: 0,
		Sync:         0,
	}
}

// QoS reputation computation configuration.
var (
	DefaultFailureCost   int64   = 3
	DefaultSyncFactor    float64 = 0.3
	DefaultStrategyFactor float64 = 1.0 // balanced

	// strategy factors (multipliers to the sync factor)
	BalancedStrategyFactor      float64 = 1.0
	LatencyStrategyFactor       float64 = 1.0 / 3.0
	SyncFreshnessStrategyFactor float64 = 30.0
)

// reputationConfig holds parameters for ComputeReputation.
type reputationConfig struct {
	syncFactor            float64
	failureCost           int64
	strategyFactor        float64
	blockErrorProbability float64 // -1 means unused (default)
}

func defaultReputationConfig() reputationConfig {
	return reputationConfig{
		syncFactor:            DefaultSyncFactor,
		failureCost:           DefaultFailureCost,
		strategyFactor:        DefaultStrategyFactor,
		blockErrorProbability: -1,
	}
}

// Option configures a reputationConfig.
type Option func(*reputationConfig)

func WithSyncFactor(factor cosmosmath.LegacyDec) Option {
	return func(c *reputationConfig) {
		f, _ := factor.Float64()
		c.syncFactor = f
	}
}

func WithFailureCost(cost int64) Option {
	return func(c *reputationConfig) {
		c.failureCost = cost
	}
}

func WithStrategyFactor(factor cosmosmath.LegacyDec) Option {
	return func(c *reputationConfig) {
		f, _ := factor.Float64()
		c.strategyFactor = f
	}
}

func WithBlockErrorProbability(probability cosmosmath.LegacyDec) Option {
	return func(c *reputationConfig) {
		f, _ := probability.Float64()
		c.blockErrorProbability = f
	}
}

// Validate returns an error if the QualityOfServiceReport fields are invalid.
func (q *QualityOfServiceReport) Validate() error {
	if q == nil {
		return fmt.Errorf("nil QualityOfServiceReport")
	}
	if q.Latency < 0 {
		return fmt.Errorf("invalid QoS latency, latency is negative: %v", q.Latency)
	}
	if q.Sync < 0 {
		return fmt.Errorf("invalid QoS sync, sync is negative: %v", q.Sync)
	}
	if q.Availability <= 0 {
		return fmt.Errorf("invalid QoS availability, availability is non-positive: %v", q.Availability)
	}
	return nil
}

// ComputeReputation calculates a composite reputation score from the QoS report.
// Formula (for latest-block / not-applicable requests):
//
//	score = latency + sync*syncFactor*strategyFactor + ((1/availability) - 1) * failureCost
//
// Returns a math.LegacyDec for backward compatibility with callers.
func (q *QualityOfServiceReport) ComputeReputation(opts ...Option) (cosmosmath.LegacyDec, error) {
	if err := q.Validate(); err != nil {
		return cosmosmath.LegacyZeroDec(), err
	}

	cfg := defaultReputationConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	latency := q.Latency
	var syncComponent float64
	if cfg.blockErrorProbability >= 0 {
		// use block-error-probability path
		syncComponent = cfg.blockErrorProbability * float64(cfg.failureCost)
	} else {
		syncComponent = q.Sync * cfg.syncFactor * cfg.strategyFactor
	}
	availabilityComponent := ((1.0 / q.Availability) - 1.0) * float64(cfg.failureCost)

	total := latency + syncComponent + availabilityComponent

	// Convert to LegacyDec: multiply by 10^18 precision then use NewDecWithPrec
	const precision = 18
	scaled := int64(math.Round(total * math.Pow(10, float64(precision))))
	return cosmosmath.LegacyNewDecWithPrec(scaled, precision), nil
}

// ComputeReputationFloat64 is a convenience wrapper returning float64.
func (q *QualityOfServiceReport) ComputeReputationFloat64(opts ...Option) (float64, error) {
	dec, err := q.ComputeReputation(opts...)
	if err != nil {
		return 0, err
	}
	f, err := dec.Float64()
	if err != nil {
		return 0, err
	}
	return f, nil
}

// ComputeQoS computes the geometric mean of availability, latency, and sync
// scores (all expected in [0,1]). Returns an error if any score is out of range.
func (q *QualityOfServiceReport) ComputeQoS() (cosmosmath.LegacyDec, error) {
	if q.Availability > 1 || q.Availability < 0 ||
		q.Latency > 1 || q.Latency < 0 ||
		q.Sync > 1 || q.Sync < 0 {
		return cosmosmath.LegacyZeroDec(), fmt.Errorf("QoS scores is not between 0-1")
	}

	result := math.Cbrt(q.Availability * q.Sync * q.Latency)
	const precision = 18
	scaled := int64(math.Round(result * math.Pow(10, float64(precision))))
	return cosmosmath.LegacyNewDecWithPrec(scaled, precision), nil
}
