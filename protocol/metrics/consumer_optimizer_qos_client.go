package metrics

import (
	"context"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/goccy/go-json"
	"github.com/lavanet/lava/v4/utils"
	"github.com/lavanet/lava/v4/utils/rand"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
	"golang.org/x/exp/maps"
)

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
	chainIdToProviderToEpochToStake    map[string]map[string]map[uint64]int64 // third key is epoch
	currentEpoch                       atomic.Uint64
	lock                               sync.RWMutex
	reportsToSend                      []OptimizerQoSReportToSend
}

type OptimizerQoSReport struct {
	ProviderAddress   string
	SyncScore         float64
	AvailabilityScore float64
	LatencyScore      float64
	GenericScore      float64
	EntryIndex        int
}

type OptimizerQoSReportToSend struct {
	Timestamp         time.Time `json:"timestamp"`
	SyncScore         float64   `json:"sync_score"`
	AvailabilityScore float64   `json:"availability_score"`
	LatencyScore      float64   `json:"latency_score"`
	GenericScore      float64   `json:"generic_score"`
	ProviderAddress   string    `json:"provider"`
	ConsumerHostname  string    `json:"consumer_hostname"`
	ConsumerAddress   string    `json:"consumer_pub_address"`
	ChainId           string    `json:"chain_id"`
	NodeErrorRate     float64   `json:"node_error_rate"`
	Epoch             uint64    `json:"epoch"`
	ProviderStake     int64     `json:"provider_stake"`
	EntryIndex        int       `json:"entry_index"`
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

func NewConsumerOptimizerQoSClient(consumerAddress, endpointAddress string, interval ...time.Duration) *ConsumerOptimizerQoSClient {
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
		chainIdToProviderToEpochToStake:    map[string]map[string]map[uint64]int64{},
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

func (coqc *ConsumerOptimizerQoSClient) appendOptimizerQoSReport(report *OptimizerQoSReport, chainId string, epoch uint64) OptimizerQoSReportToSend {
	// must be called under read lock
	optimizerQoSReportToSend := OptimizerQoSReportToSend{
		Timestamp:         time.Now(),
		ConsumerHostname:  coqc.consumerHostname,
		ConsumerAddress:   coqc.consumerAddress,
		SyncScore:         report.SyncScore,
		AvailabilityScore: report.AvailabilityScore,
		LatencyScore:      report.LatencyScore,
		GenericScore:      report.GenericScore,
		ProviderAddress:   report.ProviderAddress,
		EntryIndex:        report.EntryIndex,
		ChainId:           chainId,
		Epoch:             epoch,
		NodeErrorRate:     coqc.calculateNodeErrorRate(chainId, report.ProviderAddress),
		ProviderStake:     coqc.getProviderChainStake(chainId, report.ProviderAddress, epoch),
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
				coqc.SetReportsToSend(coqc.getReportsFromOptimizers())
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
