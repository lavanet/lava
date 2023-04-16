package provideroptimizer

import (
	"math"
	"math/rand"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/score"
)

const (
	CacheMaxCost               = 100  // each item cost would be 1
	CacheNumCounters           = 1000 // expect 100 items
	INITIAL_DATA_STALENESS     = 24
	HALF_LIFE_TIME             = time.Hour
	MAX_HALF_TIME              = 14 * 24 * time.Hour
	PROBE_UPDATE_WEIGHT        = 0.25
	RELAY_UPDATE_WEIGHT        = 1
	ASSUMED_VARIANCE           = 0.33
	DEFAULT_EXPLORATION_CHANCE = 0.1
)

type ProviderOptimizer struct {
	strategy                        Strategy
	providersStorage                *ristretto.Cache
	providerRelayStats              *ristretto.Cache
	averageBlockTime                time.Duration
	baseWorldLatency                time.Duration
	allowedBlockLagForQosSync       int64
	wantedNumProvidersInConcurrency int
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

// returns a sub set of selected providers according to their scores, perturbation factor will be added to each score in order to randomly select providers that are not always on top
func (po *ProviderOptimizer) ChooseProvider(allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64, perturbationPercentage float64) (addresses []string) {
	returnedProviders := make([]string, 1) // location 0 is always the best score
	latencyScore := math.MaxFloat64        // smaller = better i.e less latency
	syncScore := math.MaxFloat64           // smaller = better i.e less sync lag
	for _, providerAddress := range allAddresses {
		if _, ok := ignoredProviders[providerAddress]; ok {
			// ignored provider, skip it
			continue
		}
		providerData := po.getProviderData(providerAddress)

		// latency score
		latencyScoreCurrent := po.calculateLatencyScore(providerData, cu, requestedBlock) // smaller == better i.e less latency
		// latency perturbation
		latencyScoreCurrent = pertrubWithNormalGaussian(latencyScoreCurrent, perturbationPercentage)

		// sync score
		syncScoreCurrent := float64(0)
		if requestedBlock < 0 {
			// means user didn't ask for a specific block and we want to give him the best
			syncScoreCurrent = po.calculateSyncScore(providerData.Sync) // smaller == better i.e less sync lag
			// sync perturbation
			syncScoreCurrent = pertrubWithNormalGaussian(syncScoreCurrent, perturbationPercentage)
		}

		// we want the minimum latency and sync diff
		if po.isBetterProviderScore(latencyScore, latencyScoreCurrent, syncScore, syncScoreCurrent) || len(returnedProviders) == 0 {
			if len(returnedProviders) > 0 && po.shouldExplore(len(returnedProviders)) {
				// we are about to overwrite position 0, and this provider needs a chance to be in exploration
				returnedProviders = append(returnedProviders, returnedProviders[0])
			}
			returnedProviders[0] = providerAddress // best provider is always on position 0
			latencyScore = latencyScoreCurrent
			syncScore = syncScoreCurrent
			continue
		}
		if po.shouldExplore(len(returnedProviders)) {
			returnedProviders = append(returnedProviders, providerAddress)
		}
	}

	return returnedProviders
}

func (po *ProviderOptimizer) shouldExplore(currentNumProvders int) bool {
	if currentNumProvders > po.wantedNumProvidersInConcurrency {
		return false
	}
	explorationChance := DEFAULT_EXPLORATION_CHANCE
	switch po.strategy {
	case STRATEGY_LATENCY:
		return true // we want a lot of parallel tries on latency
	case STRATEGY_ACCURACY:
		return true
	case STRATEGY_COST:
		explorationChance = 0.01
	case STRATEGY_PRIVACY:
		return false // only one at a time
	}
	return rand.Float64() < explorationChance
}

func (po *ProviderOptimizer) isBetterProviderScore(latencyScore float64, latencyScoreCurrent float64, syncScore float64, syncScoreCurrent float64) bool {
	// TODO: change into score_latency^a * sync_score^b (do log for computation performance)
	var latencyWeight float64
	switch po.strategy {
	case STRATEGY_LATENCY:
		latencyWeight = 0.9
	case STRATEGY_SYNC_FRESHNESS:
		latencyWeight = 0.2
	case STRATEGY_PRIVACY:
		// pick at random regardless of score
		if rand.Intn(2) == 0 {
			return true
		}
	default:
		latencyWeight = 0.8
	}
	if syncScoreCurrent == 0 {
		return latencyScore > latencyScoreCurrent
	}
	return latencyScore*latencyWeight+syncScore*(1-latencyWeight) > latencyScoreCurrent*latencyWeight+syncScoreCurrent*(1-latencyWeight)
}

func (po *ProviderOptimizer) calculateSyncScore(SyncScore score.ScoreStore) float64 {
	// TODO: do the same as latency score
	return 1
}

func (po *ProviderOptimizer) calculateLatencyScore(providerData ProviderData, cu uint64, requestedBlock int64) float64 {
	baseLatency := po.baseWorldLatency + common.BaseTimePerCU(cu)/2 // divide by two because the returned time is for timeout not for average
	timeoutDuration := common.GetTimePerCu(cu)
	var historicalLatency time.Duration
	if providerData.Latency.Denom == 0 {
		historicalLatency = baseLatency
	} else {
		historicalLatency = baseLatency * time.Duration(providerData.Latency.Num/providerData.Latency.Denom)
	}

	probabilityBlockError := po.CalculateProbabilityOfBlockError(requestedBlock, providerData)
	probabilityOfTimeout := po.CalculateProbabilityOfTimeout(providerData.Availability)
	probabilityOfNoError := (1 - probabilityBlockError) * (1 - probabilityOfTimeout)

	costBlockError := historicalLatency.Seconds() + baseLatency.Seconds()
	costTimeout := timeoutDuration.Seconds() + baseLatency.Seconds()
	costSuccess := historicalLatency.Seconds()

	return probabilityBlockError*costBlockError + probabilityOfTimeout*costTimeout + probabilityOfNoError*costSuccess
}

func (po *ProviderOptimizer) CalculateProbabilityOfTimeout(availabilityScore score.ScoreStore) float64 {
	probabilityTimeout := float64(0)
	if availabilityScore.Denom > 0 { // shouldn't happen since we have default values but protect just in case
		mean := availabilityScore.Num / availabilityScore.Denom
		// bernoulli distribution assumption means probability of '1' is the mean, success is 1
		return 1 - mean
	}
	return probabilityTimeout
}

func (po *ProviderOptimizer) CalculateProbabilityOfBlockError(requestedBlock int64, providerData ProviderData) float64 {
	probabilityBlockError := float64(0)
	if requestedBlock > 0 && providerData.SyncBlock < uint64(requestedBlock) {
		// requested a specific block, so calculate a probability of provider having that block
		averageBlockTime := po.averageBlockTime.Seconds()
		blockDistanceRequired := uint64(requestedBlock) - providerData.SyncBlock
		repetitions := time.Since(providerData.Sync.Time).Seconds() / averageBlockTime
		probabilityBlockError = 1 - probValueAfterRepetitions(averageBlockTime, ASSUMED_VARIANCE, float64(blockDistanceRequired), repetitions) // we need greater than or equal not less than so complementary probability
	}
	return probabilityBlockError
}

func (po *ProviderOptimizer) AppendRelayData(providerAddress string, latency time.Duration, isHangingApi bool, success bool, cu uint64, syncBlock uint64, syncLag uint64) {
	providerData := po.getProviderData(providerAddress)
	halfTime := po.calculateHalfTime(providerAddress)
	providerData = po.updateProbeEntryAvailability(providerData, success, RELAY_UPDATE_WEIGHT, halfTime)
	if success {
		if latency > 0 {
			baseLatency := po.baseWorldLatency + common.BaseTimePerCU(cu)
			providerData = po.updateProbeEntryLatency(providerData, latency, baseLatency, RELAY_UPDATE_WEIGHT, halfTime)
		}
		if syncBlock > providerData.SyncBlock {
			// do not allow providers to go back
			providerData.SyncBlock = syncBlock
		}
		providerData = po.updateProbeEntrySync(providerData, syncLag, uint64(po.allowedBlockLagForQosSync), halfTime)
	}
	po.providersStorage.Set(providerAddress, providerData, 1)
	po.updateRelayTime(providerAddress)
}

func (po *ProviderOptimizer) AppendProbeRelayData(providerAddress string, latency time.Duration, success bool) {
	providerData := po.getProviderData(providerAddress)
	halfTime := po.calculateHalfTime(providerAddress)
	providerData = po.updateProbeEntryAvailability(providerData, success, PROBE_UPDATE_WEIGHT, halfTime)
	if success && latency > 0 {
		// base latency for a probe is the world latency
		providerData = po.updateProbeEntryLatency(providerData, latency, po.baseWorldLatency, PROBE_UPDATE_WEIGHT, halfTime)
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

func (po *ProviderOptimizer) updateProbeEntrySync(providerData ProviderData, sync uint64, baseSync uint64, halfTime time.Duration) ProviderData {
	newScore := score.NewScoreStore(float64(sync), float64(baseSync), time.Now())
	oldScore := providerData.Sync
	providerData.Sync = score.CalculateTimeDecayFunctionUpdate(oldScore, newScore, halfTime, RELAY_UPDATE_WEIGHT)
	return providerData
}

func (po *ProviderOptimizer) updateProbeEntryAvailability(providerData ProviderData, success bool, weight float64, halfTime time.Duration) ProviderData {
	newNumerator := float64(1)
	if !success {
		// if we failed we need the score update to be 0
		newNumerator = 0
	}
	oldScore := providerData.Availability
	newScore := score.NewScoreStore(newNumerator, 1, time.Now()) // denom is 1, entry time is now
	providerData.Availability = score.CalculateTimeDecayFunctionUpdate(oldScore, newScore, halfTime, weight)
	return providerData
}

// update latency data, base latency is the latency for the api defined in the spec
func (po *ProviderOptimizer) updateProbeEntryLatency(providerData ProviderData, latency time.Duration, baseLatency time.Duration, weight float64, halfTime time.Duration) ProviderData {
	newScore := score.NewScoreStore(latency.Seconds(), baseLatency.Seconds(), time.Now())
	oldScore := providerData.Latency
	providerData.Latency = score.CalculateTimeDecayFunctionUpdate(oldScore, newScore, halfTime, weight)
	return providerData
}

func (po *ProviderOptimizer) updateRelayTime(providerAddress string) {
	times := po.getRelayStatsTimes(providerAddress)
	if len(times) == 0 {
		po.providerRelayStats.Set(providerAddress, []time.Time{time.Now()}, 1)
		return
	}
	times = append(times, time.Now())
	po.providerRelayStats.Set(providerAddress, times, 1)
}

func (po *ProviderOptimizer) calculateHalfTime(providerAddress string) time.Duration {
	halfTime := HALF_LIFE_TIME
	relaysHalfTime := po.getRelayStatsTimeDiff(providerAddress)
	if relaysHalfTime > halfTime {
		halfTime = relaysHalfTime
	}
	if halfTime > MAX_HALF_TIME {
		halfTime = MAX_HALF_TIME
	}
	return halfTime
}

func (po *ProviderOptimizer) getRelayStatsTimeDiff(providerAddress string) time.Duration {
	times := po.getRelayStatsTimes(providerAddress)
	if len(times) == 0 {
		return 0
	}
	return time.Since(times[(len(times)-1)/2])
}

func (po *ProviderOptimizer) getRelayStatsTimes(providerAddress string) []time.Time {
	storedVal, found := po.providerRelayStats.Get(providerAddress)
	if found {
		times, ok := storedVal.([]time.Time)
		if !ok {
			utils.LavaFormatFatal("invalid usage of optimizer relay stats cache", nil, utils.Attribute{Key: "storedVal", Value: storedVal})
		}
		return times
	}
	return nil
}

func NewProviderOptimizer(strategy Strategy, allowedBlockLagForQosSync int64, averageBlockTIme time.Duration, baseWorldLatency time.Duration, wantedNumProvidersInConcurrency int) *ProviderOptimizer {
	cache, err := ristretto.NewCache(&ristretto.Config{NumCounters: CacheNumCounters, MaxCost: CacheMaxCost, BufferItems: 64})
	if err != nil {
		utils.LavaFormatFatal("failed setting up cache for queries", err)
	}
	relayCache, err := ristretto.NewCache(&ristretto.Config{NumCounters: CacheNumCounters, MaxCost: CacheMaxCost, BufferItems: 64})
	if err != nil {
		utils.LavaFormatFatal("failed setting up cache for queries", err)
	}
	if strategy == STRATEGY_PRIVACY {
		// overwrite
		wantedNumProvidersInConcurrency = 1
	}
	return &ProviderOptimizer{strategy: strategy, providersStorage: cache, averageBlockTime: averageBlockTIme, allowedBlockLagForQosSync: allowedBlockLagForQosSync, baseWorldLatency: baseWorldLatency, providerRelayStats: relayCache, wantedNumProvidersInConcurrency: wantedNumProvidersInConcurrency}
}

func zScore(mean float64, variance float64, value float64) float64 {
	stdDev := math.Sqrt(variance)
	return (value - mean) / stdDev
}

// calculate the probability a random variable with a given average and variance with Z repetitions will be less than or equal to value
func probValueAfterRepetitions(mean float64, variance float64, value float64, repetitions float64) float64 {
	// TODO: do a poisson probability calculation instead of a repetition of a normal distribution with a given mean and variance
	// Calculate the mean and variance of the sum of the random variables
	sumMean := mean * repetitions
	sumVariance := variance * repetitions

	// Calculate the integer and fractional parts of the repetitions
	intPart := math.Floor(repetitions)
	fracPart := repetitions - intPart

	// Calculate the mean and variance of the fractional part of the sum of the random variables
	fracSumMean := mean * fracPart
	fracSumVariance := variance * fracPart

	// Calculate the z-score of the fractional part of the sum of the random variables
	fracSumZ := zScore(fracSumMean, fracSumVariance, value)

	// Calculate the z-score of the integer part of the sum of the random variables
	intSumZ := zScore(sumMean, sumVariance, value)

	// Calculate the probability using the cumulative distribution function of the standard normal distribution
	prob := 1.0 - (1.0-math.Pow(0.5, intPart))*0.5*(1.0+math.Erf(intSumZ/math.Sqrt2)) - 0.5*(1.0+math.Erf(fracSumZ/math.Sqrt2))

	return prob
}

func pertrubWithNormalGaussian(orig float64, percentage float64) float64 {
	return orig + rand.NormFloat64()*percentage*orig
}
