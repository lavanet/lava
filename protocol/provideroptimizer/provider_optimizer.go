package provideroptimizer

import (
	"context"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/lavanet/lava/v5/protocol/chainstate"
	"github.com/lavanet/lava/v5/protocol/metrics"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/lavaslices"
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
	CacheMaxCost     = 20000 // each item cost would be 1
	CacheNumCounters = 20000 // expect 2000 items
)

type ConcurrentBlockStore struct {
	Lock  sync.Mutex
	Time  time.Time
	Block uint64
}

type cacheInf interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, cost int64) bool
	// Clear empties the cache. Used by ResetState to discard future-dated entries
	// after a debug clock reset so that real-time samples are no longer rejected.
	Clear()
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
	stakeCache                      ProviderStakeCache // provider stake amounts used in weighted selection
	consumerOptimizerQoSClient      consumerOptimizerQoSClientInf
	chainId                         string
	weightedSelector                *WeightedSelector            // Weighted random selection based on composite QoS scores
	globalLatencyCalculator         *score.AdaptiveMaxCalculator // Global T-Digest for all providers' latency samples
	globalSyncCalculator            *score.AdaptiveMaxCalculator // Global T-Digest for all providers' sync samples
	adaptiveLock                    sync.RWMutex                 // Lock for accessing adaptive calculators
	NowFunc                         func() time.Time             // NowFunc overrides the clock used for score updates nil = use real time.Now()
	chainState                      *chainstate.ChainState       // shared per-chain state — single source of truth for majorityBaseline + latestBlock
	metricsManager                  metrics.ConsumerMetricsManagerInf // unification observability (M6: sync-scoring outlier-skip per provider)
}

func (po *ProviderOptimizer) GetAverageBlockTime() time.Duration {
	return po.averageBlockTime
}

type ProviderData struct {
	Availability score.ScoreStorer // will be used to calculate the probability of error
	Latency      score.ScoreStorer // will be used to calculate the latency score
	Sync         score.ScoreStorer // will be used to calculate the sync score for spectypes.LATEST_BLOCK/spectypes.NOT_APPLICABLE requests
}

// Strategy defines the pairing strategy. Using different
// strategies allow users to determine the providers type they'll
// be paired with: providers with low latency, fresh sync and more.
type Strategy int

const (
	StrategyBalanced      Strategy = iota
	StrategyLatency                // prefer low latency
	StrategySyncFreshness          // prefer better sync
	StrategyCost                   // prefer low CU cost
	StrategyPrivacy                // prefer pairing with a single provider (not fully implemented)
	StrategyAccuracy               // higher cost for more accuracy
	StrategyDistributed            // prefer pairing with different providers
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

func (po *ProviderOptimizer) Strategy() Strategy {
	return po.strategy
}

// ConfigureWeightedSelector rebuilds the weighted selector using the supplied
// configuration. Strategy is always enforced from the optimizer so callers only
// provide weights and selection chance values.
func (po *ProviderOptimizer) ConfigureWeightedSelector(config WeightedSelectorConfig) {
	if po == nil {
		return
	}
	config.Strategy = po.strategy

	// Wire up Phase 2: Enable adaptive P10-P90 normalization
	config.UseAdaptiveLatencyMax = true
	config.AdaptiveLatencyGetter = po.getAdaptiveLatencyBounds

	config.UseAdaptiveSyncMax = true
	config.AdaptiveSyncGetter = po.getAdaptiveSyncBounds

	po.weightedSelector = NewWeightedSelector(config)
}

// getAdaptiveLatencyBounds returns the current P10 and P90 bounds for latency normalization
// from the global T-Digest that aggregates data from all providers
func (po *ProviderOptimizer) getAdaptiveLatencyBounds() (p10, p90 float64) {
	if po == nil {
		return score.AdaptiveP10MinBound, score.DefaultLatencyAdaptiveMaxMax
	}

	po.adaptiveLock.RLock()
	defer po.adaptiveLock.RUnlock()

	if po.globalLatencyCalculator == nil {
		return score.AdaptiveP10MinBound, score.DefaultLatencyAdaptiveMaxMax
	}

	p10, p90 = po.globalLatencyCalculator.GetAdaptiveBounds()
	if math.IsNaN(p10) || math.IsNaN(p90) || math.IsInf(p10, 0) || math.IsInf(p90, 0) || p10 <= 0 || p90 <= 0 || p90 <= p10 {
		utils.LavaFormatWarning("invalid adaptive latency bounds, using defaults",
			nil,
			utils.LogAttr("p10", p10),
			utils.LogAttr("p90", p90),
		)
		return score.AdaptiveP10MinBound, score.DefaultLatencyAdaptiveMaxMax
	}
	return p10, p90
}

// getAdaptiveSyncBounds returns the current P10 and P90 bounds for sync normalization
// from the global T-Digest that aggregates data from all providers
func (po *ProviderOptimizer) getAdaptiveSyncBounds() (p10, p90 float64) {
	if po == nil {
		return score.AdaptiveSyncP10MinBound, score.DefaultSyncAdaptiveMaxMax
	}

	po.adaptiveLock.RLock()
	defer po.adaptiveLock.RUnlock()

	if po.globalSyncCalculator == nil {
		return score.AdaptiveSyncP10MinBound, score.DefaultSyncAdaptiveMaxMax
	}

	p10, p90 = po.globalSyncCalculator.GetAdaptiveBounds()
	if math.IsNaN(p10) || math.IsNaN(p90) || math.IsInf(p10, 0) || math.IsInf(p90, 0) || p10 <= 0 || p90 <= 0 || p90 <= p10 {
		utils.LavaFormatWarning("invalid adaptive sync bounds, using defaults",
			nil,
			utils.LogAttr("p10", p10),
			utils.LogAttr("p90", p90),
		)
		return score.AdaptiveSyncP10MinBound, score.DefaultSyncAdaptiveMaxMax
	}
	return p10, p90
}

// UpdateWeights updates provider stake amounts in the cache and metrics
func (po *ProviderOptimizer) UpdateWeights(weights map[string]int64, epoch uint64) {
	po.stakeCache.UpdateStakes(weights)

	// Update the stake map for metrics
	if po.consumerOptimizerQoSClient != nil {
		po.consumerOptimizerQoSClient.UpdatePairingListStake(weights, po.chainId, epoch)
	}
}

// now returns the current time, using NowFunc if set (for testing) or time.Now() otherwise
// This allows us to control time in tests for deterministic behavior
func (po *ProviderOptimizer) now() time.Time {
	if po.NowFunc != nil {
		return po.NowFunc()
	}
	return time.Now()
}

// ResetState clears all time-dependent internal state so the optimizer works correctly
// after a debug clock reset (i.e. when the time offset is set back to 0).
//
// Why this is necessary:
// When the clock is shifted forward (e.g. +24 h via the debug server), all ScoreStore
// entries written during that window carry future timestamps.  When the offset is reset to
// 0, po.now() returns real time again — but ScoreStore.Update rejects any sample whose
// timestamp is earlier than the stored one ("TimeConflictingScoresError").  That means
// every new relay sample would be silently dropped for the next 24 hours, leaving the
// optimizer effectively frozen.  Calling ResetState discards all the future-dated data so
// incoming real-time samples are accepted immediately.
func (po *ProviderOptimizer) ResetState() {
	// Discard all per-provider score caches.  Every ProviderData entry holds ScoreStore
	// objects whose Time field was advanced to the shifted period; without clearing them
	// new real-time samples would be rejected by the TimeConflictingScores guard.
	po.providersStorage.Clear()

	// Discard relay-stats timestamps (used for half-time / sync-lag calculations).
	// Future-dated relay times would produce negative or wildly inflated durations.
	po.providerRelayStats.Clear()

	// Step 4 removed the per-optimizer latestSyncData store; chainState now owns the
	// unified latestBlock tracker and is shared across optimizers, so per-optimizer
	// reset is no longer applicable here.

	// Reset both global adaptive calculators under their shared write lock.
	// T-Digest samples recorded at shifted timestamps distort the P10/P90 bounds
	// used for score normalisation until they decay out — resetting clears them
	// instantly so normalization is back to defaults right away.
	po.adaptiveLock.Lock()
	defer po.adaptiveLock.Unlock()
	if po.globalLatencyCalculator != nil {
		po.globalLatencyCalculator.Reset()
	}
	if po.globalSyncCalculator != nil {
		po.globalSyncCalculator.Reset()
	}
}

// AppendRelayFailure updates a provider's QoS metrics for a failed relay
func (po *ProviderOptimizer) AppendRelayFailure(provider string) {
	po.appendRelayData(provider, 0, false, 0, 0, po.now())
}

// AppendRelayData updates a provider's QoS metrics for a successful relay
func (po *ProviderOptimizer) AppendRelayData(provider string, latency time.Duration, cu, syncBlock uint64) {
	po.appendRelayData(provider, latency, true, cu, syncBlock, po.now())
}

// appendRelayData gets three new QoS metrics samples and updates the provider's metrics using a decaying weighted average
func (po *ProviderOptimizer) appendRelayData(provider string, latency time.Duration, success bool, cu, syncBlock uint64, sampleTime time.Time) {
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
		// Sync scoring: outlier samples drop ALL scoring contributions (avail + latency
		// updates earlier in this branch are not persisted because Set is below the
		// fall-through point). Per-sample syncBlock feeds calculateSyncLag directly;
		// the Sync EWMA's half-life provides smoothing.
		//
		// Sanity guard: a uint64 with the high bit set wraps to a negative int64 and
		// would silently bypass IsOutlier. Realistic block heights never approach 2^63.
		if syncBlock > uint64(math.MaxInt64) {
			return
		}
		if po.chainState.IsOutlier(int64(syncBlock)) {
			// M6: per-provider visibility into sync-scoring outlier-skip (Option B drop-all-scoring).
			po.metricsManager.SetSyncScoringOutlierSkipped(po.chainId, provider, "relay")
			return
		}
		block, ts, advanced := po.chainState.SetLatestBlock(int64(syncBlock))
		latestSync, timeSync := uint64(block), ts
		// Race-window detection: a concurrent SetMajorityBaseline between IsOutlier (read
		// lock) and SetLatestBlock (write lock) can raise the bar such that syncBlock is
		// now an outlier; SetLatestBlock's internal guard then rejects it, returning
		// advanced=false with the existing snapshot. `!advanced && syncBlock > latestSync`
		// uniquely identifies that path (monotonic-rejected has syncBlock <= latestSync;
		// advance has advanced=true). Drop all scoring, same as the front-door guard.
		if !advanced && syncBlock > latestSync {
			po.metricsManager.SetSyncScoringOutlierSkipped(po.chainId, provider, "relay")
			return
		}
		syncLag := po.calculateSyncLag(latestSync, timeSync, syncBlock, sampleTime)
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
		utils.LogAttr("cu", cu),
		utils.LogAttr("providerAddress", provider),
		utils.LogAttr("latency", latency),
		utils.LogAttr("success", success),
	)
}

// AppendProbeRelayData updates a provider's QoS metrics for a probe relay message
func (po *ProviderOptimizer) AppendProbeRelayData(providerAddress string, latency time.Duration, success bool, syncBlock uint64) {
	providerData, _ := po.getProviderData(providerAddress)
	sampleTime := po.now()
	halfTime := po.calculateHalfTime(providerAddress, sampleTime)
	weight := score.ProbeUpdateWeight
	var updateErr error
	if success {
		providerData, updateErr = po.updateDecayingWeightedAverage(providerData, score.AvailabilityScoreType, 1, weight, halfTime, 0, sampleTime)
		if updateErr != nil {
			return
		}
		providerData, updateErr = po.updateDecayingWeightedAverage(providerData, score.LatencyScoreType, latency.Seconds(), weight, halfTime, 0, sampleTime)
		if updateErr != nil {
			return
		}
		// Sync scoring: mirror appendRelayData. Skip when syncBlock=0 (static providers
		// or failed probes that didn't return a block height). See appendRelayData for
		// the full rationale on the sanity guard, outlier guard, and race-window detection.
		if syncBlock > 0 {
			if syncBlock > uint64(math.MaxInt64) {
				return
			}
			if po.chainState.IsOutlier(int64(syncBlock)) {
				po.metricsManager.SetSyncScoringOutlierSkipped(po.chainId, providerAddress, "probe")
				return
			}
			block, ts, advanced := po.chainState.SetLatestBlock(int64(syncBlock))
			latestSync, timeSync := uint64(block), ts
			if !advanced && syncBlock > latestSync {
				po.metricsManager.SetSyncScoringOutlierSkipped(po.chainId, providerAddress, "probe")
				return
			}
			syncLag := po.calculateSyncLag(latestSync, timeSync, syncBlock, sampleTime)
			providerData, updateErr = po.updateDecayingWeightedAverage(providerData, score.SyncScoreType, syncLag.Seconds(), weight, halfTime, 0, sampleTime)
			if updateErr != nil {
				return
			}
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
func (po *ProviderOptimizer) CalculateQoSScoresForMetrics(allAddresses []string, ignoredProviders map[string]struct{}) []*metrics.OptimizerQoSReport {
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
	_, qosReports, _ := po.weightedSelector.CalculateProviderScores(
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
func (po *ProviderOptimizer) ChooseProvider(ctx context.Context, allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) (addresses []string) {
	addresses, _ = po.ChooseProviderWithStats(ctx, allAddresses, ignoredProviders, cu, requestedBlock)
	return addresses
}

// ChooseProviderWithStats returns a subset of selected providers and detailed selection statistics
func (po *ProviderOptimizer) ChooseProviderWithStats(ctx context.Context, allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) (addresses []string, stats *SelectionStats) {
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
	providerScores, _, scoreDetails := po.weightedSelector.CalculateProviderScores(
		allAddresses,
		ignoredProviders,
		providerDataGetter,
		stakeGetter,
	)

	if len(providerScores) == 0 {
		// No providers to choose from
		utils.LavaFormatWarning("[Optimizer] no providers available for selection", nil)
		return []string{}, nil
	}

	// Select provider using weighted random selection with stats
	selectedProvider, selectionStats := po.weightedSelector.SelectProviderWithStats(ctx, providerScores, scoreDetails)
	returnedProviders := []string{selectedProvider}

	utils.LavaFormatTrace("[Optimizer] returned providers",
		utils.LogAttr("providers", strings.Join(returnedProviders, ",")),
		utils.LogAttr("selectedWeight", getProviderSelectionWeight(selectedProvider, providerScores)),
		utils.LogAttr("selectedCompositeScore", getProviderCompositeScore(selectedProvider, providerScores)),
		utils.LogAttr("numScores", len(providerScores)),
		utils.LogAttr("requestedBlock", requestedBlock),
	)

	return returnedProviders, selectionStats
}

// getProviderScore is a helper function to find a provider's score in the scores list
func getProviderSelectionWeight(address string, scores []ProviderScore) float64 {
	for _, ps := range scores {
		if ps.Address == address {
			return ps.SelectionWeight
		}
	}
	return 0.0
}

func getProviderCompositeScore(address string, scores []ProviderScore) float64 {
	for _, ps := range scores {
		if ps.Address == address {
			return ps.CompositeScore
		}
	}
	return 0.0
}

// ChooseBestProvider selects a single high-quality provider using weighted selection
// This is used for sticky sessions and other scenarios requiring consistent provider selection
func (po *ProviderOptimizer) ChooseBestProvider(ctx context.Context, allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) (addresses []string) {
	addresses, _ = po.ChooseBestProviderWithStats(ctx, allAddresses, ignoredProviders, cu, requestedBlock)
	return addresses
}

// ChooseBestProviderWithStats selects a single high-quality provider and returns detailed selection statistics
func (po *ProviderOptimizer) ChooseBestProviderWithStats(ctx context.Context, allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) (addresses []string, stats *SelectionStats) {
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
	providerScores, _, scoreDetails := po.weightedSelector.CalculateProviderScores(
		allAddresses,
		ignoredProviders,
		providerDataGetter,
		stakeGetter,
	)

	if len(providerScores) == 0 {
		utils.LavaFormatWarning("[Optimizer] no providers available for selection", nil)
		return []string{}, nil
	}

	// Select the single best provider using weighted random selection
	// This gives higher probability to better providers while still allowing variety
	selectedProvider, selectionStats := po.weightedSelector.SelectProviderWithStats(ctx, providerScores, scoreDetails)

	utils.LavaFormatTrace("[Optimizer] returned provider",
		utils.LogAttr("provider", selectedProvider),
		utils.LogAttr("selectedWeight", getProviderSelectionWeight(selectedProvider, providerScores)),
		utils.LogAttr("selectedCompositeScore", getProviderCompositeScore(selectedProvider, providerScores)),
		utils.LogAttr("numCandidates", len(providerScores)),
		utils.LogAttr("requestedBlock", requestedBlock),
	)

	return []string{selectedProvider}, selectionStats
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
			Latency:      score.NewScoreStore(score.LatencyScoreType),      // default score of 10ms
			Sync:         score.NewScoreStore(score.SyncScoreType),         // default score of 100ms
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

		// Phase 2: Feed sample to global T-Digest for adaptive normalization
		// Apply the same latency CU factor as the score store
		adjustedSample := sample * score.GetLatencyFactor(cu)
		po.adaptiveLock.Lock()
		if po.globalLatencyCalculator != nil {
			if err := po.globalLatencyCalculator.AddSample(adjustedSample, sampleTime); err != nil {
				utils.LavaFormatWarning("failed to update global latency adaptive calculator",
					err,
					utils.LogAttr("sample", adjustedSample),
					utils.LogAttr("sampleTime", sampleTime),
				)
			}
		}
		po.adaptiveLock.Unlock()

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

		// Phase 2: Feed sample to global T-Digest for adaptive normalization
		po.adaptiveLock.Lock()
		if po.globalSyncCalculator != nil {
			if err := po.globalSyncCalculator.AddSample(sample, sampleTime); err != nil {
				utils.LavaFormatWarning("failed to update global sync adaptive calculator",
					err,
					utils.LogAttr("sample", sample),
					utils.LogAttr("sampleTime", sampleTime),
				)
			}
		}
		po.adaptiveLock.Unlock()

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

func NewProviderOptimizer(strategy Strategy, averageBlockTIme time.Duration, wantedNumProvidersInConcurrency uint, consumerOptimizerQoSClient consumerOptimizerQoSClientInf, chainId string, cState *chainstate.ChainState, metricsManager metrics.ConsumerMetricsManagerInf) *ProviderOptimizer {
	// ChainState is mandatory: ProviderOptimizer reads/writes it on every relay and every
	// probe. A nil cState would silently zero the sync path; refusing it at construction
	// keeps the failure mode loud.
	if cState == nil {
		utils.LavaFormatFatal("ProviderOptimizer requires a non-nil ChainState", nil)
	}
	// metricsManager is wrapped in SafeMetrics so a nil parameter from test scaffolding
	// becomes a no-op without nil-checking on every emit (hot path).
	metricsManager = metrics.SafeMetrics(metricsManager)
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

	// Initialize global adaptive calculators for Phase 2 (P10-P90 normalization)
	globalLatencyCalculator := score.NewAdaptiveMaxCalculator(
		score.DefaultHalfLifeTime,          // halfLife (1 hour default)
		score.AdaptiveP10MinBound,          // minP10 (0.001s = 1ms)
		score.AdaptiveP10MaxBound,          // maxP10 (10s)
		score.DefaultLatencyAdaptiveMinMax, // minMax (P90 lower bound: 1.0s)
		score.DefaultLatencyAdaptiveMaxMax, // maxMax (P90 upper bound: 30.0s)
		score.DefaultTDigestCompression,    // compression (100.0)
	)

	globalSyncCalculator := score.NewAdaptiveMaxCalculator(
		score.DefaultHalfLifeTime,       // halfLife (1 hour default)
		score.AdaptiveSyncP10MinBound,   // minP10 (0.1s)
		score.AdaptiveSyncP10MaxBound,   // maxP10 (60s)
		score.DefaultSyncAdaptiveMinMax, // minMax (P90 lower bound: 30.0s)
		score.DefaultSyncAdaptiveMaxMax, // maxMax (P90 upper bound: 1200.0s)
		score.DefaultTDigestCompression, // compression (100.0)
	)

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
		globalLatencyCalculator:         globalLatencyCalculator,
		globalSyncCalculator:            globalSyncCalculator,
		chainState:                      cState,
		metricsManager:                  metricsManager,
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

// SetDeterministicSeed sets a deterministic seed for the weighted selector
// This is used for testing purposes only to ensure reproducible provider selection
func (po *ProviderOptimizer) SetDeterministicSeed(seed int64) {
	if po.weightedSelector != nil {
		po.weightedSelector.SetDeterministicSeed(seed)
	}
}
