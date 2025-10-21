package provideroptimizer

import (
	"sync"
)

// ProviderStakeCache maintains provider stake amounts for use in selection algorithms.
// It provides thread-safe access to stake values used by the weighted selector to
// calculate provider selection probabilities.
type ProviderStakeCache interface {
	GetStake(address string) int64
	UpdateStakes(stakes map[string]int64)
}

type providerStakeCacheInst struct {
	lock   sync.RWMutex
	stakes map[string]int64
}

func NewProviderStakeCache() ProviderStakeCache {
	return &providerStakeCacheInst{
		stakes: make(map[string]int64),
	}
}

func (psc *providerStakeCacheInst) GetStake(address string) int64 {
	psc.lock.RLock()
	defer psc.lock.RUnlock()
	return psc.getStakeInner(address)
}

// assumes lock is held
func (psc *providerStakeCacheInst) getStakeInner(address string) int64 {
	stake, ok := psc.stakes[address]
	if !ok {
		// default stake is 1 to ensure all providers can be selected
		return 1
	}
	return stake
}

func (psc *providerStakeCacheInst) UpdateStakes(stakes map[string]int64) {
	psc.lock.Lock()
	defer psc.lock.Unlock()
	for address, stake := range stakes {
		psc.stakes[address] = stake
	}
}
