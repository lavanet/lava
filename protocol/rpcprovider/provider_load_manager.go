package rpcprovider

import (
	"sync/atomic"
)

type ProviderLoadManager struct {
	totalSimultaneousRelays int64
	activeRequestsPerSecond int64
}

func (loadManager *ProviderLoadManager) addRelayCall() {
	atomic.AddInt64(&loadManager.totalSimultaneousRelays, 1)
}

func (loadManager *ProviderLoadManager) removeRelayCall() {
	atomic.AddInt64(&loadManager.totalSimultaneousRelays, -1)
}

func (loadManager *ProviderLoadManager) getRelayCallCount() int64 {
	atomic.LoadInt64(&loadManager.totalSimultaneousRelays)
	return loadManager.totalSimultaneousRelays
}

func (loadManager *ProviderLoadManager) getProviderLoad() float64 {
	if loadManager.getRelayCallCount() == 0 {
		return 0
	}

	return float64(loadManager.activeRequestsPerSecond / loadManager.getRelayCallCount())
}
