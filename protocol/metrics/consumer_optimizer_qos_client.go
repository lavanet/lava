package metrics

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/goccy/go-json"
	"github.com/lavanet/lava/v3/utils"
	epochstoragetypes "github.com/lavanet/lava/v3/x/epochstorage/types"
	spectypes "github.com/lavanet/lava/v3/x/spec/types"
)

type ConsumerOptimizerQoSClient struct {
	consumerOrigin string
	queueSender    *QueueSender
	optimizers     map[string]OptimizerInf // keys are chain ids
	// keys are provider addresses
	providerToChainIdToRelaysCount     map[string]map[string]*atomic.Uint64
	providerToChainIdToNodeErrorsCount map[string]map[string]*atomic.Uint64
	providerToChainIdToEpochToStake    map[string]map[string]map[uint64]uint64
	atomicCurrentEpoch                 uint64
	lock                               sync.RWMutex
}

type OptimizerQoSReport struct {
	ProviderAddress   string
	SyncScore         float64
	AvailabilityScore float64
	LatencyScore      float64
	GenericScore      float64
}

type optimizerQoSReportToSend struct {
	Timestamp         time.Time `json:"timestamp"`
	SyncScore         float64   `json:"sync_score"`
	AvailabilityScore float64   `json:"availability_score"`
	LatencyScore      float64   `json:"latency_score"`
	GenericScore      float64   `json:"generic_score"`
	ProviderAddress   string    `json:"provider"`
	ConsumerOrigin    string    `json:"consumer"`
	ChainId           string    `json:"chain_id"`
	NodeErrorRate     float64   `json:"node_error_rate"`
	Epoch             uint64    `json:"epoch"`
	ProviderStake     uint64    `json:"provider_stake"`
}

func (oqosr optimizerQoSReportToSend) String() string {
	bytes, err := json.Marshal(oqosr)
	if err != nil {
		return ""
	}
	return string(bytes)
}

type OptimizerInf interface {
	CalculateQoSScoresForMetrics(allAddresses []string, ignoredProviders map[string]struct{}, cu uint64, requestedBlock int64) []OptimizerQoSReport
}

func NewConsumerOptimizerQoSClient(endpointAddress string, interval ...time.Duration) *ConsumerOptimizerQoSClient {
	hostname, err := os.Hostname()
	if err != nil {
		utils.LavaFormatWarning("Error while getting hostname for ConsumerOptimizerQoSClient", err)
		hostname = "unknown"
	}

	return &ConsumerOptimizerQoSClient{
		consumerOrigin:                     hostname,
		queueSender:                        NewQueueSender(endpointAddress, "ConsumerOptimizerQoS", nil, interval...),
		optimizers:                         map[string]OptimizerInf{},
		providerToChainIdToRelaysCount:     map[string]map[string]*atomic.Uint64{},
		providerToChainIdToNodeErrorsCount: map[string]map[string]*atomic.Uint64{},
		providerToChainIdToEpochToStake:    map[string]map[string]map[uint64]uint64{},
	}
}

func (coqc *ConsumerOptimizerQoSClient) getProviderChainMapCounterValue(counterStore map[string]map[string]*atomic.Uint64, providerAddress, chainId string) uint64 {
	// must be called under read lock
	if counterChainsMap, found := counterStore[providerAddress]; found {
		if atomicRelaysCount, found := counterChainsMap[chainId]; found {
			return atomicRelaysCount.Load()
		}
	}
	return 0
}

func (coqc *ConsumerOptimizerQoSClient) getProviderChainRelaysCount(providerAddress, chainId string) uint64 {
	// must be called under read lock
	return coqc.getProviderChainMapCounterValue(coqc.providerToChainIdToRelaysCount, providerAddress, chainId)
}

func (coqc *ConsumerOptimizerQoSClient) getProviderChainNodeErrorsCount(providerAddress, chainId string) uint64 {
	// must be called under read lock
	return coqc.getProviderChainMapCounterValue(coqc.providerToChainIdToNodeErrorsCount, providerAddress, chainId)
}

func (coqc *ConsumerOptimizerQoSClient) getProviderChainStake(providerAddress, chainId string, epoch uint64) uint64 {
	// must be called under read lock
	if chainMap, found := coqc.providerToChainIdToEpochToStake[providerAddress]; found {
		if epochMap, found := chainMap[chainId]; found {
			if stake, found := epochMap[epoch]; found {
				return stake
			}
		}
	}
	return 0
}

func (coqc *ConsumerOptimizerQoSClient) calculateNodeErrorRate(providerAddress, chainId string) float64 {
	// must be called under read lock
	relaysCount := coqc.getProviderChainRelaysCount(providerAddress, chainId)
	if relaysCount > 0 {
		errorsCount := coqc.getProviderChainNodeErrorsCount(providerAddress, chainId)
		return float64(errorsCount) / float64(relaysCount)
	}

	return 0
}

func (coqc *ConsumerOptimizerQoSClient) appendOptimizerQoSReport(report OptimizerQoSReport, chainId string, epoch uint64) {
	// must be called under read lock
	if coqc == nil {
		return
	}

	optimizerQoSReportToSend := optimizerQoSReportToSend{
		Timestamp:         time.Now(),
		ConsumerOrigin:    coqc.consumerOrigin,
		SyncScore:         report.SyncScore,
		AvailabilityScore: report.AvailabilityScore,
		LatencyScore:      report.LatencyScore,
		GenericScore:      report.GenericScore,
		ProviderAddress:   report.ProviderAddress,
		ChainId:           chainId,
		Epoch:             epoch,
		NodeErrorRate:     coqc.calculateNodeErrorRate(report.ProviderAddress, chainId),
		ProviderStake:     coqc.getProviderChainStake(report.ProviderAddress, chainId, epoch),
	}

	coqc.queueSender.appendQueue(optimizerQoSReportToSend)
}

func (coqc *ConsumerOptimizerQoSClient) createChainToProvidersMap() map[string][]string {
	// must be called under read lock
	chainIdToProviderAddresses := map[string][]string{}
	for providerAddr, chainsMap := range coqc.providerToChainIdToEpochToStake {
		for chainId := range chainsMap {
			if _, ok := chainIdToProviderAddresses[chainId]; !ok {
				chainIdToProviderAddresses[chainId] = []string{}
			}

			chainIdToProviderAddresses[chainId] = append(chainIdToProviderAddresses[chainId], providerAddr)
		}
	}

	return chainIdToProviderAddresses
}

func (coqc *ConsumerOptimizerQoSClient) getReportsFromOptimizers() {
	coqc.lock.RLock() // we only read from the maps here
	defer coqc.lock.RUnlock()

	ignoredProviders := map[string]struct{}{}
	cu := uint64(10)
	requestedBlock := spectypes.LATEST_BLOCK

	chainIdToProviders := coqc.createChainToProvidersMap()
	currentEpoch := atomic.LoadUint64(&coqc.atomicCurrentEpoch)

	for chainId, optimizer := range coqc.optimizers {
		providersAddresses, ok := chainIdToProviders[chainId]
		if !ok {
			continue
		}

		reports := optimizer.CalculateQoSScoresForMetrics(providersAddresses, ignoredProviders, cu, requestedBlock)
		for _, report := range reports {
			coqc.appendOptimizerQoSReport(report, chainId, currentEpoch)
		}
	}
}

func (coqc *ConsumerOptimizerQoSClient) StartOptimizersQoSReportsCollecting(ctx context.Context, samplingInterval time.Duration) {
	utils.LavaFormatTrace("Starting ConsumerOptimizerQoSClient reports collecting")
	go func() {
		for {
			select {
			case <-ctx.Done():
				utils.LavaFormatTrace("ConsumerOptimizerQoSClient context done")
				return
			case <-time.After(samplingInterval):
				coqc.getReportsFromOptimizers()
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

func (coqc *ConsumerOptimizerQoSClient) initAtomicUint64() *atomic.Uint64 {
	newAtomic := &atomic.Uint64{}
	newAtomic.Add(1)
	return newAtomic
}

func (coqc *ConsumerOptimizerQoSClient) incrementStoreCounter(store map[string]map[string]*atomic.Uint64, providerAddress, chainId string) {
	// must be called under write lock
	if coqc == nil {
		return
	}

	chainMap, found := store[providerAddress]
	if !found {
		store[providerAddress] = map[string]*atomic.Uint64{chainId: coqc.initAtomicUint64()}
		return
	}

	count, found := chainMap[chainId]
	if !found {
		store[providerAddress][chainId] = coqc.initAtomicUint64()
		return
	}

	count.Add(1)
}

func (coqc *ConsumerOptimizerQoSClient) SetRelaySentToProvider(providerAddress string, chainId string) {
	if coqc == nil {
		return
	}

	coqc.lock.Lock()
	defer coqc.lock.Unlock()

	coqc.incrementStoreCounter(coqc.providerToChainIdToRelaysCount, providerAddress, chainId)
}

func (coqc *ConsumerOptimizerQoSClient) SetNodeErrorToProvider(providerAddress string, chainId string) {
	if coqc == nil {
		return
	}

	coqc.lock.Lock()
	defer coqc.lock.Unlock()

	coqc.incrementStoreCounter(coqc.providerToChainIdToNodeErrorsCount, providerAddress, chainId)
}

func (coqc *ConsumerOptimizerQoSClient) setProviderStake(providerAddress, chainId string, epoch, stake uint64) {
	// must be called under write lock
	atomic.StoreUint64(&coqc.atomicCurrentEpoch, epoch)

	chainMap, found := coqc.providerToChainIdToEpochToStake[providerAddress]
	if !found {
		coqc.providerToChainIdToEpochToStake[providerAddress] = map[string]map[uint64]uint64{chainId: {epoch: stake}}
		return
	}

	epochMap, found := chainMap[chainId]
	if !found {
		coqc.providerToChainIdToEpochToStake[providerAddress][chainId] = map[uint64]uint64{epoch: stake}
		return
	}

	epochMap[epoch] = stake
}

func (coqc *ConsumerOptimizerQoSClient) UpdatePairingStakeEntries(pairingList []epochstoragetypes.StakeEntry, epoch uint64) {
	if coqc == nil {
		return
	}

	coqc.lock.Lock()
	defer coqc.lock.Unlock()

	for _, pairing := range pairingList {
		coqc.setProviderStake(pairing.Address, pairing.Chain, epoch, pairing.Stake.Amount.Uint64())
	}
}
