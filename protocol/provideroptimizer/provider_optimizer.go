package provideroptimizer

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dgraph-io/ristretto"
	"github.com/lavanet/lava/utils"
)

const (
	CacheMaxCost     = 100  // each item cost would be 1
	CacheNumCounters = 1000 // expect 100 items
)

type ProviderOptimizer struct {
	strategy         Strategy
	providersStorage *ristretto.Cache
	averageBlockTime time.Duration
}

type NumDenomStore struct {
	Num   sdk.Dec
	Denom sdk.Dec
}

type ProviderData struct {
	Availability NumDenomStore
	Latency      NumDenomStore
	SyncNum      NumDenomStore

	UpdateTime time.Time
}

type Strategy int

const (
	STRATEGY_BALANCED Strategy = iota
	STRATEGY_QOS
	STRATEGY_COST
	STRATEGY_PRIVACY
	STRATEGY_ACCURACY
)

func (po *ProviderOptimizer) AppendRelayData(providerAddress string, latency time.Duration, failure bool) {
	// po.providersStorage.
	// .Get(PairingRespKey + chainID)
	// .SetWithTTL(PairingRespKey+chainID, pairingResp, 1, DefaultTimeToLiveExpiration)
}

func NewProviderOptimizer(strategy Strategy, averageBlockTIme time.Duration) *ProviderOptimizer {
	cache, err := ristretto.NewCache(&ristretto.Config{NumCounters: CacheNumCounters, MaxCost: CacheMaxCost, BufferItems: 64})
	if err != nil {
		utils.LavaFormatFatal("failed setting up cache for queries", err)
	}
	return &ProviderOptimizer{strategy: strategy, providersStorage: cache, averageBlockTime: averageBlockTIme}
}
