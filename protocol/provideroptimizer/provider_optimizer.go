package provideroptimizer

import "time"

type ProviderOptimizer struct {
	strategy Strategy
}

type Strategy int

const (
	STRATEGY_QOS Strategy = iota
	STRATEGY_COST
	STRATEGY_PRIVACY
	STRATEGY_ACCURACY
)

func (po *ProviderOptimizer) AppendRelayData(providerAddress string, latency time.Duration, failure bool) {
}

func NewProviderOptimizer(strategy Strategy) *ProviderOptimizer {
	return &ProviderOptimizer{strategy: strategy}
}
