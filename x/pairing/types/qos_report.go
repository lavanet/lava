package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/score"
)

// QoS (quality of service) is a report that consists three metrics that are
// used to measure providers performance. The metrics are:
// 	1. Latency: the time it takes the provider to answer to consumer relays.
//
//  2. Sync: the latest block that the provider percieves is close to the actual
//           last block of the chain.
//
//  3. Availability: the provider's up time.

var (
	DefaultFailureCost           int64 = 3
	DefaultSyncFactor                  = sdk.NewDecWithPrec(3, 1) // 0.3
	DefaultStrategyFactor              = BalancedStrategyFactor
	DefaultBlockErrorProbability       = sdk.NewDec(-1) // default: BlockErrorProbability should not be used

	// strategy factors (used as multipliers to the sync factor)
	// 1. balanced strategy: multiply the sync factor by 1 -> staying with default sync factor
	// 2. latency strategy: make latency more influential -> divide the default sync factor by 3
	// 3. sync freshness strategy: make sync more influential -> multiply the default sync factor by 3
	BalancedStrategyFactor      = sdk.OneDec()             // 1
	LatencyStrategyFactor       = sdk.OneDec().QuoInt64(3) // 1/3
	SyncFreshnessStrategyFactor = sdk.NewDec(30)           // 3
)

// Config defines a collection of parameters that can be used when calculating
// a QoS excellence report score
type Config struct {
	SyncFactor            sdk.Dec // a fractional factor to diminish the sync score influence compared to the latency score
	FailureCost           int64   // the cost (in seconds) for a provider failing to service a relay
	StrategyFactor        sdk.Dec // a factor to further configure the sync factor
	BlockErrorProbability sdk.Dec // a probability that a provider doesn't have the requested block the optimizer needs (used for non-latest QoS scores)
}

// Validate validates the Config's fields hold valid values
func (c Config) Validate() error {
	if c.SyncFactor.IsNegative() || c.SyncFactor.GT(sdk.OneDec()) {
		return fmt.Errorf("invalid config: sync factor must be between 0-1, sync factor: %s", c.SyncFactor.String())
	}
	if c.FailureCost < 0 {
		return fmt.Errorf("invalid config: failure cost cannot be negative, failure cost: %d", c.FailureCost)
	}
	if c.StrategyFactor.IsNegative() {
		return fmt.Errorf("invalid config: strategy factor cannot be negative, failure cost: %s", c.StrategyFactor.String())
	}
	if !c.BlockErrorProbability.Equal(DefaultBlockErrorProbability) && (c.BlockErrorProbability.IsNegative() || c.BlockErrorProbability.GT(sdk.OneDec())) {
		return fmt.Errorf("invalid config: block error probability must be default unused (-1) or between 0-1, probability: %s", c.BlockErrorProbability.String())
	}

	return nil
}

// String prints a Config's fields
func (c Config) String() string {
	return fmt.Sprintf("sync factor: %s, failure cost sec: %d, strategy factor: %s, block error probability: %s",
		c.SyncFactor.String(), c.FailureCost, c.StrategyFactor.String(), c.BlockErrorProbability.String())
}

// Default configuration
var DefaultConfig = Config{
	SyncFactor:            DefaultSyncFactor,
	FailureCost:           DefaultFailureCost,
	StrategyFactor:        DefaultStrategyFactor,
	BlockErrorProbability: DefaultBlockErrorProbability,
}

// Option is used as a generic and elegant way to configure a new ScoreStore
type Option func(*Config)

func WithSyncFactor(factor sdk.Dec) Option {
	return func(c *Config) {
		c.SyncFactor = factor
	}
}

func WithFailureCost(cost int64) Option {
	return func(c *Config) {
		c.FailureCost = cost
	}
}

func WithStrategyFactor(factor sdk.Dec) Option {
	return func(c *Config) {
		c.StrategyFactor = factor
	}
}

func WithBlockErrorProbability(probability sdk.Dec) Option {
	return func(c *Config) {
		c.BlockErrorProbability = probability
	}
}

// ComputeReputation calculates a score from the QoS excellence report by the following formula:
// If the requested block is the latest block or "not applicable" (called from the node's code):
//
//	score = latency + sync*syncFactor + ((1/availability) - 1) * FailureCost
//
// note, the syncFactor is multiplied by the strategy factor
//
// for every other request:
//
//	score = latency + blockErrorProbability * FailureCost + ((1/availability) - 1) * FailureCost
//
// Important: when using this function from the node's code, do not configure the block error probability
// (in default mode, it's unused)
// TODO: after the reputation feature is merged, use this method to calculate the QoS excellence score
func (qos *QualityOfServiceReport) ComputeReputation(opts ...Option) (sdk.Dec, error) {
	if err := qos.Validate(); err != nil {
		return sdk.ZeroDec(), err
	}

	cfg := DefaultConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	if err := cfg.Validate(); err != nil {
		return sdk.ZeroDec(), err
	}

	latency := qos.Latency
	sync := qos.Sync.Mul(cfg.SyncFactor).Mul(cfg.StrategyFactor)
	if !cfg.BlockErrorProbability.Equal(DefaultBlockErrorProbability) {
		// BlockErrorProbability is not default, calculate sync using it (already validated above in cfg.Validate())
		sync = cfg.BlockErrorProbability.MulInt64(cfg.FailureCost)
	}
	availability := ((sdk.OneDec().Quo(qos.Availability)).Sub(sdk.OneDec())).MulInt64(cfg.FailureCost)

	return latency.Add(sync).Add(availability), nil
}

func (qos *QualityOfServiceReport) ComputeReputationFloat64(opts ...Option) (float64, error) {
	scoreDec, err := qos.ComputeReputation(opts...)
	if err != nil {
		return 0, err
	}
	score, err := scoreDec.Float64()
	if err != nil {
		return 0, err
	}
	return score, nil
}

func (qos *QualityOfServiceReport) Validate() error {
	if qos.Latency.IsNegative() {
		return fmt.Errorf("invalid QoS latency, latency is negative: %s", qos.Latency.String())
	}
	if qos.Sync.IsNegative() {
		return fmt.Errorf("invalid QoS sync, sync is negative: %s", qos.Sync.String())
	}
	if qos.Availability.IsNegative() || qos.Availability.IsZero() {
		return fmt.Errorf("invalid QoS availability, availability is non-positive: %s", qos.Availability.String())
	}

	return nil
}

func (qos *QualityOfServiceReport) ComputeQoS() (sdk.Dec, error) {
	if qos.Availability.GT(sdk.OneDec()) || qos.Availability.LT(sdk.ZeroDec()) ||
		qos.Latency.GT(sdk.OneDec()) || qos.Latency.LT(sdk.ZeroDec()) ||
		qos.Sync.GT(sdk.OneDec()) || qos.Sync.LT(sdk.ZeroDec()) {
		return sdk.ZeroDec(), fmt.Errorf("QoS scores is not between 0-1")
	}

	return qos.Availability.Mul(qos.Sync).Mul(qos.Latency).ApproxRoot(3)
}

func (qos *QualityOfServiceReport) GetScoresFloat64() (float64, float64, float64) {
	latency, err := qos.Latency.Float64()
	if err != nil {
		utils.LavaFormatError("critical: failed to convert latency score to float64", err, utils.LogAttr("latency", qos.Latency.String()))
		latency = score.WorstLatencyScore
	}
	sync, err := qos.Sync.Float64()
	if err != nil {
		utils.LavaFormatError("critical: failed to convert sync score to float64", err, utils.LogAttr("sync", qos.Sync.String()))
		sync = score.WorstSyncScore
	}
	availability, err := qos.Availability.Float64()
	if err != nil {
		utils.LavaFormatError("critical: failed to convert availability score to float64", err, utils.LogAttr("availability", qos.Availability.String()))
		availability = score.WorstAvailabilityScore
	}

	return latency, sync, availability
}
