package metrics

import (
	"context"
	"sync"
	"time"
)

type ProviderDataScore struct {
	Availability float64
	Sync         float64
	Latency      float64
}

type ProviderDataWithChosen struct {
	providerData ProviderDataScore
	epoch        uint64
	chosen       bool
}

type ConsumerOptimizerDataCollector struct {
	latestProvidersOptimizerData map[string]ProviderDataWithChosen
	interval                     time.Duration
	consumerMetricsManager       *ConsumerMetricsManager
	chainId                      string
	apiInterface                 string
	lock                         sync.RWMutex
}

func NewConsumerOptimizerDataCollector(chainId, apiInterface string, interval time.Duration, consumerMetricsManager *ConsumerMetricsManager) *ConsumerOptimizerDataCollector {
	return &ConsumerOptimizerDataCollector{
		latestProvidersOptimizerData: make(map[string]ProviderDataWithChosen),
		chainId:                      chainId,
		interval:                     interval,
		consumerMetricsManager:       consumerMetricsManager,
		apiInterface:                 apiInterface,
	}
}

func (codc *ConsumerOptimizerDataCollector) GetChinId() string {
	if codc == nil {
		return ""
	}

	return codc.chainId
}

func (codc *ConsumerOptimizerDataCollector) GetApiInterface() string {
	if codc == nil {
		return ""
	}

	return codc.apiInterface
}

func (codc *ConsumerOptimizerDataCollector) SetProviderData(providerAddress string, epoch uint64, chosen bool, availability, sync, latency float64) {
	if codc == nil {
		return
	}

	codc.lock.Lock()
	defer codc.lock.Unlock()
	codc.latestProvidersOptimizerData[providerAddress] = ProviderDataWithChosen{
		providerData: ProviderDataScore{
			Availability: availability,
			Sync:         sync,
			Latency:      latency,
		},
		epoch:  epoch,
		chosen: chosen,
	}
}

func (codc *ConsumerOptimizerDataCollector) Start(ctx context.Context) {
	if codc == nil {
		return
	}

	go func() {
		ticker := time.NewTicker(codc.interval)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				codc.collectProvidersData()
			}
		}
	}()
}

func (codc *ConsumerOptimizerDataCollector) collectProvidersData() {
	if codc == nil {
		return
	}

	codc.lock.RLock()
	defer codc.lock.RUnlock()

	for providerAddress, providerData := range codc.latestProvidersOptimizerData {
		codc.consumerMetricsManager.UpdateOptimizerChoosingProviderWithData(
			codc.chainId,
			codc.apiInterface,
			providerAddress,
			providerData.epoch,
			providerData.chosen,
			providerData.providerData,
		)
	}
}
