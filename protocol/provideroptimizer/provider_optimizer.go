package provideroptimizer

import (
	"strings"
	"sync"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dgraph-io/ristretto/v2"
	"github.com/lavanet/lava/v5/protocol/metrics"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/lavaslices"
	"github.com/lavanet/lava/v5/utils/rand"
	"github.com/lavanet/lava/v5/utils/score"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	"gonum.org/v1/gonum/mathext"
)

// The provider optimizer is a mechanism within the consumer that is responsible for choosing
// the optimal provider for the consumer.
// The choice depends on the provider's QoS reputation metrics: latency, sync and availability.
// Providers are selected using weighted random selection based on their composite QoS scores
// and stake amounts.

const (
	CacheMaxCost             = 20000 // each item cost would be 1
	CacheNumCounters         = 20000 // expect 2000 items
	DefaultExplorationChance = 0.1
	CostExplorationChance    = 0.01
)

type ConcurrentBlockStore struct {
	Lock  sync.Mutex
	Time  time.Time
	Block uint64
}

type cacheInf interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, cost int64) bool
}

type consumerOptimizerQoSClientInf interface {
	UpdatePairingListStake(stakeMap map[string]int64, chainId string, epoch uint64)
}

type ProviderOptimizer struct {
	strategy                        Strategy
	providersStorage                cacheInf
	providerRelayStats              *ristretto.Cache[string, any] // used to decide on the half time of the decay
	averageBlockTime                time.Duration
	wantedNumProvidersInConcurrency uint
	latestSyncData                  ConcurrentBlockStore
	stakeCache                      ProviderStakeCache // provider stake amounts used in weighted selection
	consumerOptimizerQoSClient      consumerOptimizerQoSClientInf
	chainId                         string
	weightedSelector                *WeightedSelector // Weighted random selection based on composite QoS scores
}

type ProviderData struct {
	Availability score.ScoreStorer // will be used to calculate the probability of error
	Latency      score.ScoreStorer // will be used to calculate the latency score
	Sync         score.ScoreStorer // will be used to calculate the sync score for spectypes.LATEST_BLOCK/spectypes.NOT_APPLICABLE requests
	SyncBlock    uint64            // will be used to calculate the probability of block error
}

// Strategy defines the pairing strategy. Using different
// strategies allow users to determine the providers type they'll
// be paired with: providers with low latency, fresh sync and more.
type Strategy int

const (
	StrategyBalanced      Strategy = iota
	StrategyLatency                // prefer low latency
	StrategySyncFreshness          // prefer better sync
	StrategyCost                   // prefer low CU cost (minimize optimizer exploration)
	StrategyPrivacy                // prefer pairing with a single provider (not fully implemented)
	StrategyAccuracy               // encourage optimizer exploration (higher cost)
	StrategyDistributed            // prefer pairing with different providers (slightly minimize optimizer exploration)
)

func (s Strategy) String() string {
	switch s {
	case StrategyBalanced:
		return "balanced"
	case StrategyLatency:
		return "latency"
	case StrategySyncFreshness:
		return "sync_freshness"
	case StrategyCost:
		return "cost"
	case StrategyPrivacy:
		return "privacy"
	case StrategyAccuracy:
		return "accuracy"
	case StrategyDistributed:
		return "distributed"
	}

	return ""
}

// GetStrategyFactor gets the appropriate factor to multiply the sync factor
// with according to the strategy
func (s Strategy) GetStrategyFactor() math.LegacyDec {
	switch s {
	case StrategyLatency:
		return pairingtypes.LatencyStrategyFactor
	case StrategySyncFreshness:
		return pairingtypes.SyncFreshnessStrategyFactor
	}

	return pairingtypes.BalancedStrategyFactor
}

func (po *ProviderOptimizer) Strategy() Strategy {
	return po.strategy
}

// UpdateWeights updates provider stake amounts in the cache and metrics
func (po *ProviderOptimizer) UpdateWeights(weights map[string]int64, epoch uint64) {
	po.stakeCache.UpdateStakes(weights)

	// Update the stake map for metrics
	if po.consumerOptimizerQoSClient != nil {
		po.consumerOptimizerQoSClient.UpdatePairingListStake(weights, po.chainId, epoch)
	}
}

// AppendRelayFailure updates a provider's QoS metrics for a failed relay
func (po *ProviderOptimizer) AppendRelayFailure(provider string) {
	po.appendRelayData(provider, 0, false, 0, 0, time.Now())
}

// AppendRelayData updates a provider's QoS metrics for a successful relay
func (po *ProviderOptimizer) AppendRelayData(provider string, latency time.Duration, cu, syncBlock uint64) {
	po.appendRelayData(provider, latency, true, cu, syncBlock, time.Now())
}

// appendRelayData gets three new QoS metrics samples and updates the provider's metrics using a decaying weighted average
func (po *ProviderOptimizer) appendRelayData(provider string, latency time.Duration, success bool, cu, syncBlock uint64, sampleTime time.Time) {
	latestSync, timeSync := po.updateLatestSyncData(syncBlock, sampleTime)
	providerData, _ := po.getProviderData(provider)
	halfTime := po.calculateHalfTime(provider, sampleTime)
	weight := score.RelayUpdateWeight
	var updateErr error
	if success {
		// on a successful relay, update all the QoS metrics
		providerData, updateErr = po.updateDecayingWeightedAverage(providerData, score.AvailabilityScoreType, 1, weight, halfTime, cu, sampleTime)
		if updateErr != nil {
			return
		}
		providerData, updateErr = po.updateDecayingWeightedAverage(providerData, score.LatencyScoreType, latency.Seconds(), weight, halfTime, cu, sampleTime)
		if updateErr != nil {
			return
		}
		if syncBlock > providerData.SyncBlock {
			// do not allow providers to go back
			providerData.SyncBlock = syncBlock
		}
		syncLag := po.calculateSyncLag(latestSync, timeSync, providerData.SyncBlock, sampleTime)
		providerData, updateErr = po.updateDecayingWeightedAverage(providerData, score.SyncScoreType, syncLag.Seconds(), weight, halfTime, cu, sampleTime)
		if updateErr != nil {
			return
		}
	} else {
		// on a failed relay, update the availability metric with a failure score
		providerData, updateErr = po.updateDecayingWeightedAverage(providerData, score.AvailabilityScoreType, 0, weight, halfTime, cu, sampleTime)
		if updateErr != nil {
			return
		}
	}

	po.providersStorage.Set(provider, providerData, 1)
	po.updateRelayTime(provider, sampleTime)

	utils.LavaFormatTrace("[Optimizer] relay update",
		utils.LogAttr("providerData", providerData),
		utils.LogAttr("syncBlock", syncBlock),
		utils.LogAttr("cu", cu),
		utils.LogAttr("providerAddress", provider),
		utils.LogAttr("latency", latency),
		utils.LogAttr("success", success),
	)
}

// AppendProbeRelayData updates a provider's QoS metrics for a probe relay message
func (po *ProviderOptimizer) AppendProbeRelayData(providerAddress string, latency time.Duration, success bool) {
	providerData, _ := po.getProviderData(providerAddress)
	sampleTime := time.Now()
	halfTime := po.calculateHalfTime(providerAddress, sampleTime)
	weight := score.ProbeUpdateWeight
	var updateErr error
	if success {
		// update latency only on success
		providerData, updateErr = po.updateDecayingWeightedAverage(providerData, score.AvailabilityScoreType, 1, weight, halfTime, 0, sampleTime)
		if updateErr != nil {
			return
		}
		providerData, updateErr = po.updateDecayingWeightedAverage(providerData, score.LatencyScoreType, latency.Seconds(), weight, halfTime, 0, sampleTime)
		if updateErr != nil {
			return
		}
	} else {
		providerData, updateErr = po.updateDecayingWeightedAverage(providerData, score.AvailabilityScoreType, 0, weight, halfTime, 0, sampleTime)
		if updateErr != nil {
			return
		}
	}
	po.providersStorage.Set(providerAddress, providerData, 1)

	utils.LavaFormatTrace("[Optimizer] probe update",
		utils.LogAttr("providerAddress", providerAddress),
		utils.LogAttr("latency", latency),
		utils.LogAttr("success", success),
	)
}

// CalculateQoSScoresForMetrics calculates QoS scores for all providers for metrics reporting
func (po *ProviderOptimizer) CalculateQoSScoresForMetrics(allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) []*metrics.OptimizerQoSReport {
	// Get provider data for weighted selection
	providerDataGetter := func(addr string) (*pairingtypes.QualityOfServiceReport, time.Time, bool) {
		qos, lastUpdate := po.GetReputationReportForProvider(addr)
		if qos == nil {
			return nil, time.Time{}, false
		}
		return qos, lastUpdate, true
	}

	stakeGetter := func(addr string) int64 {
		return po.stakeCache.GetStake(addr)
	}

	// Calculate provider scores using weighted selector
	_, qosReports := po.weightedSelector.CalculateProviderScores(
		allAddresses,
		ignoredProviders,
		providerDataGetter,
		stakeGetter,
	)

	// Convert map to slice and add entry indices
	reports := make([]*metrics.OptimizerQoSReport, 0, len(qosReports))
	idx := 0
	for _, report := range qosReports {
		report.EntryIndex = idx
		reports = append(reports, report)
		idx++
	}

	return reports
}

// ChooseProvider returns a subset of selected providers using weighted random selection based on QoS scores
func (po *ProviderOptimizer) ChooseProvider(allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) (addresses []string, tier int) {
	// Get provider data for weighted selection
	providerDataGetter := func(addr string) (*pairingtypes.QualityOfServiceReport, time.Time, bool) {
		qos, lastUpdate := po.GetReputationReportForProvider(addr)
		if qos == nil {
			return nil, time.Time{}, false
		}
		return qos, lastUpdate, true
	}

	stakeGetter := func(addr string) int64 {
		// Get stake from provider stake cache
		return po.stakeCache.GetStake(addr)
	}

	// Calculate provider scores using weighted selector
	providerScores, qosReports := po.weightedSelector.CalculateProviderScores(
		allAddresses,
		ignoredProviders,
		providerDataGetter,
		stakeGetter,
	)

	if len(providerScores) == 0 {
		// No providers to choose from
		utils.LavaFormatWarning("[Optimizer] no providers available for selection", nil)
		return []string{}, -1
	}

	// Select provider using weighted random selection
	selectedProvider := po.weightedSelector.SelectProvider(providerScores)
	returnedProviders := []string{selectedProvider}

	// Add exploration candidate if should explore
	// Find the exploration candidate (provider not updated recently)
	explorationCandidate := ""
	oldestUpdateTime := time.Now()
	for addr := range qosReports {
		if _, ignored := ignoredProviders[addr]; ignored {
			continue
		}
		if addr == selectedProvider {
			continue // Don't add the same provider twice
		}
		_, lastUpdate, found := providerDataGetter(addr)
		if found && lastUpdate.Add(10*time.Second).Before(time.Now()) && lastUpdate.Before(oldestUpdateTime) {
			explorationCandidate = addr
			oldestUpdateTime = lastUpdate
		}
	}

	if explorationCandidate != "" && po.shouldExplore(1) {
		returnedProviders = append(returnedProviders, explorationCandidate)
	}

	utils.LavaFormatTrace("[Optimizer] returned providers",
		utils.LogAttr("providers", strings.Join(returnedProviders, ",")),
		utils.LogAttr("selectedScore", getProviderScore(selectedProvider, providerScores)),
		utils.LogAttr("numScores", len(providerScores)),
	)

	// Return -1 for tier (no longer using tiers)
	return returnedProviders, -1
}

// getProviderScore is a helper function to find a provider's score in the scores list
func getProviderScore(address string, scores []ProviderScore) float64 {
	for _, ps := range scores {
		if ps.Address == address {
			return ps.CompositeScore
		}
	}
	return 0.0
}

// ChooseProviderFromTopTier selects a single high-quality provider using weighted selection
// This is used for sticky sessions and other scenarios requiring consistent provider selection
func (po *ProviderOptimizer) ChooseProviderFromTopTier(allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) (addresses []string) {
	// Get provider data for weighted selection
	providerDataGetter := func(addr string) (*pairingtypes.QualityOfServiceReport, time.Time, bool) {
		qos, lastUpdate := po.GetReputationReportForProvider(addr)
		if qos == nil {
			return nil, time.Time{}, false
		}
		return qos, lastUpdate, true
	}

	stakeGetter := func(addr string) int64 {
		return po.stakeCache.GetStake(addr)
	}

	// Calculate provider scores
	providerScores, _ := po.weightedSelector.CalculateProviderScores(
		allAddresses,
		ignoredProviders,
		providerDataGetter,
		stakeGetter,
	)

	if len(providerScores) == 0 {
		utils.LavaFormatWarning("[Optimizer] no providers available for selection", nil)
		return []string{}
	}

	// Select the single best provider using weighted random selection
	// This gives higher probability to better providers while still allowing variety
	selectedProvider := po.weightedSelector.SelectProvider(providerScores)

	utils.LavaFormatTrace("[Optimizer] returned provider",
		utils.LogAttr("provider", selectedProvider),
		utils.LogAttr("score", getProviderScore(selectedProvider, providerScores)),
		utils.LogAttr("numCandidates", len(providerScores)),
	)

	return []string{selectedProvider}
}

// CalculateProbabilityOfBlockError calculates the probability that a provider doesn't a specific requested
// block when the consumer asks the optimizer to fetch a provider with the specific block
func (po *ProviderOptimizer) CalculateProbabilityOfBlockError(requestedBlock int64, providerData ProviderData) sdk.Dec {
	probabilityBlockError := float64(0)
	// if there is no syncBlock data we assume successful relays so we don't over fit providers who were lucky to update
	if requestedBlock > 0 && providerData.SyncBlock < uint64(requestedBlock) && providerData.SyncBlock > 0 {
		// requested a specific block, so calculate a probability of provider having that block
		averageBlockTime := po.averageBlockTime.Seconds()
		blockDistanceRequired := uint64(requestedBlock) - providerData.SyncBlock
		if blockDistanceRequired > 0 {
			timeSinceSyncReceived := time.Since(providerData.Sync.GetLastUpdateTime()).Seconds()
			eventRate := timeSinceSyncReceived / averageBlockTime // a new block every average block time, numerator is time passed, gamma=rt
			// probValueAfterRepetitions(k,lambda) calculates the probability for k events or less meaning p(x<=k),
			// an error occurs if we didn't have enough blocks, so the chance of error is p(x<k) where k is the required number of blocks so we do p(x<=k-1)
			probabilityBlockError = CumulativeProbabilityFunctionForPoissonDist(blockDistanceRequired-1, eventRate) // this calculates the probability we received insufficient blocks. too few
		} else {
			probabilityBlockError = 0
		}
	}
	return score.ConvertToDec(probabilityBlockError)
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

// shouldExplore determines whether the optimizer should continue exploring
// after finding an appropriate provider for pairing.
// The exploration mechanism makes the optimizer return providers that were not talking
// to the consumer for a long time (a couple of seconds). This allows better distribution
// of paired providers by avoiding returning the same best providers over and over.
// Note, the legacy disributed strategy acts as the default balanced strategy
func (po *ProviderOptimizer) shouldExplore(currentNumProviders int) bool {
	if uint(currentNumProviders) >= po.wantedNumProvidersInConcurrency {
		return false
	}
	explorationChance := DefaultExplorationChance
	switch po.strategy {
	case StrategyLatency:
		return true // we want a lot of parallel tries on latency
	case StrategyAccuracy:
		return true
	case StrategyCost:
		explorationChance = CostExplorationChance
	case StrategyDistributed:
		explorationChance = DefaultExplorationChance * 0.25
	case StrategyPrivacy:
		return false // only one at a time
	}
	return rand.Float64() < explorationChance
}

// getProviderData gets a specific proivder's QoS data. If it doesn't exist, it returns a default provider data struct
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
			Availability: score.NewScoreStore(score.AvailabilityScoreType), // default score of 100%
			Latency:      score.NewScoreStore(score.LatencyScoreType),      // default score of 10ms (encourage exploration)
			Sync:         score.NewScoreStore(score.SyncScoreType),         // default score of 100ms (encourage exploration)
			SyncBlock:    0,
		}
	}

	return providerData, found
}

func (po *ProviderOptimizer) validateUpdateError(err error, errorMsg string) error {
	if !score.TimeConflictingScoresError.Is(err) {
		utils.LavaFormatError(errorMsg, err)
	}
	return err
}

// updateDecayingWeightedAverage updates a provider's QoS metric ScoreStore with a new sample
func (po *ProviderOptimizer) updateDecayingWeightedAverage(providerData ProviderData, scoreType string, sample float64, weight float64, halfTime time.Duration, cu uint64, sampleTime time.Time) (ProviderData, error) {
	switch scoreType {
	case score.LatencyScoreType:
		err := providerData.Latency.UpdateConfig(
			score.WithWeight(weight),
			score.WithDecayHalfLife(halfTime),
			score.WithLatencyCuFactor(score.GetLatencyFactor(cu)),
		)
		if err != nil {
			utils.LavaFormatError("[UpdateConfig] did not update provider latency score", err)
			return providerData, err
		}
		err = providerData.Latency.Update(sample, sampleTime)
		if err != nil {
			return providerData, po.validateUpdateError(err, "[Update] did not update provider latency score")
		}

	case score.SyncScoreType:
		err := providerData.Sync.UpdateConfig(score.WithWeight(weight), score.WithDecayHalfLife(halfTime))
		if err != nil {
			utils.LavaFormatError("[UpdateConfig] did not update provider sync score", err)
			return providerData, err
		}
		err = providerData.Sync.Update(sample, sampleTime)
		if err != nil {
			return providerData, po.validateUpdateError(err, "[Update] did not update provider sync score")
		}

	case score.AvailabilityScoreType:
		err := providerData.Availability.UpdateConfig(score.WithWeight(weight), score.WithDecayHalfLife(halfTime))
		if err != nil {
			utils.LavaFormatError("[UpdateConfig] did not update provider availability score", err)
			return providerData, err
		}
		err = providerData.Availability.Update(sample, sampleTime)
		if err != nil {
			return providerData, po.validateUpdateError(err, "[Update] did not update provider availability score")
		}
	}

	return providerData, nil
}

// updateRelayTime adds a relay sample time to a provider's data
func (po *ProviderOptimizer) updateRelayTime(providerAddress string, sampleTime time.Time) {
	times := po.getRelayStatsTimes(providerAddress)
	if len(times) == 0 {
		po.providerRelayStats.Set(providerAddress, []time.Time{sampleTime}, 1)
		return
	}
	times = append(times, sampleTime)
	po.providerRelayStats.Set(providerAddress, times, 1)
}

// calculateHalfTime calculates a provider's half life time for a relay sampled in sampleTime
func (po *ProviderOptimizer) calculateHalfTime(providerAddress string, sampleTime time.Time) time.Duration {
	halfTime := score.DefaultHalfLifeTime
	relaysHalfTime := po.getRelayStatsTimeDiff(providerAddress, sampleTime)
	if relaysHalfTime > halfTime {
		halfTime = relaysHalfTime
	}
	if halfTime > score.MaxHalfTime {
		halfTime = score.MaxHalfTime
	}
	return halfTime
}

// getRelayStatsTimeDiff returns the time passed since the provider optimizer's saved relay times median
func (po *ProviderOptimizer) getRelayStatsTimeDiff(providerAddress string, sampleTime time.Time) time.Duration {
	times := po.getRelayStatsTimes(providerAddress)
	if len(times) == 0 {
		return 0
	}
	medianTime := times[(len(times)-1)/2]
	if medianTime.Before(sampleTime) {
		return sampleTime.Sub(medianTime)
	}
	utils.LavaFormatWarning("did not use sample time in optimizer calculation", nil,
		utils.LogAttr("median", medianTime.UTC().Unix()),
		utils.LogAttr("sample", sampleTime.UTC().Unix()),
		utils.LogAttr("diff", sampleTime.UTC().Unix()-medianTime.UTC().Unix()),
	)
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

func NewProviderOptimizer(strategy Strategy, averageBlockTIme time.Duration, wantedNumProvidersInConcurrency uint, consumerOptimizerQoSClient consumerOptimizerQoSClientInf, chainId string, qosSelectionEnabled bool) *ProviderOptimizer {
	cache, err := ristretto.NewCache(&ristretto.Config[string, any]{NumCounters: CacheNumCounters, MaxCost: CacheMaxCost, BufferItems: 64, IgnoreInternalCost: true})
	if err != nil {
		utils.LavaFormatFatal("failed setting up cache for queries", err)
	}
	relayCache, err := ristretto.NewCache(&ristretto.Config[string, any]{NumCounters: CacheNumCounters, MaxCost: CacheMaxCost, BufferItems: 64, IgnoreInternalCost: true})
	if err != nil {
		utils.LavaFormatFatal("failed setting up cache for queries", err)
	}
	if strategy == StrategyPrivacy {
		// overwrite
		wantedNumProvidersInConcurrency = 1
	}

	// Initialize weighted selector with default configuration
	weightedConfig := DefaultWeightedSelectorConfig()
	weightedConfig.Strategy = strategy
	weightedSelector := NewWeightedSelector(weightedConfig)

	return &ProviderOptimizer{
		strategy:                        strategy,
		providersStorage:                cache,
		averageBlockTime:                averageBlockTIme,
		providerRelayStats:              relayCache,
		wantedNumProvidersInConcurrency: wantedNumProvidersInConcurrency,
		stakeCache:                      NewProviderStakeCache(),
		consumerOptimizerQoSClient:      consumerOptimizerQoSClient,
		chainId:                         chainId,
		weightedSelector:                weightedSelector,
	}
}

func (po *ProviderOptimizer) GetReputationReportForProvider(providerAddress string) (report *pairingtypes.QualityOfServiceReport, lastUpdateTime time.Time) {
	providerData, found := po.getProviderData(providerAddress)
	if !found {
		utils.LavaFormatWarning("provider data not found, using default", nil, utils.LogAttr("address", providerAddress))
	}

	latency, err := providerData.Latency.Resolve()
	if err != nil {
		utils.LavaFormatError("could not resolve latency score", err, utils.LogAttr("address", providerAddress))
		return nil, time.Time{}
	}
	if latency > score.WorstLatencyScore {
		latency = score.WorstLatencyScore
	}

	sync, err := providerData.Sync.Resolve()
	if err != nil {
		utils.LavaFormatError("could not resolve sync score", err, utils.LogAttr("address", providerAddress))
		return nil, time.Time{}
	}
	if sync == 0 {
		// if our sync score is uninitialized due to lack of providers
		// note, we basically penalize perfect providers, but assigning the sync score to 1
		// is making it 1ms, which is a very low value that doesn't harm the provider's score
		// too much
		sync = 1
	} else if sync > score.WorstSyncScore {
		sync = score.WorstSyncScore
	}

	availability, err := providerData.Availability.Resolve()
	if err != nil {
		utils.LavaFormatError("could not resolve availability score", err, utils.LogAttr("address", providerAddress))
		return nil, time.Time{}
	}

	report = &pairingtypes.QualityOfServiceReport{
		Latency:      score.ConvertToDec(latency),
		Availability: score.ConvertToDec(availability),
		Sync:         score.ConvertToDec(sync),
	}

	utils.LavaFormatTrace("[Optimizer] QoS Excellence for provider",
		utils.LogAttr("address", providerAddress),
		utils.LogAttr("report", report),
	)

	return report, providerData.Latency.GetLastUpdateTime()
}

// UpdateWeightedSelectorStrategy updates the weighted selector's strategy
// This should be called when the optimizer's strategy changes
func (po *ProviderOptimizer) UpdateWeightedSelectorStrategy(strategy Strategy) {
	if po.weightedSelector != nil {
		po.weightedSelector.UpdateStrategy(strategy)
		utils.LavaFormatTrace("[Optimizer] weighted selector strategy updated",
			utils.LogAttr("strategy", strategy.String()),
		)
	}
}

// GetWeightedSelectorConfig returns the current weighted selector configuration
func (po *ProviderOptimizer) GetWeightedSelectorConfig() WeightedSelectorConfig {
	if po.weightedSelector != nil {
		return po.weightedSelector.GetConfig()
	}
	return WeightedSelectorConfig{}
}
