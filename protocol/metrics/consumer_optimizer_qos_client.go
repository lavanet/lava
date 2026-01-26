package metrics

import (
	"context"
	"math"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/goccy/go-json"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/rand"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"golang.org/x/exp/maps"
)

// sanitizeFloat returns 0 if the value is NaN or Inf, otherwise returns the value
func sanitizeFloat(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return v
}

var (
	OptimizerQosServerPushInterval     time.Duration
	OptimizerQosServerSamplingInterval time.Duration
)

type ConsumerOptimizerQoSClient struct {
	consumerHostname string
	consumerAddress  string
	queueSender      *QueueSender
	optimizers       map[string]OptimizerInf // keys are chain ids
	// keys are chain ids, values are maps with provider addresses as keys
	chainIdToProviderToRelaysCount     map[string]map[string]uint64
	chainIdToProviderToNodeErrorsCount map[string]map[string]uint64
	chainIdToProviderToSelectionCount  map[string]map[string]uint64           // tracks how many times each provider was selected
	chainIdToProviderToQoSScoreSum     map[string]map[string]float64          // sum of QoS scores at selection time for averaging
	chainIdToProviderToLastRNGValue    map[string]map[string]float64          // last RNG value used for selection
	chainIdToProviderToEpochToStake    map[string]map[string]map[uint64]int64 // third key is epoch
	chainIdToTotalSelections           map[string]uint64                      // total selections per chain for calculating selection rate
	currentEpoch                       atomic.Uint64
	lock                               sync.RWMutex
	reportsToSend                      []OptimizerQoSReportToSend
	geoLocation                        uint64
}

type OptimizerQoSReport struct {
	ProviderAddress string
	// Legacy fields - Raw EWMA values from score stores (NOT normalized for WRS)
	SyncScore         float64 // Raw sync lag in seconds from EWMA (lower is better)
	AvailabilityScore float64 // Raw availability from EWMA (0-1, higher is better)
	LatencyScore      float64 // Raw latency in seconds from EWMA (lower is better)
	GenericScore      float64 // Old composite score (deprecated, use SelectionComposite)
	EntryIndex        int     // Index in provider list
	// WRS normalized scores - Used in weighted random selection algorithm (0-1, higher is better)
	SelectionAvailability float64 // Normalized availability after Phase 1 rescaling
	SelectionLatency      float64 // Normalized latency after Phase 2 P10-P90
	SelectionSync         float64 // Normalized sync after Phase 2 P10-P90
	SelectionStake        float64 // Normalized stake after square root scaling
	SelectionComposite    float64 // Final composite score used for selection
	// Weighted contributions - How much each parameter contributes to SelectionComposite
	AvailabilityContribution float64 // SelectionAvailability × availability_weight
	LatencyContribution      float64 // SelectionLatency × latency_weight
	SyncContribution         float64 // SelectionSync × sync_weight
	StakeContribution        float64 // SelectionStake × stake_weight
}

type OptimizerQoSReportToSend struct {
	Timestamp time.Time `json:"timestamp"`
	// Legacy fields - Raw EWMA values from score stores (NOT normalized for WRS)
	SyncScore         float64 `json:"sync_score"`         // Raw sync lag in seconds from EWMA (lower is better)
	AvailabilityScore float64 `json:"availability_score"` // Raw availability from EWMA (0-1, higher is better)
	LatencyScore      float64 `json:"latency_score"`      // Raw latency in seconds from EWMA (lower is better)
	GenericScore      float64 `json:"generic_score"`      // Old composite score (deprecated, use selection_composite)
	// Provider metadata
	ProviderAddress  string  `json:"provider"`
	ConsumerHostname string  `json:"consumer_hostname"`
	ConsumerAddress  string  `json:"consumer_pub_address"`
	ChainId          string  `json:"chain_id"`
	NodeErrorRate    float64 `json:"node_error_rate"`
	Epoch            uint64  `json:"epoch"`
	ProviderStake    int64   `json:"provider_stake"`
	EntryIndex       int     `json:"entry_index"`
	GeoLocation      uint64  `json:"geo_location"`
	// WRS normalized scores - Used in weighted random selection algorithm (0-1, higher is better)
	SelectionAvailability float64 `json:"selection_availability"` // Normalized availability after Phase 1 rescaling
	SelectionLatency      float64 `json:"selection_latency"`      // Normalized latency after Phase 2 P10-P90
	SelectionSync         float64 `json:"selection_sync"`         // Normalized sync after Phase 2 P10-P90
	SelectionStake        float64 `json:"selection_stake"`        // Normalized stake after square root scaling
	SelectionComposite    float64 `json:"selection_composite"`    // Final composite score used for selection
	// Provider selection tracking
	SelectionCount    uint64  `json:"selection_count"`     // Number of times this provider was selected
	SelectionRate     float64 `json:"selection_rate"`      // Percentage of total selections (0-1)
	SelectionQoSScore float64 `json:"selection_qos_score"` // Average QoS score at time of selection (0-1)
	SelectionRNGValue float64 `json:"selection_rng_value"` // Last RNG value used for selecting this provider
	// Weighted contributions - How much each parameter contributes to selection_composite
	AvailabilityContribution float64 `json:"availability_contribution"` // selection_availability × availability_weight
	LatencyContribution      float64 `json:"latency_contribution"`      // selection_latency × latency_weight
	SyncContribution         float64 `json:"sync_contribution"`         // selection_sync × sync_weight
	StakeContribution        float64 `json:"stake_contribution"`        // selection_stake × stake_weight
}

func (oqosr OptimizerQoSReportToSend) String() string {
	bytes, err := json.Marshal(oqosr)
	if err != nil {
		return ""
	}
	return string(bytes)
}

type OptimizerInf interface {
	CalculateQoSScoresForMetrics(allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) []*OptimizerQoSReport
}

func NewConsumerOptimizerQoSClient(consumerAddress, endpointAddress string, geoLocation uint64, interval ...time.Duration) *ConsumerOptimizerQoSClient {
	hostname, err := os.Hostname()
	if err != nil {
		utils.LavaFormatWarning("Error while getting hostname for ConsumerOptimizerQoSClient", err)
		hostname = "unknown" + strconv.FormatUint(rand.Uint64(), 10) // random seed for different unknowns
	}
	return &ConsumerOptimizerQoSClient{
		consumerHostname:                   hostname,
		consumerAddress:                    consumerAddress,
		queueSender:                        NewQueueSender(endpointAddress, "ConsumerOptimizerQoS", nil, interval...),
		optimizers:                         map[string]OptimizerInf{},
		chainIdToProviderToRelaysCount:     map[string]map[string]uint64{},
		chainIdToProviderToNodeErrorsCount: map[string]map[string]uint64{},
		chainIdToProviderToSelectionCount:  map[string]map[string]uint64{},
		chainIdToProviderToQoSScoreSum:     map[string]map[string]float64{},
		chainIdToProviderToLastRNGValue:    map[string]map[string]float64{},
		chainIdToProviderToEpochToStake:    map[string]map[string]map[uint64]int64{},
		chainIdToTotalSelections:           map[string]uint64{},
		geoLocation:                        geoLocation,
	}
}

func (coqc *ConsumerOptimizerQoSClient) getProviderChainMapCounterValue(counterStore map[string]map[string]uint64, chainId, providerAddress string) uint64 {
	// must be called under read lock
	if counterProvidersMap, found := counterStore[chainId]; found {
		return counterProvidersMap[providerAddress]
	}
	return 0
}

func (coqc *ConsumerOptimizerQoSClient) getProviderChainRelaysCount(chainId, providerAddress string) uint64 {
	// must be called under read lock
	return coqc.getProviderChainMapCounterValue(coqc.chainIdToProviderToRelaysCount, chainId, providerAddress)
}

func (coqc *ConsumerOptimizerQoSClient) getProviderChainNodeErrorsCount(chainId, providerAddress string) uint64 {
	// must be called under read lock
	return coqc.getProviderChainMapCounterValue(coqc.chainIdToProviderToNodeErrorsCount, chainId, providerAddress)
}

func (coqc *ConsumerOptimizerQoSClient) getProviderChainStake(chainId, providerAddress string, epoch uint64) int64 {
	// must be called under read lock
	if providersMap, found := coqc.chainIdToProviderToEpochToStake[chainId]; found {
		if epochMap, found := providersMap[providerAddress]; found {
			if stake, found := epochMap[epoch]; found {
				return stake
			}
		}
	}
	return 0
}

func (coqc *ConsumerOptimizerQoSClient) calculateNodeErrorRate(chainId, providerAddress string) float64 {
	// must be called under read lock
	relaysCount := coqc.getProviderChainRelaysCount(chainId, providerAddress)
	if relaysCount > 0 {
		errorsCount := coqc.getProviderChainNodeErrorsCount(chainId, providerAddress)
		return float64(errorsCount) / float64(relaysCount)
	}

	return 0
}

func (coqc *ConsumerOptimizerQoSClient) getProviderChainSelectionCount(chainId, providerAddress string) uint64 {
	// must be called under read lock
	return coqc.getProviderChainMapCounterValue(coqc.chainIdToProviderToSelectionCount, chainId, providerAddress)
}

func (coqc *ConsumerOptimizerQoSClient) calculateSelectionRate(chainId, providerAddress string) float64 {
	// must be called under read lock
	totalSelections := coqc.chainIdToTotalSelections[chainId]
	if totalSelections > 0 {
		selectionCount := coqc.getProviderChainSelectionCount(chainId, providerAddress)
		return float64(selectionCount) / float64(totalSelections)
	}
	return 0
}

func (coqc *ConsumerOptimizerQoSClient) calculateAverageSelectionQoSScore(chainId, providerAddress string) float64 {
	// must be called under read lock
	selectionCount := coqc.getProviderChainSelectionCount(chainId, providerAddress)
	if selectionCount > 0 {
		if providersMap, found := coqc.chainIdToProviderToQoSScoreSum[chainId]; found {
			if scoreSum, found := providersMap[providerAddress]; found {
				return scoreSum / float64(selectionCount)
			}
		}
	}
	return 0
}

func (coqc *ConsumerOptimizerQoSClient) getLastRNGValue(chainId, providerAddress string) float64 {
	// must be called under read lock
	if providersMap, found := coqc.chainIdToProviderToLastRNGValue[chainId]; found {
		if rngValue, found := providersMap[providerAddress]; found {
			return rngValue
		}
	}
	return 0
}

func (coqc *ConsumerOptimizerQoSClient) appendOptimizerQoSReport(report *OptimizerQoSReport, chainId string, epoch uint64) OptimizerQoSReportToSend {
	// must be called under read lock
	// Use sanitizeFloat to prevent NaN/Inf values in JSON output
	optimizerQoSReportToSend := OptimizerQoSReportToSend{
		Timestamp:         time.Now(),
		ConsumerHostname:  coqc.consumerHostname,
		ConsumerAddress:   coqc.consumerAddress,
		SyncScore:         sanitizeFloat(report.SyncScore),
		AvailabilityScore: sanitizeFloat(report.AvailabilityScore),
		LatencyScore:      sanitizeFloat(report.LatencyScore),
		GenericScore:      sanitizeFloat(report.GenericScore),
		ProviderAddress:   report.ProviderAddress,
		EntryIndex:        report.EntryIndex,
		ChainId:           chainId,
		Epoch:             epoch,
		NodeErrorRate:     sanitizeFloat(coqc.calculateNodeErrorRate(chainId, report.ProviderAddress)),
		ProviderStake:     coqc.getProviderChainStake(chainId, report.ProviderAddress, epoch),
		GeoLocation:       coqc.geoLocation,
		// WRS normalized scores
		SelectionAvailability: sanitizeFloat(report.SelectionAvailability),
		SelectionLatency:      sanitizeFloat(report.SelectionLatency),
		SelectionSync:         sanitizeFloat(report.SelectionSync),
		SelectionStake:        sanitizeFloat(report.SelectionStake),
		SelectionComposite:    sanitizeFloat(report.SelectionComposite),
		// Provider selection tracking
		SelectionCount:    coqc.getProviderChainSelectionCount(chainId, report.ProviderAddress),
		SelectionRate:     sanitizeFloat(coqc.calculateSelectionRate(chainId, report.ProviderAddress)),
		SelectionQoSScore: sanitizeFloat(coqc.calculateAverageSelectionQoSScore(chainId, report.ProviderAddress)),
		SelectionRNGValue: sanitizeFloat(coqc.getLastRNGValue(chainId, report.ProviderAddress)),
		// Weighted contributions
		AvailabilityContribution: sanitizeFloat(report.AvailabilityContribution),
		LatencyContribution:      sanitizeFloat(report.LatencyContribution),
		SyncContribution:         sanitizeFloat(report.SyncContribution),
		StakeContribution:        sanitizeFloat(report.StakeContribution),
	}

	coqc.queueSender.appendQueue(optimizerQoSReportToSend)
	return optimizerQoSReportToSend
}

func (coqc *ConsumerOptimizerQoSClient) getReportsFromOptimizers() []OptimizerQoSReportToSend {
	coqc.lock.RLock() // we only read from the maps here
	defer coqc.lock.RUnlock()

	ignoredProviders := map[string]struct{}{}
	cu := uint64(10)
	requestedBlock := spectypes.LATEST_BLOCK

	currentEpoch := coqc.currentEpoch.Load()
	reportsToSend := []OptimizerQoSReportToSend{}
	for chainId, optimizer := range coqc.optimizers {
		providersMap, ok := coqc.chainIdToProviderToEpochToStake[chainId]
		if !ok {
			continue
		}

		reports := optimizer.CalculateQoSScoresForMetrics(maps.Keys(providersMap), ignoredProviders, cu, requestedBlock)
		for _, report := range reports {
			reportsToSend = append(reportsToSend, coqc.appendOptimizerQoSReport(report, chainId, currentEpoch))
		}
	}
	return reportsToSend
}

func (coqc *ConsumerOptimizerQoSClient) SetReportsToSend(reports []OptimizerQoSReportToSend) {
	coqc.lock.Lock()
	defer coqc.lock.Unlock()
	coqc.reportsToSend = reports
}

func (coqc *ConsumerOptimizerQoSClient) GetReportsToSend() []OptimizerQoSReportToSend {
	coqc.lock.RLock()
	defer coqc.lock.RUnlock()
	return coqc.reportsToSend
}

func (coqc *ConsumerOptimizerQoSClient) StartOptimizersQoSReportsCollecting(ctx context.Context, samplingInterval time.Duration) {
	if coqc == nil {
		return
	}

	utils.LavaFormatTrace("Starting ConsumerOptimizerQoSClient reports collecting")
	go func() {
		for {
			select {
			case <-ctx.Done():
				utils.LavaFormatTrace("ConsumerOptimizerQoSClient context done")
				return
			case <-time.After(samplingInterval):
				reports := coqc.getReportsFromOptimizers()
				coqc.SetReportsToSend(reports)
			}
		}
	}()
}

func (coqc *ConsumerOptimizerQoSClient) RegisterOptimizer(optimizer OptimizerInf, chainId string) {
	if coqc == nil {
		return
	}

	coqc.lock.Lock()
	defer coqc.lock.Unlock()

	if _, found := coqc.optimizers[chainId]; found {
		utils.LavaFormatWarning("Optimizer already registered for chain", nil, utils.LogAttr("chainId", chainId))
		return
	}

	coqc.optimizers[chainId] = optimizer
}

func (coqc *ConsumerOptimizerQoSClient) incrementStoreCounter(store map[string]map[string]uint64, chainId, providerAddress string) {
	// must be called under write lock
	if coqc == nil {
		return
	}

	providersMap, found := store[chainId]
	if !found {
		store[chainId] = map[string]uint64{providerAddress: 1}
		return
	}

	count, found := providersMap[providerAddress]
	if !found {
		store[chainId][providerAddress] = 1
		return
	}

	store[chainId][providerAddress] = count + 1
}

func (coqc *ConsumerOptimizerQoSClient) SetRelaySentToProvider(providerAddress string, chainId string) {
	if coqc == nil {
		return
	}

	coqc.lock.Lock()
	defer coqc.lock.Unlock()

	coqc.incrementStoreCounter(coqc.chainIdToProviderToRelaysCount, chainId, providerAddress)
}

func (coqc *ConsumerOptimizerQoSClient) SetNodeErrorToProvider(providerAddress string, chainId string) {
	if coqc == nil {
		return
	}

	coqc.lock.Lock()
	defer coqc.lock.Unlock()

	coqc.incrementStoreCounter(coqc.chainIdToProviderToNodeErrorsCount, chainId, providerAddress)
}

func (coqc *ConsumerOptimizerQoSClient) SetProviderSelected(providerAddress string, chainId string, qosScore float64, rngValue float64) {
	if coqc == nil {
		return
	}

	coqc.lock.Lock()
	defer coqc.lock.Unlock()

	coqc.incrementStoreCounter(coqc.chainIdToProviderToSelectionCount, chainId, providerAddress)
	coqc.chainIdToTotalSelections[chainId]++

	// Accumulate QoS score for averaging
	if _, found := coqc.chainIdToProviderToQoSScoreSum[chainId]; !found {
		coqc.chainIdToProviderToQoSScoreSum[chainId] = make(map[string]float64)
	}
	coqc.chainIdToProviderToQoSScoreSum[chainId][providerAddress] += qosScore

	// Store last RNG value used for selection
	if _, found := coqc.chainIdToProviderToLastRNGValue[chainId]; !found {
		coqc.chainIdToProviderToLastRNGValue[chainId] = make(map[string]float64)
	}
	coqc.chainIdToProviderToLastRNGValue[chainId][providerAddress] = rngValue
}

func (coqc *ConsumerOptimizerQoSClient) setProviderStake(chainId, providerAddress string, epoch uint64, stake int64) {
	// must be called under write lock
	coqc.currentEpoch.Store(epoch)

	providersMap, found := coqc.chainIdToProviderToEpochToStake[chainId]
	if !found {
		coqc.chainIdToProviderToEpochToStake[chainId] = map[string]map[uint64]int64{providerAddress: {epoch: stake}}
		return
	}

	epochMap, found := providersMap[providerAddress]
	if !found {
		coqc.chainIdToProviderToEpochToStake[chainId][providerAddress] = map[uint64]int64{epoch: stake}
		return
	}

	epochMap[epoch] = stake
}

func (coqc *ConsumerOptimizerQoSClient) UpdatePairingListStake(stakeMap map[string]int64, chainId string, epoch uint64) {
	if coqc == nil {
		return
	}

	coqc.lock.Lock()
	defer coqc.lock.Unlock()

	for providerAddr, stake := range stakeMap {
		coqc.setProviderStake(chainId, providerAddr, epoch, stake)
	}
}
