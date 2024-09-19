package metrics

import (
	"context"
	"os"
	"sync/atomic"
	"time"

	"github.com/goccy/go-json"
	"github.com/lavanet/lava/v3/protocol/common"
	"github.com/lavanet/lava/v3/utils"
	epochstoragetypes "github.com/lavanet/lava/v3/x/epochstorage/types"
	spectypes "github.com/lavanet/lava/v3/x/spec/types"
)

type ConsumerOptimizerQoSClient struct {
	consumerOrigin string
	queueSender    *QueueSender
	optimizers     *common.SafeSyncMap[string, OptimizerInf] // keys are chain ids
	// keys are provider addresses
	providerToChainIdToRelaysCount     *common.SafeSyncMap[string, *common.SafeSyncMap[string, *atomic.Uint64]]
	providerToChainIdToNodeErrorsCount *common.SafeSyncMap[string, *common.SafeSyncMap[string, *atomic.Uint64]]
	providerToChainIdToEpochToStake    *common.SafeSyncMap[string, *common.SafeSyncMap[string, *common.SafeSyncMap[uint64, uint64]]]
	atomicCurrentEpoch                 uint64
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
		optimizers:                         &common.SafeSyncMap[string, OptimizerInf]{},
		providerToChainIdToRelaysCount:     &common.SafeSyncMap[string, *common.SafeSyncMap[string, *atomic.Uint64]]{},
		providerToChainIdToNodeErrorsCount: &common.SafeSyncMap[string, *common.SafeSyncMap[string, *atomic.Uint64]]{},
		providerToChainIdToEpochToStake:    &common.SafeSyncMap[string, *common.SafeSyncMap[string, *common.SafeSyncMap[uint64, uint64]]]{},
	}
}

func (coqc *ConsumerOptimizerQoSClient) getProviderChainMapCounterValue(counterStore *common.SafeSyncMap[string, *common.SafeSyncMap[string, *atomic.Uint64]], providerAddress, chainId string) uint64 {
	counterChainsMap, loaded, _ := counterStore.Load(providerAddress)
	if loaded {
		atomicRelaysCount, loaded, _ := counterChainsMap.Load(chainId)
		if loaded {
			return atomicRelaysCount.Load()
		}
	}
	return 0
}

func (coqc *ConsumerOptimizerQoSClient) getProviderChainRelaysCount(providerAddress, chainId string) uint64 {
	return coqc.getProviderChainMapCounterValue(coqc.providerToChainIdToRelaysCount, providerAddress, chainId)
}

func (coqc *ConsumerOptimizerQoSClient) getProviderChainNodeErrorsCount(providerAddress, chainId string) uint64 {
	return coqc.getProviderChainMapCounterValue(coqc.providerToChainIdToNodeErrorsCount, providerAddress, chainId)
}

func (coqc *ConsumerOptimizerQoSClient) getProviderChainStake(providerAddress, chainId string, epoch uint64) uint64 {
	chainMap, loaded, _ := coqc.providerToChainIdToEpochToStake.Load(providerAddress)
	if loaded {
		epochMap, loaded, _ := chainMap.Load(chainId)
		if loaded {
			stake, loaded, _ := epochMap.Load(epoch)
			if loaded {
				return stake
			}
		}
	}
	return 0
}

func (coqc *ConsumerOptimizerQoSClient) calculateNodeErrorRate(providerAddress, chainId string) float64 {
	relaysCount := coqc.getProviderChainRelaysCount(providerAddress, chainId)
	if relaysCount > 0 {
		errorsCount := coqc.getProviderChainNodeErrorsCount(providerAddress, chainId)
		return float64(errorsCount) / float64(relaysCount)
	}

	return 0
}

func (coqc *ConsumerOptimizerQoSClient) appendOptimizerQoSReport(report OptimizerQoSReport, chainId string, epoch uint64) {
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
	chainIdToProviderAddresses := map[string][]string{}
	coqc.providerToChainIdToEpochToStake.Range(func(providerAddr string, chainsMap *common.SafeSyncMap[string, *common.SafeSyncMap[uint64, uint64]]) bool {
		chainsMap.Range(func(chainId string, _ *common.SafeSyncMap[uint64, uint64]) bool {
			if _, ok := chainIdToProviderAddresses[chainId]; !ok {
				chainIdToProviderAddresses[chainId] = []string{}
			}

			chainIdToProviderAddresses[chainId] = append(chainIdToProviderAddresses[chainId], providerAddr)
			return true
		})
		return true
	})

	return chainIdToProviderAddresses
}

func (coqc *ConsumerOptimizerQoSClient) getReportsFromOptimizers() {
	ignoredProviders := map[string]struct{}{}
	cu := uint64(10)
	requestedBlock := spectypes.LATEST_BLOCK

	chainIdToProviders := coqc.createChainToProvidersMap()
	currentEpoch := atomic.LoadUint64(&coqc.atomicCurrentEpoch)

	coqc.optimizers.Range(func(chainId string, optimizer OptimizerInf) bool {
		providersAddresses, ok := chainIdToProviders[chainId]
		if !ok {
			return true
		}

		reports := optimizer.CalculateQoSScoresForMetrics(providersAddresses, ignoredProviders, cu, requestedBlock)
		for _, report := range reports {
			coqc.appendOptimizerQoSReport(report, chainId, currentEpoch)
		}

		return true
	})
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

	coqc.optimizers.Store(chainId, optimizer)
}

func (coqc *ConsumerOptimizerQoSClient) initAtomicUint64() *atomic.Uint64 {
	newAtomic := &atomic.Uint64{}
	newAtomic.Add(1)
	return newAtomic
}

func (coqc *ConsumerOptimizerQoSClient) incrementStoreCounter(store *common.SafeSyncMap[string, *common.SafeSyncMap[string, *atomic.Uint64]], providerAddress, chainId string) {
	if coqc == nil {
		return
	}

	atomicRelaysCount := coqc.initAtomicUint64()
	newChainMap := &common.SafeSyncMap[string, *atomic.Uint64]{}
	newChainMap.Store(chainId, atomicRelaysCount)

	chainMap, loaded, err := store.LoadOrStore(providerAddress, newChainMap)
	if err != nil {
		utils.LavaFormatWarning("Error while loading or storing chain map", err)
		return
	}

	if loaded {
		count, loaded, err := chainMap.LoadOrStore(chainId, atomicRelaysCount)
		if err != nil {
			utils.LavaFormatWarning("Error while loading counter", err)
			return
		}
		if loaded {
			count.Add(1)
		}
	}
}

func (coqc *ConsumerOptimizerQoSClient) SetRelaySentToProvider(providerAddress string, chainId string) {
	if coqc == nil {
		return
	}

	coqc.incrementStoreCounter(coqc.providerToChainIdToRelaysCount, providerAddress, chainId)
}

func (coqc *ConsumerOptimizerQoSClient) SetNodeErrorToProvider(providerAddress string, chainId string) {
	if coqc == nil {
		return
	}

	coqc.incrementStoreCounter(coqc.providerToChainIdToNodeErrorsCount, providerAddress, chainId)
}

func (coqc *ConsumerOptimizerQoSClient) SetProviderStake(providerAddress, chainId string, epoch, stake uint64) {
	if coqc == nil {
		return
	}

	atomic.StoreUint64(&coqc.atomicCurrentEpoch, epoch)

	newEpochMap := &common.SafeSyncMap[uint64, uint64]{}
	newEpochMap.Store(epoch, stake)

	newChainMap := &common.SafeSyncMap[string, *common.SafeSyncMap[uint64, uint64]]{}
	newChainMap.Store(chainId, newEpochMap)

	chainMap, loaded, err := coqc.providerToChainIdToEpochToStake.LoadOrStore(providerAddress, newChainMap)
	if err != nil {
		utils.LavaFormatWarning("Error while loading or storing chain map", err)
		return
	}

	if loaded {
		epochMap, loaded, err := chainMap.LoadOrStore(chainId, newEpochMap)
		if err != nil {
			utils.LavaFormatWarning("Error while loading or storing epoch map", err)
			return
		}

		if loaded {
			epochMap.Store(epoch, stake)
		}
	}
}

func (coqc *ConsumerOptimizerQoSClient) UpdatePairingStakeEntries(pairingList []epochstoragetypes.StakeEntry, epoch uint64) {
	if coqc == nil {
		return
	}

	for _, pairing := range pairingList {
		coqc.SetProviderStake(pairing.Address, pairing.Chain, epoch, pairing.Stake.Amount.Uint64())
	}
}
