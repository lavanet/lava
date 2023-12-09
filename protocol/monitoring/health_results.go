package monitoring

import (
	"sync"
	"time"

	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/utils/slices"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type HealthResults struct {
	LatestBlocks       map[string]int64
	ProviderData       map[LavaEntity]ReplyData
	ConsumerBlocks     map[LavaEntity]int64
	SubscriptionsData  map[string]SubscriptionData
	FrozenProviders    map[LavaEntity]struct{}
	UnhealthyProviders map[LavaEntity]string
	Specs              map[string]*spectypes.Spec
	Lock               sync.RWMutex
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
		healthResults.LatestBlocks[specId] = slices.Max([]int64{existing, latestBlock})
	}
}

func (healthResults *HealthResults) updateConsumer(endpoint *lavasession.RPCEndpoint, latestBlock int64) {
	healthResults.Lock.Lock()
	defer healthResults.Lock.Unlock()
	healthResults.ConsumerBlocks[LavaEntity{
		Address: endpoint.String(),
		SpecId:  endpoint.ChainID,
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
		if latestData.block == 0 {
			latestData.block = existing.block
		} else {
			latestData.block = slices.Min([]int64{existing.block, latestData.block})
		}
		latestData.latency = slices.Max([]time.Duration{existing.latency, latestData.latency})
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
