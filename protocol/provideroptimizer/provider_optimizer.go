package provideroptimizer

import (
	"math"
	"strings"
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dgraph-io/ristretto"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/lavaslices"
	"github.com/lavanet/lava/utils/rand"
	"github.com/lavanet/lava/utils/score"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"gonum.org/v1/gonum/mathext"
)

const (
	CacheMaxCost               = 2000  // each item cost would be 1
	CacheNumCounters           = 20000 // expect 2000 items
	INITIAL_DATA_STALENESS     = 24
	HALF_LIFE_TIME             = time.Hour
	MAX_HALF_TIME              = 3 * time.Hour
	PROBE_UPDATE_WEIGHT        = 0.25
	RELAY_UPDATE_WEIGHT        = 1
	DEFAULT_EXPLORATION_CHANCE = 0.1
	COST_EXPLORATION_CHANCE    = 0.01
	WANTED_PRECISION           = int64(8)
)

type ConcurrentBlockStore struct {
	Lock  sync.Mutex
	Time  time.Time
	Block uint64
}

type cacheInf interface {
	Get(key interface{}) (interface{}, bool)
	Set(key, value interface{}, cost int64) bool
}

type ProviderOptimizer struct {
	strategy                        Strategy
	providersStorage                cacheInf
	providerRelayStats              *ristretto.Cache // used to decide on the half time of the decay
	averageBlockTime                time.Duration
	baseWorldLatency                time.Duration
	wantedNumProvidersInConcurrency uint
	latestSyncData                  ConcurrentBlockStore
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
	STRATEGY_DISTRIBUTED
)

func (po *ProviderOptimizer) AppendRelayFailure(providerAddress string) {
	po.appendRelayData(providerAddress, 0, false, false, 0, 0, time.Now())
}

func (po *ProviderOptimizer) AppendRelayData(providerAddress string, latency time.Duration, isHangingApi bool, cu, syncBlock uint64) {
	po.appendRelayData(providerAddress, latency, isHangingApi, true, cu, syncBlock, time.Now())
}

func (po *ProviderOptimizer) appendRelayData(providerAddress string, latency time.Duration, isHangingApi, success bool, cu, syncBlock uint64, sampleTime time.Time) {
	latestSync, timeSync := po.updateLatestSyncData(syncBlock, sampleTime)
	providerData, _ := po.getProviderData(providerAddress)
	halfTime := po.calculateHalfTime(providerAddress, sampleTime)
	providerData = po.updateProbeEntryAvailability(providerData, success, RELAY_UPDATE_WEIGHT, halfTime, sampleTime)
	if success {
		if latency > 0 {
			baseLatency := po.baseWorldLatency + common.BaseTimePerCU(cu)/2
			if isHangingApi {
				baseLatency += po.averageBlockTime / 2 // hanging apis take longer
			}
			providerData = po.updateProbeEntryLatency(providerData, latency, baseLatency, RELAY_UPDATE_WEIGHT, halfTime, sampleTime)
		}
		if syncBlock > providerData.SyncBlock {
			// do not allow providers to go back
			providerData.SyncBlock = syncBlock
		}
		syncLag := po.calculateSyncLag(latestSync, timeSync, providerData.SyncBlock, sampleTime)
		providerData = po.updateProbeEntrySync(providerData, syncLag, po.averageBlockTime, halfTime, sampleTime)
	}
	po.providersStorage.Set(providerAddress, providerData, 1)
	po.updateRelayTime(providerAddress, sampleTime)

	utils.LavaFormatTrace("relay update",
		utils.LogAttr("providerData", providerData),
		utils.LogAttr("syncBlock", syncBlock),
		utils.LogAttr("cu", cu),
		utils.LogAttr("providerAddress", providerAddress),
		utils.LogAttr("latency", latency),
		utils.LogAttr("success", success),
	)
}

func (po *ProviderOptimizer) AppendProbeRelayData(providerAddress string, latency time.Duration, success bool) {
	providerData, _ := po.getProviderData(providerAddress)
	sampleTime := time.Now()
	halfTime := po.calculateHalfTime(providerAddress, sampleTime)
	providerData = po.updateProbeEntryAvailability(providerData, success, PROBE_UPDATE_WEIGHT, halfTime, sampleTime)
	if success && latency > 0 {
		// base latency for a probe is the world latency
		providerData = po.updateProbeEntryLatency(providerData, latency, po.baseWorldLatency, PROBE_UPDATE_WEIGHT, halfTime, sampleTime)
	}
	po.providersStorage.Set(providerAddress, providerData, 1)

	utils.LavaFormatTrace("probe update",
		utils.LogAttr("providerAddress", providerAddress),
		utils.LogAttr("latency", latency),
		utils.LogAttr("success", success),
	)
}

// returns a sub set of selected providers according to their scores, perturbation factor will be added to each score in order to randomly select providers that are not always on top
func (po *ProviderOptimizer) ChooseProvider(allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64, perturbationPercentage float64) (addresses []string) {
	returnedProviders := make([]string, 1) // location 0 is always the best score
	latencyScore := math.MaxFloat64        // smaller = better i.e less latency
	syncScore := math.MaxFloat64           // smaller = better i.e less sync lag
	numProviders := len(allAddresses)
	if po.strategy == STRATEGY_DISTRIBUTED {
		// distribute relays across more providers
		perturbationPercentage *= 2
	}
	for _, providerAddress := range allAddresses {
		if _, ok := ignoredProviders[providerAddress]; ok {
			// ignored provider, skip it
			continue
		}
		providerData, found := po.getProviderData(providerAddress)
		if !found {
			utils.LavaFormatTrace("provider data was not found for address", utils.LogAttr("providerAddress", providerAddress))
		}
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

		utils.LavaFormatTrace("scores information",
			utils.LogAttr("providerAddress", providerAddress),
			utils.LogAttr("latencyScoreCurrent", latencyScoreCurrent),
			utils.LogAttr("syncScoreCurrent", syncScoreCurrent),
			utils.LogAttr("latencyScore", latencyScore),
			utils.LogAttr("syncScore", syncScore),
		)

		// we want the minimum latency and sync diff
		if po.isBetterProviderScore(latencyScore, latencyScoreCurrent, syncScore, syncScoreCurrent) || len(returnedProviders) == 0 {
			if returnedProviders[0] != "" && po.shouldExplore(len(returnedProviders), numProviders) {
				// we are about to overwrite position 0, and this provider needs a chance to be in exploration
				returnedProviders = append(returnedProviders, returnedProviders[0])
			}
			returnedProviders[0] = providerAddress // best provider is always on position 0
			latencyScore = latencyScoreCurrent
			syncScore = syncScoreCurrent
			continue
		}
		if po.shouldExplore(len(returnedProviders), numProviders) {
			returnedProviders = append(returnedProviders, providerAddress)
		}
	}

	utils.LavaFormatTrace("returned providers",
		utils.LogAttr("providers", strings.Join(returnedProviders, ",")),
		utils.LogAttr("cu", cu),
	)

	return returnedProviders
}

// calculate the expected average time until this provider catches up with the given latestSync block
// for the first block difference we take the minimum between the time passed since block arrived and the average block time
// for any other block we take the averageBlockTime
func (po *ProviderOptimizer) calculateSyncLag(latestSync uint64, timeSync time.Time, providerBlock uint64, sampleTime time.Time) time.Duration {
	// check gap is >=1
	if latestSync <= providerBlock {
		return 0
	}
	// lag on first block
	timeLag := sampleTime.Sub(timeSync) // received the latest block at time X, this provider provided the entry at time Y, which is X-Y time after
	firstBlockLag := lavaslices.Min([]time.Duration{po.averageBlockTime, timeLag})
	blocksGap := latestSync - providerBlock - 1                     // latestSync > providerBlock
	blocksGapTime := time.Duration(blocksGap) * po.averageBlockTime // the provider is behind by X blocks, so is expected to catch up in averageBlockTime * X
	timeLag = firstBlockLag + blocksGapTime
	return timeLag
}

func (po *ProviderOptimizer) updateLatestSyncData(providerLatestBlock uint64, sampleTime time.Time) (uint64, time.Time) {
	po.latestSyncData.Lock.Lock()
	defer po.latestSyncData.Lock.Unlock()
	latestBlock := po.latestSyncData.Block
	if latestBlock < providerLatestBlock {
		// saved latest block is older, so update
		po.latestSyncData.Block = providerLatestBlock
		po.latestSyncData.Time = sampleTime
	}
	return po.latestSyncData.Block, po.latestSyncData.Time
}

func (po *ProviderOptimizer) shouldExplore(currentNumProvders, numProviders int) bool {
	if uint(currentNumProvders) >= po.wantedNumProvidersInConcurrency {
		return false
	}
	explorationChance := DEFAULT_EXPLORATION_CHANCE
	switch po.strategy {
	case STRATEGY_LATENCY:
		return true // we want a lot of parallel tries on latency
	case STRATEGY_ACCURACY:
		return true
	case STRATEGY_COST:
		explorationChance = COST_EXPLORATION_CHANCE
	case STRATEGY_DISTRIBUTED:
		explorationChance = DEFAULT_EXPLORATION_CHANCE * 0.25
	case STRATEGY_PRIVACY:
		return false // only one at a time
	}
	// Dividing the random threshold by the loop count ensures that the overall probability of success is the requirement for the entire loop not per iteration
	return rand.Float64() < explorationChance/float64(numProviders)
}

func (po *ProviderOptimizer) isBetterProviderScore(latencyScore, latencyScoreCurrent, syncScore, syncScoreCurrent float64) bool {
	var latencyWeight float64
	switch po.strategy {
	case STRATEGY_LATENCY:
		latencyWeight = 0.7
	case STRATEGY_SYNC_FRESHNESS:
		latencyWeight = 0.2
	case STRATEGY_PRIVACY:
		// pick at random regardless of score
		if rand.Intn(2) == 0 {
			return true
		}
		return false
	default:
		latencyWeight = 0.6
	}
	if syncScoreCurrent == 0 {
		return latencyScore > latencyScoreCurrent
	}
	return latencyScore*latencyWeight+syncScore*(1-latencyWeight) > latencyScoreCurrent*latencyWeight+syncScoreCurrent*(1-latencyWeight)
}

func (po *ProviderOptimizer) calculateSyncScore(syncScore score.ScoreStore) float64 {
	var historicalSyncLatency time.Duration
	if syncScore.Denom == 0 {
		historicalSyncLatency = 0
	} else {
		historicalSyncLatency = time.Duration(syncScore.Num / syncScore.Denom * float64(po.averageBlockTime)) // give it units of block time
	}
	return historicalSyncLatency.Seconds()
}

func (po *ProviderOptimizer) calculateLatencyScore(providerData ProviderData, cu uint64, requestedBlock int64) float64 {
	baseLatency := po.baseWorldLatency + common.BaseTimePerCU(cu)/2 // divide by two because the returned time is for timeout not for average
	timeoutDuration := common.GetTimePerCu(cu) + common.AverageWorldLatency
	var historicalLatency time.Duration
	if providerData.Latency.Denom == 0 {
		historicalLatency = baseLatency
	} else {
		historicalLatency = time.Duration(float64(baseLatency) * providerData.Latency.Num / providerData.Latency.Denom)
	}
	if historicalLatency > timeoutDuration {
		// can't have a bigger latency than timeout
		historicalLatency = timeoutDuration
	}
	probabilityBlockError := po.CalculateProbabilityOfBlockError(requestedBlock, providerData)
	probabilityOfTimeout := po.CalculateProbabilityOfTimeout(providerData.Availability)
	probabilityOfSuccess := (1 - probabilityBlockError) * (1 - probabilityOfTimeout)

	// base latency is how much time it would cost to an average performing provider
	// timeoutDuration is the extra time we pay for a non responsive provider
	// historicalLatency is how much we are paying for the processing of this provider

	// in case of block error we are paying the time cost of this provider and the time cost of the next provider on retry
	costBlockError := historicalLatency.Seconds() + baseLatency.Seconds()
	if probabilityBlockError > 0.5 {
		costBlockError *= 3 // consistency improvement
	}
	// in case of a time out we are paying the time cost of a timeout and the time cost of the next provider on retry
	costTimeout := timeoutDuration.Seconds() + baseLatency.Seconds()
	// on success we are paying the time cost of this provider
	costSuccess := historicalLatency.Seconds()

	utils.LavaFormatTrace("latency calculation breakdown",
		utils.LogAttr("probabilityBlockError", probabilityBlockError),
		utils.LogAttr("costBlockError", costBlockError),
		utils.LogAttr("probabilityOfTimeout", probabilityOfTimeout),
		utils.LogAttr("costTimeout", costTimeout),
		utils.LogAttr("probabilityOfSuccess", probabilityOfSuccess),
		utils.LogAttr("costSuccess", costSuccess),
	)

	return probabilityBlockError*costBlockError + probabilityOfTimeout*costTimeout + probabilityOfSuccess*costSuccess
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
	// if there is no syncBlock data we assume successful relays so we don't over fit providers who were lucky to update
	if requestedBlock > 0 && providerData.SyncBlock < uint64(requestedBlock) && providerData.SyncBlock > 0 {
		// requested a specific block, so calculate a probability of provider having that block
		averageBlockTime := po.averageBlockTime.Seconds()
		blockDistanceRequired := uint64(requestedBlock) - providerData.SyncBlock
		if blockDistanceRequired > 0 {
			timeSinceSyncReceived := time.Since(providerData.Sync.Time).Seconds()
			eventRate := timeSinceSyncReceived / averageBlockTime // a new block every average block time, numerator is time passed, gamma=rt
			// probValueAfterRepetitions(k,lambda) calculates the probability for k events or less meaning p(x<=k),
			// an error occurs if we didn't have enough blocks, so the chance of error is p(x<k) where k is the required number of blocks so we do p(x<=k-1)
			probabilityBlockError = CumulativeProbabilityFunctionForPoissonDist(blockDistanceRequired-1, eventRate) // this calculates the probability we received insufficient blocks. too few
		} else {
			probabilityBlockError = 0
		}
	}
	return probabilityBlockError
}

func (po *ProviderOptimizer) getProviderData(providerAddress string) (providerData ProviderData, found bool) {
	storedVal, found := po.providersStorage.Get(providerAddress)
	if found {
		var ok bool

		providerData, ok = storedVal.(ProviderData)
		if !ok {
			utils.LavaFormatFatal("invalid usage of optimizer provider storage", nil, utils.Attribute{Key: "storedVal", Value: storedVal})
		}
	} else {
		providerData = ProviderData{
			Availability: score.NewScoreStore(0.99, 1, time.Now().Add(-1*INITIAL_DATA_STALENESS*time.Hour)), // default value of 99%
			Latency:      score.NewScoreStore(1, 1, time.Now().Add(-1*INITIAL_DATA_STALENESS*time.Hour)),    // default value of 1 score (encourage exploration)
			Sync:         score.NewScoreStore(1, 1, time.Now().Add(-1*INITIAL_DATA_STALENESS*time.Hour)),    // default value of half score (encourage exploration)
			SyncBlock:    0,
		}
	}
	return providerData, found
}

func (po *ProviderOptimizer) updateProbeEntrySync(providerData ProviderData, sync, baseSync, halfTime time.Duration, sampleTime time.Time) ProviderData {
	newScore := score.NewScoreStore(sync.Seconds(), baseSync.Seconds(), sampleTime)
	oldScore := providerData.Sync
	providerData.Sync = score.CalculateTimeDecayFunctionUpdate(oldScore, newScore, halfTime, RELAY_UPDATE_WEIGHT, sampleTime)
	return providerData
}

func (po *ProviderOptimizer) updateProbeEntryAvailability(providerData ProviderData, success bool, weight float64, halfTime time.Duration, sampleTime time.Time) ProviderData {
	newNumerator := float64(1)
	if !success {
		// if we failed we need the score update to be 0
		newNumerator = 0
	}
	oldScore := providerData.Availability
	newScore := score.NewScoreStore(newNumerator, 1, sampleTime) // denom is 1, entry time is now
	providerData.Availability = score.CalculateTimeDecayFunctionUpdate(oldScore, newScore, halfTime, weight, sampleTime)
	return providerData
}

// update latency data, base latency is the latency for the api defined in the spec
func (po *ProviderOptimizer) updateProbeEntryLatency(providerData ProviderData, latency, baseLatency time.Duration, weight float64, halfTime time.Duration, sampleTime time.Time) ProviderData {
	newScore := score.NewScoreStore(latency.Seconds(), baseLatency.Seconds(), sampleTime)
	oldScore := providerData.Latency
	providerData.Latency = score.CalculateTimeDecayFunctionUpdate(oldScore, newScore, halfTime, weight, sampleTime)
	return providerData
}

func (po *ProviderOptimizer) updateRelayTime(providerAddress string, sampleTime time.Time) {
	times := po.getRelayStatsTimes(providerAddress)
	if len(times) == 0 {
		po.providerRelayStats.Set(providerAddress, []time.Time{sampleTime}, 1)
		return
	}
	times = append(times, sampleTime)
	po.providerRelayStats.Set(providerAddress, times, 1)
}

func (po *ProviderOptimizer) calculateHalfTime(providerAddress string, sampleTime time.Time) time.Duration {
	halfTime := HALF_LIFE_TIME
	relaysHalfTime := po.getRelayStatsTimeDiff(providerAddress, sampleTime)
	if relaysHalfTime > halfTime {
		halfTime = relaysHalfTime
	}
	if halfTime > MAX_HALF_TIME {
		halfTime = MAX_HALF_TIME
	}
	return halfTime
}

func (po *ProviderOptimizer) getRelayStatsTimeDiff(providerAddress string, sampleTime time.Time) time.Duration {
	times := po.getRelayStatsTimes(providerAddress)
	if len(times) == 0 {
		return 0
	}
	medianTime := times[(len(times)-1)/2]
	if medianTime.Before(sampleTime) {
		return sampleTime.Sub(medianTime)
	}
	utils.LavaFormatWarning("did not use sample time in optimizer calculation", nil)
	return time.Since(medianTime)
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

func NewProviderOptimizer(strategy Strategy, averageBlockTIme, baseWorldLatency time.Duration, wantedNumProvidersInConcurrency uint) *ProviderOptimizer {
	cache, err := ristretto.NewCache(&ristretto.Config{NumCounters: CacheNumCounters, MaxCost: CacheMaxCost, BufferItems: 64, IgnoreInternalCost: true})
	if err != nil {
		utils.LavaFormatFatal("failed setting up cache for queries", err)
	}
	relayCache, err := ristretto.NewCache(&ristretto.Config{NumCounters: CacheNumCounters, MaxCost: CacheMaxCost, BufferItems: 64, IgnoreInternalCost: true})
	if err != nil {
		utils.LavaFormatFatal("failed setting up cache for queries", err)
	}
	if strategy == STRATEGY_PRIVACY {
		// overwrite
		wantedNumProvidersInConcurrency = 1
	}
	return &ProviderOptimizer{strategy: strategy, providersStorage: cache, averageBlockTime: averageBlockTIme, baseWorldLatency: baseWorldLatency, providerRelayStats: relayCache, wantedNumProvidersInConcurrency: wantedNumProvidersInConcurrency}
}

// calculate the probability a random variable with a poisson distribution
// poisson distribution calculates the probability of K events, in this case the probability enough blocks pass and the request will be accessible in the block

func CumulativeProbabilityFunctionForPoissonDist(k_events uint64, lambda float64) float64 {
	// calculate cumulative probability of observing k events (having k or more events):
	// GammaIncReg is the lower incomplete gamma function GammaIncReg(a,x) = (1/ Γ(a)) \int_0^x e^{-t} t^{a-1} dt
	// the CPF for k events (less than equal k) is the regularized upper incomplete gamma function
	// so to get the CPF we need to return 1 - prob
	argument := float64(k_events + 1)
	if argument <= 0 || lambda < 0 {
		utils.LavaFormatFatal("invalid function arguments", nil, utils.Attribute{Key: "argument", Value: argument}, utils.Attribute{Key: "lambda", Value: lambda})
	}
	prob := mathext.GammaIncReg(argument, lambda)
	return 1 - prob
}

func pertrubWithNormalGaussian(orig, percentage float64) float64 {
	perturb := rand.NormFloat64() * percentage * orig
	return orig + perturb
}

func (po *ProviderOptimizer) GetExcellenceQoSReportForProvider(providerAddress string) *pairingtypes.QualityOfServiceReport {
	providerData, found := po.getProviderData(providerAddress)
	if !found {
		return nil
	}
	precision := WANTED_PRECISION
	latencyScore := turnFloatToDec(providerData.Latency.Num/providerData.Latency.Denom, precision)
	syncScore := turnFloatToDec(providerData.Sync.Num/providerData.Sync.Denom, precision)
	// if our sync score is un initialized due to lack of providers
	if syncScore.IsZero() {
		syncScore = sdk.OneDec()
	}
	availabilityScore := turnFloatToDec(providerData.Availability.Num/providerData.Availability.Denom, precision)
	ret := &pairingtypes.QualityOfServiceReport{
		Latency:      latencyScore,
		Availability: availabilityScore,
		Sync:         syncScore,
	}

	utils.LavaFormatTrace("QoS Excellence for provider",
		utils.LogAttr("address", providerAddress),
		utils.LogAttr("Report", ret),
	)

	return ret
}

func turnFloatToDec(floatNum float64, precision int64) sdk.Dec {
	integerNum := int64(math.Round(floatNum * math.Pow(10, float64(precision))))
	return sdk.NewDecWithPrec(integerNum, precision)
}

func (po *ProviderOptimizer) Strategy() Strategy {
	return po.strategy
}
