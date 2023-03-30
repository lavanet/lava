package provideroptimizer

import (
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/score"
)

const (
	CacheMaxCost           = 100  // each item cost would be 1
	CacheNumCounters       = 1000 // expect 100 items
	INITIAL_DATA_STALENESS = 24
	HALF_LIFE_TIME         = time.Hour
	PROBE_UPDATE_WEIGHT    = 0.2
)

type ProviderOptimizer struct {
	strategy                  Strategy
	providersStorage          *ristretto.Cache
	averageBlockTime          time.Duration
	allowedBlockLagForQosSync int64
}

type ProviderData struct {
	Availability score.ScoreStore // will be used to calculate the probability of error
	Latency      score.ScoreStore // will be used to calculate the latency score
	Sync         score.ScoreStore // will be used to calculate the sync score for spectypes.LATEST_BLOCK/spectypes.NOT_APPLICABLE requests
	SyncBlock    uint64           // will be used to calculate the probability of block error
}

type Strategy int

const (
	STRATEGY_BALANCED Strategy = iota
	STRATEGY_QOS
	STRATEGY_COST
	STRATEGY_PRIVACY
	STRATEGY_ACCURACY
)

func (po *ProviderOptimizer) AppendProbeRelayData(providerAddress string, latency time.Duration, success bool) {
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
			Availability: score.NewScoreStore(1, 1, time.Now().Add(-1*INITIAL_DATA_STALENESS*time.Hour)),
			Latency:      score.NewScoreStore(1, 1, time.Now().Add(-1*INITIAL_DATA_STALENESS*time.Hour)),
			Sync:         score.NewScoreStore(1, 1, time.Now().Add(-1*INITIAL_DATA_STALENESS*time.Hour)),
			SyncBlock:    0,
		}
	}

	providerData = po.updateProbeEntryAvailability(providerData, success)
	if success {
		providerData = po.updateProbeEntry(providerData, latency)
	}
	po.providersStorage.Set(providerAddress, providerData, 1)
}

func (po *ProviderOptimizer) updateProbeEntryAvailability(providerData ProviderData, success bool) ProviderData {
	newNumerator := float64(1)
	if !success {
		// if we failed we need the score update to be 0
		newNumerator = 0
	}
	oldScore := providerData.Availability
	newScore := score.NewScoreStore(newNumerator, 1, time.Now()) //denom is 1, entry time is now
	providerData.Availability = score.CalculateTimeDecayFunctionUpdate(oldScore, newScore, HALF_LIFE_TIME, PROBE_UPDATE_WEIGHT)
	return providerData
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
