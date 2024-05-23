package lavasession

import "sync"

type LongLastingProvidersStorage struct {
	lock      sync.RWMutex
	providers map[string]struct{}
}

func NewLongLastingProvidersStorage() *LongLastingProvidersStorage {
	return &LongLastingProvidersStorage{
		providers: map[string]struct{}{},
	}
}

func (llps *LongLastingProvidersStorage) AddProvider(providerAddress string) {
	llps.lock.Lock()
	defer llps.lock.Unlock()

	llps.providers[providerAddress] = struct{}{}
}

func (llps *LongLastingProvidersStorage) RemoveProvider(providerAddress string) {
	llps.lock.Lock()
	defer llps.lock.Unlock()

	delete(llps.providers, providerAddress)
}

func (llps *LongLastingProvidersStorage) IsProviderLongLasting(providerAddress string) bool {
	llps.lock.RLock()
	defer llps.lock.RUnlock()

	_, ok := llps.providers[providerAddress]
	return ok
}
