package lavasession

import (
	"sync"

	"github.com/lavanet/lava/utils"
)

// stores all providers that are currently used to stream subscriptions.

type ActiveSubscriptionProvidersStorage struct {
	lock          sync.RWMutex
	providers     map[string]struct{}
	purgeWhenDone map[string]func()
}

func NewActiveSubscriptionProvidersStorage() *ActiveSubscriptionProvidersStorage {
	return &ActiveSubscriptionProvidersStorage{
		providers:     map[string]struct{}{},
		purgeWhenDone: map[string]func(){},
	}
}

func (asps *ActiveSubscriptionProvidersStorage) AddProvider(providerAddress string) {
	asps.lock.Lock()
	defer asps.lock.Unlock()

	asps.providers[providerAddress] = struct{}{}
}

func (asps *ActiveSubscriptionProvidersStorage) RemoveProvider(providerAddress string) {
	asps.lock.Lock()
	defer asps.lock.Unlock()

	delete(asps.providers, providerAddress)
	purgeCallBack, ok := asps.purgeWhenDone[providerAddress]
	if ok {
		utils.LavaFormatTrace("RemoveProvider, Purging provider on callback", utils.LogAttr("address", providerAddress))
		if purgeCallBack != nil {
			purgeCallBack()
		}
		delete(asps.purgeWhenDone, providerAddress)
	}
}

func (asps *ActiveSubscriptionProvidersStorage) IsProviderCurrentlyUsed(providerAddress string) bool {
	asps.lock.RLock()
	defer asps.lock.RUnlock()

	_, ok := asps.providers[providerAddress]
	return ok
}

func (asps *ActiveSubscriptionProvidersStorage) addToPurgeWhenDone(providerAddress string, purgeCallback func()) {
	asps.lock.Lock()
	defer asps.lock.Unlock()
	asps.purgeWhenDone[providerAddress] = purgeCallback
}
