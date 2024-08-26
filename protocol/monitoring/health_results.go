package monitoring

import (
	"sync"
	"time"

	"github.com/lavanet/lava/v2/protocol/common"
	"github.com/lavanet/lava/v2/utils/lavaslices"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
)

type HealthResults struct {
	LatestBlocks       map[string]int64            `json:"latestBlocks,omitempty"`
	ConsumerBlocks     map[LavaEntity]int64        `json:"consumerBlocks,omitempty"`
	ProviderData       map[LavaEntity]ReplyData    `json:"providerData,omitempty"`
	SubscriptionsData  map[string]SubscriptionData `json:"subscriptionsData,omitempty"`
	FrozenProviders    map[LavaEntity]struct{}     `json:"frozenProviders,omitempty"`
	UnhealthyProviders map[LavaEntity]string       `json:"unhealthyProviders,omitempty"`
	UnhealthyConsumers map[LavaEntity]string       `json:"unhealthyConsumers,omitempty"`
	Specs              map[string]*spectypes.Spec  `json:"specs,omitempty"`
	ResultsPostGUID    string                      `json:"resultsGUID,omitempty"`
	ProviderAddresses  []string                    `json:"providerAddresses,omitempty"`
	Lock               sync.RWMutex                `json:"-"`
}

func (healthResults *HealthResults) FormatForLatestBlock() map[string]uint64 {
	healthResults.Lock.RLock()
	defer healthResults.Lock.RUnlock()
	results := map[string]uint64{}
	for entity, block := range healthResults.ConsumerBlocks {
		results[entity.String()] = uint64(block)
	}
	for entity, data := range healthResults.ProviderData {
		results[entity.String()] = uint64(data.Block)
	}
	for entity := range healthResults.UnhealthyProviders {
		results[entity.String()] = 0
	}
	for entity := range healthResults.UnhealthyConsumers {
		results[entity.String()] = 0
	}
	return results
}

func (healthResults *HealthResults) GetAllEntities() map[LavaEntity]struct{} {
	healthResults.Lock.RLock()
	defer healthResults.Lock.RUnlock()
	entities := map[LavaEntity]struct{}{}
	for entity := range healthResults.FrozenProviders {
		entities[entity] = struct{}{}
	}
	for entity := range healthResults.UnhealthyProviders {
		entities[entity] = struct{}{}
	}
	for entity := range healthResults.UnhealthyConsumers {
		entities[entity] = struct{}{}
	}
	for entity := range healthResults.ConsumerBlocks {
		entities[entity] = struct{}{}
	}
	for entity := range healthResults.ProviderData {
		entities[entity] = struct{}{}
	}
	for entitySt := range healthResults.SubscriptionsData {
		entity := LavaEntity{
			Address: entitySt,
			SpecId:  "",
		}
		entities[entity] = struct{}{}
	}
	return entities
}

func (healthResults *HealthResults) FreezeProvider(providerKey LavaEntity) {
	healthResults.Lock.Lock()
	defer healthResults.Lock.Unlock()
	healthResults.FrozenProviders[providerKey] = struct{}{}
}

func (healthResults *HealthResults) setSpec(spec *spectypes.Spec) {
	healthResults.Lock.Lock()
	defer healthResults.Lock.Unlock()
	healthResults.Specs[spec.Index] = spec
}

func (healthResults *HealthResults) updateLatestBlock(specId string, latestBlock int64) {
	healthResults.Lock.Lock()
	defer healthResults.Lock.Unlock()
	existing, ok := healthResults.LatestBlocks[specId]
	if !ok {
		healthResults.LatestBlocks[specId] = latestBlock
	} else {
		healthResults.LatestBlocks[specId] = lavaslices.Max([]int64{existing, latestBlock})
	}
}

func (healthResults *HealthResults) updateConsumerError(endpoint *HealthRPCEndpoint, err error) {
	healthResults.Lock.Lock()
	defer healthResults.Lock.Unlock()
	healthResults.ConsumerBlocks[LavaEntity{
		Address:      endpoint.String(),
		SpecId:       endpoint.ChainID,
		ApiInterface: endpoint.ApiInterface,
	}] = 0
	msg := "invalid response"
	if len(err.Error()) < NiceOutputLength {
		msg = err.Error()
	}
	if common.IsTimeout(err) {
		msg = "timeout"
	}
	healthResults.UnhealthyConsumers[LavaEntity{
		Address:      endpoint.String(),
		SpecId:       endpoint.ChainID,
		ApiInterface: endpoint.ApiInterface,
	}] = msg
}

func (healthResults *HealthResults) updateConsumer(endpoint *HealthRPCEndpoint, latestBlock int64) {
	healthResults.Lock.Lock()
	defer healthResults.Lock.Unlock()
	healthResults.ConsumerBlocks[LavaEntity{
		Address:      endpoint.String(),
		SpecId:       endpoint.ChainID,
		ApiInterface: endpoint.ApiInterface,
	}] = latestBlock
}

func (healthResults *HealthResults) getSpecs() map[string]*spectypes.Spec {
	healthResults.Lock.RLock()
	defer healthResults.Lock.RUnlock()
	return healthResults.Specs
}

func (healthResults *HealthResults) getSpec(specId string) *spectypes.Spec {
	healthResults.Lock.RLock()
	defer healthResults.Lock.RUnlock()
	return healthResults.Specs[specId]
}

func (healthResults *HealthResults) getProviderData(providerKey LavaEntity) (latestData ReplyData, ok bool) {
	healthResults.Lock.Lock()
	defer healthResults.Lock.Unlock()
	data, ok := healthResults.ProviderData[providerKey]
	return data, ok
}

func (healthResults *HealthResults) SetProviderData(providerKey LavaEntity, latestData ReplyData) {
	healthResults.Lock.Lock()
	defer healthResults.Lock.Unlock()
	if existing, ok := healthResults.ProviderData[providerKey]; ok {
		if existing.Block == 0 {
			existing.Block = latestData.Block
		} else {
			latestData.Block = lavaslices.Min([]int64{existing.Block, latestData.Block})
		}
		latestData.Latency = lavaslices.Max([]time.Duration{existing.Latency, latestData.Latency})
	}
	healthResults.ProviderData[providerKey] = latestData
}

func (healthResults *HealthResults) SetUnhealthyProvider(providerKey LavaEntity, errMsg string) {
	healthResults.Lock.Lock()
	defer healthResults.Lock.Unlock()
	healthResults.UnhealthyProviders[providerKey] = errMsg
}

func (healthResults *HealthResults) setSubscriptionData(subscriptionAddr string, data SubscriptionData) {
	healthResults.Lock.Lock()
	defer healthResults.Lock.Unlock()
	healthResults.SubscriptionsData[subscriptionAddr] = data
}
