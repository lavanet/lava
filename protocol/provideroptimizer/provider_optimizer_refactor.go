package provideroptimizer

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dgraph-io/ristretto"
	"github.com/lavanet/lava/v4/utils"
	"github.com/lavanet/lava/v4/utils/lavaslices"
	"github.com/lavanet/lava/v4/utils/rand"
	"github.com/lavanet/lava/v4/utils/score"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
)

// The provider optimizer is a mechanism within the consumer that is responsible for choosing
// the optimal provider for the consumer.
// The choice depends on the provider's QoS excellence metrics: latency, sync and availability.
// Providers are picked by selection tiers that take into account their stake amount and QoS
// excellence score.

const (
	CacheMaxCost_Refactor             = 20000 // each item cost would be 1
	CacheNumCounters_Refactor         = 20000 // expect 2000 items
	DefaultExplorationChance_Refactor = 0.1
	CostExplorationChance_Refactor    = 0.01
)

var (
	OptimizerNumTiers_Refactor = 4
	MinimumEntries_Refactor    = 5
	ATierChance_Refactor       = 0.75
	LastTierChance_Refactor    = 0.0
)

type ConcurrentBlockStore_Refactor struct {
	Lock  sync.Mutex
	Time  time.Time
	Block uint64
}

type cacheInf_Refactor interface {
	Get(key interface{}) (interface{}, bool)
	Set(key, value interface{}, cost int64) bool
}

type consumerOptimizerQoSClientInf_Refactor interface {
	UpdatePairingListStake(stakeMap map[string]int64, chainId string, epoch uint64)
}

type ProviderOptimizer_Refactor struct {
	strategy                        Strategy_Refactor
	providersStorage                cacheInf_Refactor
	providerRelayStats              *ristretto.Cache // used to decide on the half time of the decay
	averageBlockTime                time.Duration
	wantedNumProvidersInConcurrency uint
	latestSyncData                  ConcurrentBlockStore_Refactor
	selectionWeighter               SelectionWeighter // weights are the providers stake
	OptimizerNumTiers               int               // number of tiers to use
	OptimizerMinTierEntries         int               // minimum number of entries in a tier to be considered for selection
	consumerOptimizerQoSClient      consumerOptimizerQoSClientInf_Refactor
	chainId                         string
}

// The exploration mechanism makes the optimizer return providers that were not talking
// to the consumer for a long time (a couple of seconds). This allows better distribution
// of paired providers by avoiding returning the same best providers over and over.
// The Exploration struct holds a provider address and last QoS metrics update time (ScoreStore)
type Exploration_Refactor struct {
	address string
	time    time.Time
}

type ProviderData_Refactor struct {
	Availability score.ScoreStorer_Refactor // will be used to calculate the probability of error
	Latency      score.ScoreStorer_Refactor // will be used to calculate the latency score
	Sync         score.ScoreStorer_Refactor // will be used to calculate the sync score for spectypes.LATEST_BLOCK/spectypes.NOT_APPLICABLE requests
	SyncBlock    uint64                     // will be used to calculate the probability of block error
}

// Strategy_Refactor defines the pairing strategy. Using different
// strategies allow users to determine the providers type they'll
// be paired with: providers with low latency, fresh sync and more.
type Strategy_Refactor int

const (
	StrategyBalanced_Refactor      Strategy_Refactor = iota
	StrategyLatency_Refactor                         // prefer low latency
	StrategySyncFreshness_Refactor                   // prefer better sync
	StrategyCost_Refactor                            // prefer low CU cost (minimize optimizer exploration)
	StrategyPrivacy_Refactor                         // prefer pairing with a single provider (not fully implemented)
	StrategyAccuracy_Refactor                        // encourage optimizer exploration (higher cost)
	StrategyDistributed_Refactor                     // prefer pairing with different providers (slightly minimize optimizer exploration)
)

func (s Strategy_Refactor) String() string {
	switch s {
	case StrategyBalanced_Refactor:
		return "balanced"
	case StrategyLatency_Refactor:
		return "latency"
	case StrategySyncFreshness_Refactor:
		return "sync_freshness"
	case StrategyCost_Refactor:
		return "cost"
	case StrategyPrivacy_Refactor:
		return "privacy"
	case StrategyAccuracy_Refactor:
		return "accuracy"
	case StrategyDistributed_Refactor:
		return "distributed"
	}

	return ""
}

// GetStrategyFactor gets the appropriate factor to multiply the sync factor
// with according to the strategy
func (s Strategy_Refactor) GetStrategyFactor() math.LegacyDec {
	switch s {
	case StrategyLatency_Refactor:
		return pairingtypes.LatencyStrategyFactor
	case StrategySyncFreshness_Refactor:
		return pairingtypes.SyncFreshnessStrategyFactor
	}

	return pairingtypes.BalancedStrategyFactor
}

// UpdateWeights update the selection weighter weights
func (po *ProviderOptimizer_Refactor) UpdateWeights_Refactor(weights map[string]int64, epoch uint64) {
	po.selectionWeighter.SetWeights(weights)

	// Update the stake map for metrics
	if po.consumerOptimizerQoSClient != nil {
		po.consumerOptimizerQoSClient.UpdatePairingListStake(weights, po.chainId, epoch)
	}
}

// AppendRelayFailure updates a provider's QoS metrics for a failed relay
func (po *ProviderOptimizer_Refactor) AppendRelayFailure_Refactor(provider string) {
	po.appendRelayData_Refactor(provider, 0, false, 0, 0, time.Now())
}

// AppendRelayData updates a provider's QoS metrics for a successful relay
func (po *ProviderOptimizer_Refactor) AppendRelayData_Refactor(provider string, latency time.Duration, cu, syncBlock uint64) {
	po.appendRelayData_Refactor(provider, latency, true, cu, syncBlock, time.Now())
}

// appendRelayData gets three new QoS metrics samples and updates the provider's metrics using a decaying weighted average
func (po *ProviderOptimizer_Refactor) appendRelayData_Refactor(provider string, latency time.Duration, success bool, cu, syncBlock uint64, sampleTime time.Time) {
	latestSync, timeSync := po.updateLatestSyncData_Refactor(syncBlock, sampleTime)
	providerData, _ := po.getProviderData_Refactor(provider)
	halfTime := po.calculateHalfTime_Refactor(provider, sampleTime)
	weight := score.RelayUpdateWeight_Refactor

	if success {
		// on a successful relay, update all the QoS metrics
		providerData = po.updateDecayingWeightedAverage_Refactor(providerData, score.AvailabilityScoreType_Refactor, 1, weight, halfTime, cu, sampleTime)
		providerData = po.updateDecayingWeightedAverage_Refactor(providerData, score.LatencyScoreType_Refactor, latency.Seconds(), weight, halfTime, cu, sampleTime)

		if syncBlock > providerData.SyncBlock {
			// do not allow providers to go back
			providerData.SyncBlock = syncBlock
		}
		syncLag := po.calculateSyncLag_Refactor(latestSync, timeSync, providerData.SyncBlock, sampleTime)
		providerData = po.updateDecayingWeightedAverage_Refactor(providerData, score.SyncScoreType_Refactor, syncLag.Seconds(), weight, halfTime, cu, sampleTime)
	} else {
		// on a failed relay, update the availability metric with a failure score
		providerData = po.updateDecayingWeightedAverage_Refactor(providerData, score.AvailabilityScoreType_Refactor, 0, weight, halfTime, cu, sampleTime)
	}

	po.providersStorage.Set(provider, providerData, 1)
	po.updateRelayTime_Refactor(provider, sampleTime)

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
func (po *ProviderOptimizer_Refactor) AppendProbeRelayData_Refactor(providerAddress string, latency time.Duration, success bool) {
	providerData, _ := po.getProviderData_Refactor(providerAddress)
	sampleTime := time.Now()
	halfTime := po.calculateHalfTime_Refactor(providerAddress, sampleTime)
	weight := score.ProbeUpdateWeight_Refactor

	if success {
		// update latency only on success
		providerData = po.updateDecayingWeightedAverage_Refactor(providerData, score.AvailabilityScoreType_Refactor, 1, weight, halfTime, 0, sampleTime)
		providerData = po.updateDecayingWeightedAverage_Refactor(providerData, score.LatencyScoreType_Refactor, latency.Seconds(), weight, halfTime, 0, sampleTime)
	} else {
		providerData = po.updateDecayingWeightedAverage_Refactor(providerData, score.AvailabilityScoreType_Refactor, 0, weight, halfTime, 0, sampleTime)
	}
	po.providersStorage.Set(providerAddress, providerData, 1)

	utils.LavaFormatTrace("[Optimizer] probe update",
		utils.LogAttr("providerAddress", providerAddress),
		utils.LogAttr("latency", latency),
		utils.LogAttr("success", success),
	)
}

func (po *ProviderOptimizer_Refactor) CalculateSelectionTiers_Refactor(allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) (SelectionTier, Exploration_Refactor) {
	explorationCandidate := Exploration_Refactor{address: "", time: time.Now().Add(time.Hour)}
	selectionTier := NewSelectionTier()
	for _, providerAddress := range allAddresses {
		if _, ok := ignoredProviders[providerAddress]; ok {
			// ignored provider, skip it
			continue
		}

		providerData, found := po.getProviderData_Refactor(providerAddress)
		if !found {
			utils.LavaFormatTrace("[Optimizer] could not get provider data, using default", utils.LogAttr("provider", providerAddress))
		}

		qos, lastUpdateTime := po.GetExcellenceQoSReportForProvider_Refactor(providerData, providerAddress)
		if qos == nil {
			utils.LavaFormatWarning("[Optimizer] cannot calculate selection tiers",
				fmt.Errorf("could not get QoS excellece report for provider"),
				utils.LogAttr("provider", providerAddress),
			)
			return NewSelectionTier(), Exploration_Refactor{}
		}

		utils.LavaFormatTrace("[Optimizer] scores information",
			utils.LogAttr("providerAddress", providerAddress),
			utils.LogAttr("latencyScore", qos.Latency.String()),
			utils.LogAttr("syncScore", qos.Sync.String()),
			utils.LogAttr("availabilityScore", qos.Availability.String()),
		)

		opts := []pairingtypes.Option{pairingtypes.WithStrategyFactor(po.strategy.GetStrategyFactor())}
		if requestedBlock > 0 {
			// add block error probability config if the request block is positive
			opts = append(opts, pairingtypes.WithBlockErrorProbability(po.CalculateProbabilityOfBlockError(requestedBlock, providerData)))
		} else if requestedBlock != spectypes.LATEST_BLOCK && requestedBlock != spectypes.NOT_APPLICABLE {
			// if the request block is not positive but not latest/not-applicable - return an error
			utils.LavaFormatWarning("[Optimizer] cannot calculate selection tiers",
				fmt.Errorf("could not configure block error probability, invalid requested block (must be >0 or -1 or -2)"),
				utils.LogAttr("provider", providerAddress),
				utils.LogAttr("requested_block", requestedBlock),
			)
			return NewSelectionTier(), Exploration_Refactor{}
		}
		score, err := qos.ComputeQoSExcellence_Refactor(opts...)
		if err != nil {
			utils.LavaFormatWarning("[Optimizer] cannot calculate selection tiers", err,
				utils.LogAttr("provider", providerAddress),
				utils.LogAttr("qos_report", qos.String()),
			)
			return NewSelectionTier(), Exploration_Refactor{}
		}
		selectionTier.AddScore(providerAddress, score.MustFloat64())

		// check if candidate for exploration
		if lastUpdateTime.Add(10*time.Second).Before(time.Now()) && lastUpdateTime.Before(explorationCandidate.time) {
			// if the provider didn't update its data for 10 seconds, it is a candidate for exploration
			explorationCandidate = Exploration_Refactor{address: providerAddress, time: lastUpdateTime}
		}
	}
	return selectionTier, explorationCandidate
}

// returns a sub set of selected providers according to their scores, perturbation factor will be added to each score in order to randomly select providers that are not always on top
func (po *ProviderOptimizer_Refactor) ChooseProvider_Refactor(allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) (addresses []string, tier int) {
	selectionTier, explorationCandidate := po.CalculateSelectionTiers_Refactor(allAddresses, ignoredProviders, cu, requestedBlock)
	if selectionTier.ScoresCount() == 0 {
		// no providers to choose from
		return []string{}, -1
	}
	initialChances := map[int]float64{0: ATierChance}
	numTiers := po.OptimizerNumTiers
	if selectionTier.ScoresCount() < numTiers {
		numTiers = selectionTier.ScoresCount()
	}
	if selectionTier.ScoresCount() >= po.OptimizerMinTierEntries*2 {
		// if we have more than 2*MinimumEntries we set the LastTierChance configured
		initialChances[(po.OptimizerNumTiers - 1)] = LastTierChance
	}
	shiftedChances := selectionTier.ShiftTierChance(numTiers, initialChances)
	tier = selectionTier.SelectTierRandomly(numTiers, shiftedChances)
	tierProviders := selectionTier.GetTier(tier, numTiers, po.OptimizerMinTierEntries)
	// TODO: add penalty if a provider is chosen too much
	selectedProvider := po.selectionWeighter.WeightedChoice(tierProviders)
	returnedProviders := []string{selectedProvider}
	if explorationCandidate.address != "" && po.shouldExplore_Refactor(1) {
		returnedProviders = append(returnedProviders, explorationCandidate.address)
	}
	utils.LavaFormatTrace("[Optimizer] returned providers",
		utils.LogAttr("providers", strings.Join(returnedProviders, ",")),
		utils.LogAttr("shiftedChances", shiftedChances),
		utils.LogAttr("tier", tier),
	)

	return returnedProviders, tier
}

// CalculateProbabilityOfBlockError calculates the probability that a provider doesn't a specific requested
// block when the consumer asks the optimizer to fetch a provider with the specific block
func (po *ProviderOptimizer_Refactor) CalculateProbabilityOfBlockError(requestedBlock int64, providerData ProviderData_Refactor) sdk.Dec {
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

// calculate the expected average time until this provider catches up with the given latestSync block
// for the first block difference we take the minimum between the time passed since block arrived and the average block time
// for any other block we take the averageBlockTime
func (po *ProviderOptimizer_Refactor) calculateSyncLag_Refactor(latestSync uint64, timeSync time.Time, providerBlock uint64, sampleTime time.Time) time.Duration {
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

func (po *ProviderOptimizer_Refactor) updateLatestSyncData_Refactor(providerLatestBlock uint64, sampleTime time.Time) (uint64, time.Time) {
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
func (po *ProviderOptimizer_Refactor) shouldExplore_Refactor(currentNumProviders int) bool {
	if uint(currentNumProviders) >= po.wantedNumProvidersInConcurrency {
		return false
	}
	explorationChance := DefaultExplorationChance_Refactor
	switch po.strategy {
	case StrategyLatency_Refactor:
		return true // we want a lot of parallel tries on latency
	case StrategyAccuracy_Refactor:
		return true
	case StrategyCost_Refactor:
		explorationChance = CostExplorationChance_Refactor
	case StrategyDistributed_Refactor:
		explorationChance = DefaultExplorationChance_Refactor * 0.25
	case StrategyPrivacy_Refactor:
		return false // only one at a time
	}
	return rand.Float64() < explorationChance
}

// getProviderData gets a specific proivder's QoS data. If it doesn't exist, it returns a default provider data struct
func (po *ProviderOptimizer_Refactor) getProviderData_Refactor(providerAddress string) (providerData ProviderData_Refactor, found bool) {
	storedVal, found := po.providersStorage.Get(providerAddress)
	if found {
		var ok bool

		providerData, ok = storedVal.(ProviderData_Refactor)
		if !ok {
			utils.LavaFormatFatal("invalid usage of optimizer provider storage", nil, utils.Attribute{Key: "storedVal", Value: storedVal})
		}
	} else {
		providerData = ProviderData_Refactor{
			Availability: score.NewScoreStore_Refactor(score.AvailabilityScoreType_Refactor), // default score of 100%
			Latency:      score.NewScoreStore_Refactor(score.LatencyScoreType_Refactor),      // default score of 10ms (encourage exploration)
			Sync:         score.NewScoreStore_Refactor(score.SyncScoreType_Refactor),         // default score of 100ms (encourage exploration)
			SyncBlock:    0,
		}
	}

	return providerData, found
}

// updateDecayingWeightedAverage updates a provider's QoS metric ScoreStore with a new sample
func (po *ProviderOptimizer_Refactor) updateDecayingWeightedAverage_Refactor(providerData ProviderData_Refactor, scoreType string, sample float64, weight float64, halfTime time.Duration, cu uint64, sampleTime time.Time) ProviderData_Refactor {
	switch scoreType {
	case score.LatencyScoreType_Refactor:
		err := providerData.Latency.UpdateConfig(
			score.WithWeight(weight),
			score.WithDecayHalfLife(halfTime),
			score.WithLatencyCuFactor(score.GetLatencyFactor(cu)),
		)
		if err != nil {
			utils.LavaFormatError("did not update provider latency score", err)
			return providerData
		}
		err = providerData.Latency.Update(sample, sampleTime)
		if err != nil {
			utils.LavaFormatError("did not update provider latency score", err)
			return providerData
		}

	case score.SyncScoreType_Refactor:
		err := providerData.Sync.UpdateConfig(score.WithWeight(weight), score.WithDecayHalfLife(halfTime))
		if err != nil {
			utils.LavaFormatError("did not update provider sync score", err)
			return providerData
		}
		err = providerData.Sync.Update(sample, sampleTime)
		if err != nil {
			utils.LavaFormatError("did not update provider sync score", err)
			return providerData
		}

	case score.AvailabilityScoreType_Refactor:
		err := providerData.Availability.UpdateConfig(score.WithWeight(weight), score.WithDecayHalfLife(halfTime))
		if err != nil {
			utils.LavaFormatError("did not update provider availability score", err)
			return providerData
		}
		err = providerData.Availability.Update(sample, sampleTime)
		if err != nil {
			utils.LavaFormatError("did not update provider availability score", err)
			return providerData
		}
	}

	return providerData
}

// updateRelayTime adds a relay sample time to a provider's data
func (po *ProviderOptimizer_Refactor) updateRelayTime_Refactor(providerAddress string, sampleTime time.Time) {
	times := po.getRelayStatsTimes_Refactor(providerAddress)
	if len(times) == 0 {
		po.providerRelayStats.Set(providerAddress, []time.Time{sampleTime}, 1)
		return
	}
	times = append(times, sampleTime)
	po.providerRelayStats.Set(providerAddress, times, 1)
}

// calculateHalfTime calculates a provider's half life time for a relay sampled in sampleTime
func (po *ProviderOptimizer_Refactor) calculateHalfTime_Refactor(providerAddress string, sampleTime time.Time) time.Duration {
	halfTime := score.DefaultHalfLifeTime_Refactor
	relaysHalfTime := po.getRelayStatsTimeDiff_Refactor(providerAddress, sampleTime)
	if relaysHalfTime > halfTime {
		halfTime = relaysHalfTime
	}
	if halfTime > score.MaxHalfTime_Refactor {
		halfTime = score.MaxHalfTime_Refactor
	}
	return halfTime
}

// getRelayStatsTimeDiff returns the time passed since the provider optimizer's saved relay times median
func (po *ProviderOptimizer_Refactor) getRelayStatsTimeDiff_Refactor(providerAddress string, sampleTime time.Time) time.Duration {
	times := po.getRelayStatsTimes_Refactor(providerAddress)
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

func (po *ProviderOptimizer_Refactor) getRelayStatsTimes_Refactor(providerAddress string) []time.Time {
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

func NewProviderOptimizer_Refactor(strategy Strategy_Refactor, averageBlockTIme time.Duration, wantedNumProvidersInConcurrency uint, consumerOptimizerQoSClient consumerOptimizerQoSClientInf_Refactor, chainId string) *ProviderOptimizer_Refactor {
	cache, err := ristretto.NewCache(&ristretto.Config{NumCounters: CacheNumCounters, MaxCost: CacheMaxCost, BufferItems: 64, IgnoreInternalCost: true})
	if err != nil {
		utils.LavaFormatFatal("failed setting up cache for queries", err)
	}
	relayCache, err := ristretto.NewCache(&ristretto.Config{NumCounters: CacheNumCounters, MaxCost: CacheMaxCost, BufferItems: 64, IgnoreInternalCost: true})
	if err != nil {
		utils.LavaFormatFatal("failed setting up cache for queries", err)
	}
	if strategy == StrategyPrivacy_Refactor {
		// overwrite
		wantedNumProvidersInConcurrency = 1
	}
	return &ProviderOptimizer_Refactor{
		strategy:                        strategy,
		providersStorage:                cache,
		averageBlockTime:                averageBlockTIme,
		providerRelayStats:              relayCache,
		wantedNumProvidersInConcurrency: wantedNumProvidersInConcurrency,
		selectionWeighter:               NewSelectionWeighter(),
		OptimizerNumTiers:               OptimizerNumTiers_Refactor,
		OptimizerMinTierEntries:         MinimumEntries_Refactor,
		consumerOptimizerQoSClient:      consumerOptimizerQoSClient,
		chainId:                         chainId,
	}
}

func (po *ProviderOptimizer_Refactor) GetExcellenceQoSReportForProvider_Refactor(providerData ProviderData_Refactor, providerAddress string) (report *pairingtypes.QualityOfServiceReport, lastUpdateTime time.Time) {
	latency, err := providerData.Latency.Resolve()
	if err != nil {
		utils.LavaFormatError("could not resolve latency score", err, utils.LogAttr("address", providerAddress))
		return nil, time.Time{}
	}
	if latency > score.WorstLatencyScore_Refactor {
		latency = score.WorstLatencyScore_Refactor
	}

	sync, err := providerData.Sync.Resolve()
	if err != nil {
		utils.LavaFormatError("could not resolve sync score", err, utils.LogAttr("address", providerAddress))
		return nil, time.Time{}
	}
	if sync == 0 {
		// if our sync score is uninitialized due to lack of providers
		sync = 1
	} else if sync > score.WorstSyncScore_Refactor {
		sync = score.WorstSyncScore_Refactor
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
