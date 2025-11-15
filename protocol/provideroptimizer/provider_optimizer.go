package provideroptimizer

import (
	"fmt"
	"strings"
	"sync"
	"time"

	stdMath "math"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/dgraph-io/ristretto/v2"
	"github.com/lavanet/lava/v5/protocol/metrics"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/lavaslices"
	"github.com/lavanet/lava/v5/utils/rand"
	"github.com/lavanet/lava/v5/utils/score"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"gonum.org/v1/gonum/mathext"
)

// The provider optimizer is a mechanism within the consumer that is responsible for choosing
// the optimal provider for the consumer.
// The choice depends on the provider's QoS reputation metrics: latency, sync and availability.
// Providers are picked by selection tiers that take into account their stake amount and QoS
// reputation score.

const (
	CacheMaxCost             = 20000 // each item cost would be 1
	CacheNumCounters         = 20000 // expect 2000 items
	DefaultExplorationChance = 0.1
	CostExplorationChance    = 0.01
)

var (
	OptimizerNumTiers = 4 // number of tiers to use
	MinimumEntries    = 5 // minimum number of entries in a tier to be considered for selection
	ATierChance       = 0.75
	LastTierChance    = 0.0
	AutoAdjustTiers   = false
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
	selectionWeighter               SelectionWeighter // weights are the providers stake
	OptimizerNumTiers               int               // number of tiers to use
	OptimizerMinTierEntries         int               // minimum number of entries in a tier to be considered for selection
	OptimizerQoSSelectionEnabled    bool              // enables QoS-based selection within tiers instead of stake-based
	consumerOptimizerQoSClient      consumerOptimizerQoSClientInf
	chainId                         string
}

// The exploration mechanism makes the optimizer return providers that were not talking
// to the consumer for a long time (a couple of seconds). This allows better distribution
// of paired providers by avoiding returning the same best providers over and over.
// The Exploration struct holds a provider address and last QoS metrics update time (ScoreStore)
type Exploration struct {
	address string
	time    time.Time
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

// UpdateWeights update the selection weighter weights
func (po *ProviderOptimizer) UpdateWeights(weights map[string]int64, epoch uint64) {
	po.selectionWeighter.SetWeights(weights)

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

func (po *ProviderOptimizer) CalculateQoSScoresForMetrics(allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) []*metrics.OptimizerQoSReport {
	selectionTier, _, providersScores := po.CalculateSelectionTiers(allAddresses, ignoredProviders, cu, requestedBlock)
	reports := []*metrics.OptimizerQoSReport{}

	rawScores := selectionTier.GetRawScores()
	tierChances := selectionTier.ShiftTierChance(po.OptimizerNumTiers, map[int]float64{0: ATierChance})
	for idx, entry := range rawScores {
		qosReport := providersScores[entry.Address]
		qosReport.EntryIndex = idx
		qosReport.TierChances = PrintTierChances(tierChances)
		qosReport.Tier = po.GetProviderTier(entry.Address, selectionTier)
		reports = append(reports, qosReport)
	}

	return reports
}

func PrintTierChances(tierChances map[int]float64) string {
	var tierChancesString string
	for tier, chance := range tierChances {
		tierChancesString += fmt.Sprintf("%d: %f, ", tier, chance)
	}
	return tierChancesString
}

func (po *ProviderOptimizer) GetProviderTier(providerAddress string, selectionTier SelectionTier) int {
	selectionTierScoresCount := selectionTier.ScoresCount()
	numTiersWanted := po.GetNumTiersWanted(selectionTier, selectionTierScoresCount)
	minTierEntries := po.GetMinTierEntries(selectionTier, selectionTierScoresCount)
	for tier := 0; tier < numTiersWanted; tier++ {
		tierProviders := selectionTier.GetTier(tier, numTiersWanted, minTierEntries)
		for _, provider := range tierProviders {
			if provider.Address == providerAddress {
				return tier
			}
		}
	}
	return -1
}

func (po *ProviderOptimizer) CalculateSelectionTiers(allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) (SelectionTier, Exploration, map[string]*metrics.OptimizerQoSReport) {
	explorationCandidate := Exploration{address: "", time: time.Now().Add(time.Hour)}
	selectionTier := NewSelectionTier()
	providerScores := make(map[string]*metrics.OptimizerQoSReport)
	for _, providerAddress := range allAddresses {
		if _, ok := ignoredProviders[providerAddress]; ok {
			// ignored provider, skip it
			continue
		}

		qos, lastUpdateTime := po.GetReputationReportForProvider(providerAddress)
		if qos == nil {
			utils.LavaFormatWarning("[Optimizer] cannot calculate selection tiers",
				fmt.Errorf("could not get QoS excellece report for provider"),
				utils.LogAttr("provider", providerAddress),
			)
			return NewSelectionTier(), Exploration{}, nil
		}

		utils.LavaFormatTrace("[Optimizer] scores information",
			utils.LogAttr("providerAddress", providerAddress),
			utils.LogAttr("latencyScore", qos.Latency.String()),
			utils.LogAttr("syncScore", qos.Sync.String()),
			utils.LogAttr("availabilityScore", qos.Availability.String()),
		)

		opts := []pairingtypes.Option{pairingtypes.WithStrategyFactor(po.strategy.GetStrategyFactor())}
		if requestedBlock >= 0 {
			providerData, found := po.getProviderData(providerAddress)
			if !found {
				utils.LavaFormatTrace("[Optimizer] could not get provider data, using default", utils.LogAttr("provider", providerAddress))
			}
			// add block error probability config if the request block is positive
			opts = append(opts, pairingtypes.WithBlockErrorProbability(po.CalculateProbabilityOfBlockError(requestedBlock, providerData)))
		} else { // all negative blocks (latest/earliest/pending/safe/finalized) will be considered as latest
			requestedBlock = spectypes.LATEST_BLOCK
		}
		score, err := qos.ComputeReputationFloat64(opts...)
		if err != nil {
			utils.LavaFormatWarning("[Optimizer] cannot calculate selection tiers", err,
				utils.LogAttr("provider", providerAddress),
				utils.LogAttr("qos_report", qos.String()),
			)
			return NewSelectionTier(), Exploration{}, nil
		}
		latency, sync, availability := qos.GetScoresFloat64()
		providerScores[providerAddress] = &metrics.OptimizerQoSReport{
			ProviderAddress:   providerAddress,
			SyncScore:         sync,
			AvailabilityScore: availability,
			LatencyScore:      latency,
			GenericScore:      score,
		}

		selectionTier.AddScore(providerAddress, score)

		// check if candidate for exploration
		if lastUpdateTime.Add(10*time.Second).Before(time.Now()) && lastUpdateTime.Before(explorationCandidate.time) {
			// if the provider didn't update its data for 10 seconds, it is a candidate for exploration
			explorationCandidate = Exploration{address: providerAddress, time: lastUpdateTime}
		}
	}
	return selectionTier, explorationCandidate, providerScores
}

// returns a sub set of selected providers according to their scores, perturbation factor will be added to each score in order to randomly select providers that are not always on top
func (po *ProviderOptimizer) ChooseProvider(allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) (addresses []string, tier int) {
	selectionTier, explorationCandidate, _ := po.CalculateSelectionTiers(allAddresses, ignoredProviders, cu, requestedBlock) // spliting to tiers by score
	selectionTierScoresCount := selectionTier.ScoresCount()                                                                  // length of the selectionTier
	localMinimumEntries := po.GetMinTierEntries(selectionTier, selectionTierScoresCount)

	if selectionTierScoresCount == 0 {
		// no providers to choose from
		return []string{}, -1
	}
	initialChances := map[int]float64{0: ATierChance}

	numberOfTiersWanted := po.GetNumTiersWanted(selectionTier, selectionTierScoresCount)
	if selectionTierScoresCount >= localMinimumEntries*2 {
		// if we have more than 2*localMinimumEntries we set the LastTierChance configured
		initialChances[(numberOfTiersWanted - 1)] = LastTierChance
	}
	shiftedChances := selectionTier.ShiftTierChance(numberOfTiersWanted, initialChances)
	tier = selectionTier.SelectTierRandomly(numberOfTiersWanted, shiftedChances)
	// Get tier inputs, what tier, how many tiers we have, and how many providers are in each tier
	tierProviders := selectionTier.GetTier(tier, numberOfTiersWanted, localMinimumEntries)
	// TODO: add penalty if a provider is chosen too much
	var selectedProvider string
	if po.OptimizerQoSSelectionEnabled {
		selectedProvider = po.selectionWeighter.WeightedChoiceByQoS(tierProviders)
	} else {
		selectedProvider = po.selectionWeighter.WeightedChoice(tierProviders)
	}
	returnedProviders := []string{selectedProvider}
	if explorationCandidate.address != "" && explorationCandidate.address != selectedProvider && po.shouldExplore(1) {
		returnedProviders = append(returnedProviders, explorationCandidate.address)
	}
	utils.LavaFormatTrace("[Optimizer] returned providers",
		utils.LogAttr("providers", strings.Join(returnedProviders, ",")),
		utils.LogAttr("shiftedChances", shiftedChances),
		utils.LogAttr("tier", tier),
	)

	return returnedProviders, tier
}

// returns a sub set of selected providers according to their scores, perturbation factor will be added to each score in order to randomly select providers that are not always on top
func (po *ProviderOptimizer) ChooseProviderFromTopTier(allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) (addresses []string) {
	selectionTier, _, _ := po.CalculateSelectionTiers(allAddresses, ignoredProviders, cu, requestedBlock)
	selectionTierScoresCount := selectionTier.ScoresCount()
	numberOfTiersWanted := po.GetNumTiersWanted(selectionTier, selectionTierScoresCount)
	localMinimumEntries := po.GetMinTierEntries(selectionTier, selectionTierScoresCount)

	// Get tier inputs, what tier, how many tiers we have, and how many providers are in each tier
	tierProviders := selectionTier.GetTier(0, numberOfTiersWanted, localMinimumEntries)
	// TODO: add penalty if a provider is chosen too much
	var selectedProvider string
	if po.OptimizerQoSSelectionEnabled {
		selectedProvider = po.selectionWeighter.WeightedChoiceByQoS(tierProviders)
	} else {
		selectedProvider = po.selectionWeighter.WeightedChoice(tierProviders)
	}
	returnedProviders := []string{selectedProvider}

	utils.LavaFormatTrace("[Optimizer] returned top tier provider",
		utils.LogAttr("providers", strings.Join(returnedProviders, ",")),
	)

	return returnedProviders
}

// GetMinTierEntries gets minimum number of entries in a tier to be considered for selection
// if AutoAdjustTiers global is true, the number of providers per tier is divided equally
// between them
func (po *ProviderOptimizer) GetMinTierEntries(selectionTier SelectionTier, selectionTierScoresCount int) int {
	localMinimumEntries := po.OptimizerMinTierEntries
	if AutoAdjustTiers {
		adjustedProvidersPerTier := int(stdMath.Ceil(float64(selectionTierScoresCount) / float64(po.OptimizerNumTiers)))
		if localMinimumEntries > adjustedProvidersPerTier {
			utils.LavaFormatTrace("optimizer AutoAdjustTiers activated",
				utils.LogAttr("set_to_adjustedProvidersPerTier", adjustedProvidersPerTier),
				utils.LogAttr("was_MinimumEntries", po.OptimizerMinTierEntries),
				utils.LogAttr("tiers_count_po.OptimizerNumTiers", po.OptimizerNumTiers),
				utils.LogAttr("selectionTierScoresCount", selectionTierScoresCount))
			localMinimumEntries = adjustedProvidersPerTier
		}
	}
	return localMinimumEntries
}

// GetNumTiersWanted returns the number of tiers wanted
// if we have enough providers to create the tiers return the configured number of tiers wanted
// if not, set the number of tiers to the number of providers we currently have
func (po *ProviderOptimizer) GetNumTiersWanted(selectionTier SelectionTier, selectionTierScoresCount int) int {
	numberOfTiersWanted := po.OptimizerNumTiers
	if selectionTierScoresCount < po.OptimizerNumTiers {
		numberOfTiersWanted = selectionTierScoresCount
	}

	return numberOfTiersWanted
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
	return &ProviderOptimizer{
		strategy:                        strategy,
		providersStorage:                cache,
		averageBlockTime:                averageBlockTIme,
		providerRelayStats:              relayCache,
		wantedNumProvidersInConcurrency: wantedNumProvidersInConcurrency,
		selectionWeighter:               NewSelectionWeighter(),
		OptimizerNumTiers:               OptimizerNumTiers,
		OptimizerMinTierEntries:         MinimumEntries,
		OptimizerQoSSelectionEnabled:    qosSelectionEnabled,
		consumerOptimizerQoSClient:      consumerOptimizerQoSClient,
		chainId:                         chainId,
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
