package lavasession

import "sync"

// stores all providers that are currently used to stream subscriptions.
type ActiveSubscriptionProvidersStorage struct {
	lock      sync.RWMutex
	providers map[string]struct{}
}

func NewActiveSubscriptionProvidersStorage() *ActiveSubscriptionProvidersStorage {
	return &ActiveSubscriptionProvidersStorage{
		providers: map[string]struct{}{},
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
}

func (asps *ActiveSubscriptionProvidersStorage) IsProviderCurrentlyUsed(providerAddress string) bool {
	asps.lock.RLock()
	defer asps.lock.RUnlock()

	_, ok := asps.providers[providerAddress]
	return ok
}
