package provideroptimizer

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dgraph-io/ristretto"
	"github.com/lavanet/lava/utils"
)

const (
	CacheMaxCost           = 100  // each item cost would be 1
	CacheNumCounters       = 1000 // expect 100 items
	INITIAL_DATA_STALENESS = 24
)

type ProviderOptimizer struct {
	strategy                  Strategy
	providersStorage          *ristretto.Cache
	averageBlockTime          time.Duration
	allowedBlockLagForQosSync int64
}

type ScoreStore struct {
	Num   sdk.Dec
	Denom sdk.Dec
	Time  time.Time
}

type ProviderData struct {
	Availability ScoreStore // will be used to calculate the probability of error
	Latency      ScoreStore // will be used to calculate the latency score
	Sync         ScoreStore // will be used to calculate the sync score for spectypes.LATEST_BLOCK/spectypes.NOT_APPLICABLE requests
	SyncBlock    uint64     // will be used to calculate the probability of block error
}

type Strategy int

const (
	STRATEGY_BALANCED Strategy = iota
	STRATEGY_QOS
	STRATEGY_COST
	STRATEGY_PRIVACY
	STRATEGY_ACCURACY
)

func (po *ProviderOptimizer) AppendProbeRelayData(providerAddress string, latency time.Duration, failure bool) {
	var providerData ProviderData

	storedVal, found := po.providersStorage.Get(providerAddress)
	if found {
		var ok bool
		// first instance of the data
		providerData, ok = storedVal.(ProviderData)
		if !ok {
			utils.LavaFormatFatal("invalid usage of optimizer provider storage", nil, utils.Attribute{Key: "storedVal", Value: storedVal})
		}
	} else {
		// we start with defaults of 1 with very old data
		providerData = ProviderData{
			Availability: ScoreStore{Num: sdk.OneDec(), Denom: sdk.OneDec(), Time: time.Now().Add(-1 * INITIAL_DATA_STALENESS * time.Hour)},
			Latency:      ScoreStore{Num: sdk.OneDec(), Denom: sdk.OneDec(), Time: time.Now().Add(-1 * INITIAL_DATA_STALENESS * time.Hour)},
			Sync:         ScoreStore{Num: sdk.OneDec(), Denom: sdk.OneDec(), Time: time.Now().Add(-1 * INITIAL_DATA_STALENESS * time.Hour)},
			SyncBlock:    0,
		}
	}
	if failure {
		providerData = po.updateProbeEntryFailure(providerData)
	} else {
		providerData = po.updateProbeEntry(providerData, latency)
	}
	po.providersStorage.Set(providerAddress, providerData, 1)
}

func (po *ProviderOptimizer) updateProbeEntryFailure(providerData ProviderData) ProviderData {
	lastUpdateTime := providerData.Availability.Time
	numerator := providerData.Availability.Num
	denominator := providerData.Availability.Denom
	numerator, denominator = CalculateDecayFunctionUpdate(numerator, denominator, lastUpdateTime)
	providerData.Availability.Num = numerator
	providerData.Availability.Denom = denominator
	providerData.Availability.Time = time.Now()
	return providerData // TODO
}

func CalculateDecayFunctionUpdate(numerator_old sdk.Dec, denom_old sdk.Dec, old_update_time time.Time) (updated_num sdk.Dec, updated_denom sdk.Dec) {
	return numerator_old, denom_old // TODO
}

func (po *ProviderOptimizer) updateProbeEntry(providerData ProviderData, latency time.Duration) ProviderData {
	return providerData // TODO
}

func NewProviderOptimizer(strategy Strategy, allowedBlockLagForQosSync int64, averageBlockTIme time.Duration) *ProviderOptimizer {
	cache, err := ristretto.NewCache(&ristretto.Config{NumCounters: CacheNumCounters, MaxCost: CacheMaxCost, BufferItems: 64})
	if err != nil {
		utils.LavaFormatFatal("failed setting up cache for queries", err)
	}
	return &ProviderOptimizer{strategy: strategy, providersStorage: cache, averageBlockTime: averageBlockTIme, allowedBlockLagForQosSync: allowedBlockLagForQosSync}
}
