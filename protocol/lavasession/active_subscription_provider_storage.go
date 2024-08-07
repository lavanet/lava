package lavasession

import (
	"sync"

	"github.com/lavanet/lava/v2/utils"
)

// stores all providers that are currently used to stream subscriptions.

type NumberOfActiveSubscriptions int64

type ActiveSubscriptionProvidersStorage struct {
	lock          sync.RWMutex
	providers     map[string]NumberOfActiveSubscriptions
	purgeWhenDone map[string]func()
}

func NewActiveSubscriptionProvidersStorage() *ActiveSubscriptionProvidersStorage {
	return &ActiveSubscriptionProvidersStorage{
		providers:     map[string]NumberOfActiveSubscriptions{},
		purgeWhenDone: map[string]func(){},
	}
}

func (asps *ActiveSubscriptionProvidersStorage) AddProvider(providerAddress string) {
	asps.lock.Lock()
	defer asps.lock.Unlock()
	numberOfSubscriptionsActive := asps.providers[providerAddress]
	// Increase numberOfSubscriptionsActive by 1 (even if it didn't exist it will be 0)
	asps.providers[providerAddress] = numberOfSubscriptionsActive + 1
}

func (asps *ActiveSubscriptionProvidersStorage) RemoveProvider(providerAddress string) {
	asps.lock.Lock()
	defer asps.lock.Unlock()
	// Fetch number of currently active subscriptions for this provider address.
	activeSubscriptions, foundProviderAddress := asps.providers[providerAddress]
	if foundProviderAddress {
		// Check there are no other active subscriptions
		if activeSubscriptions <= 1 {
			delete(asps.providers, providerAddress)
			purgeCallBack, foundPurgerCb := asps.purgeWhenDone[providerAddress]
			if foundPurgerCb {
				utils.LavaFormatTrace("RemoveProvider, Purging provider on callback", utils.LogAttr("address", providerAddress))
				if purgeCallBack != nil {
					purgeCallBack()
				}
				delete(asps.purgeWhenDone, providerAddress)
			}
		} else {
			// Reduce number of active subscriptions on this provider address
			utils.LavaFormatTrace("RemoveProvider, Reducing number of active provider subscriptions", utils.LogAttr("address", providerAddress))
			asps.providers[providerAddress] = activeSubscriptions - 1
		}
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
