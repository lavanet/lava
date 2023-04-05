package provideroptimizer

import (
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/score"
)

const (
	CacheMaxCost           = 100  // each item cost would be 1
	CacheNumCounters       = 1000 // expect 100 items
	INITIAL_DATA_STALENESS = 24
	HALF_LIFE_TIME         = time.Hour
	PROBE_UPDATE_WEIGHT    = 0.25
	RELAY_UPDATE_WEIGHT    = 1
)

type ProviderOptimizer struct {
	strategy                  Strategy
	providersStorage          *ristretto.Cache
	averageBlockTime          time.Duration
	baseWorldLatency          time.Duration
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
	STRATEGY_LATENCY
	STRATEGY_SYNC_FRESHNESS
	STRATEGY_COST
	STRATEGY_PRIVACY
	STRATEGY_ACCURACY
)

func (po *ProviderOptimizer) AppendRelayData(providerAddress string, latency time.Duration, isHangingApi bool, success bool, cu uint64, syncBlock uint64, sync uint64) {
	providerData := po.getProviderData(providerAddress)
	providerData = po.updateProbeEntryAvailability(providerData, success, RELAY_UPDATE_WEIGHT)
	if success {
		if latency > 0 {
			baseLatency := po.baseWorldLatency + common.BaseTimePerCU(cu)
			if isHangingApi {
				baseLatency += po.averageBlockTime
			}
			providerData = po.updateProbeEntryLatency(providerData, latency, baseLatency, RELAY_UPDATE_WEIGHT)
		}
		if syncBlock > providerData.SyncBlock {
			// do not allow providers to go back
			providerData.SyncBlock = syncBlock
		}
		providerData = po.updateProbeEntrySync(providerData, sync, uint64(po.allowedBlockLagForQosSync))
	}
	po.providersStorage.Set(providerAddress, providerData, 1)
}

func (po *ProviderOptimizer) AppendProbeRelayData(providerAddress string, latency time.Duration, success bool) {
	providerData := po.getProviderData(providerAddress)
	providerData = po.updateProbeEntryAvailability(providerData, success, PROBE_UPDATE_WEIGHT)
	if success && latency > 0 {
		// base latency for a probe is the world latency
		providerData = po.updateProbeEntryLatency(providerData, latency, po.baseWorldLatency, PROBE_UPDATE_WEIGHT)
	}
	po.providersStorage.Set(providerAddress, providerData, 1)
}

func (po *ProviderOptimizer) getProviderData(providerAddress string) ProviderData {
	var providerData ProviderData

	storedVal, found := po.providersStorage.Get(providerAddress)
	if found {
		var ok bool

		providerData, ok = storedVal.(ProviderData)
		if !ok {
			utils.LavaFormatFatal("invalid usage of optimizer provider storage", nil, utils.Attribute{Key: "storedVal", Value: storedVal})
		}
	} else {

		providerData = ProviderData{
			Availability: score.NewScoreStore(1, 2, time.Now().Add(-1*INITIAL_DATA_STALENESS*time.Hour)), // default value of half score
			Latency:      score.NewScoreStore(2, 1, time.Now().Add(-1*INITIAL_DATA_STALENESS*time.Hour)), // default value of half score (twice the time)
			Sync:         score.NewScoreStore(2, 1, time.Now().Add(-1*INITIAL_DATA_STALENESS*time.Hour)), // default value of half score (twice the diff)
			SyncBlock:    0,
		}
	}
	return providerData
}

func (po *ProviderOptimizer) updateProbeEntrySync(providerData ProviderData, sync uint64, baseSync uint64) ProviderData {
	newScore := score.NewScoreStore(float64(sync), float64(baseSync), time.Now())
	oldScore := providerData.Sync
	providerData.Sync = score.CalculateTimeDecayFunctionUpdate(oldScore, newScore, HALF_LIFE_TIME, RELAY_UPDATE_WEIGHT)
	return providerData
}

func (po *ProviderOptimizer) updateProbeEntryAvailability(providerData ProviderData, success bool, weight float64) ProviderData {
	newNumerator := float64(1)
	if !success {
		// if we failed we need the score update to be 0
		newNumerator = 0
	}
	oldScore := providerData.Availability
	newScore := score.NewScoreStore(newNumerator, 1, time.Now()) //denom is 1, entry time is now
	providerData.Availability = score.CalculateTimeDecayFunctionUpdate(oldScore, newScore, HALF_LIFE_TIME, weight)
	return providerData
}

// update latency data, base latency is the latency for the api defined in the spec
func (po *ProviderOptimizer) updateProbeEntryLatency(providerData ProviderData, latency time.Duration, baseLatency time.Duration, weight float64) ProviderData {
	newScore := score.NewScoreStore(latency.Seconds(), baseLatency.Seconds(), time.Now())
	oldScore := providerData.Latency
	providerData.Latency = score.CalculateTimeDecayFunctionUpdate(oldScore, newScore, HALF_LIFE_TIME, weight)
	return providerData
}

func NewProviderOptimizer(strategy Strategy, allowedBlockLagForQosSync int64, averageBlockTIme time.Duration, baseWorldLatency time.Duration) *ProviderOptimizer {
	cache, err := ristretto.NewCache(&ristretto.Config{NumCounters: CacheNumCounters, MaxCost: CacheMaxCost, BufferItems: 64})
	if err != nil {
		utils.LavaFormatFatal("failed setting up cache for queries", err)
	}
	return &ProviderOptimizer{strategy: strategy, providersStorage: cache, averageBlockTime: averageBlockTIme, allowedBlockLagForQosSync: allowedBlockLagForQosSync, baseWorldLatency: baseWorldLatency}
}
