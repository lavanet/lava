package rpcprovider

import (
	"sync/atomic"
)

type ProviderLoadManager struct {
	TotalSimultaneousRelays int64
	ActiveRequestsPerSecond int64
}

func (loadManager *ProviderLoadManager) addRelayCall() {
	atomic.AddInt64(&loadManager.ActiveRequestsPerSecond, 1)
}

func (loadManager *ProviderLoadManager) removeRelayCall() {
	atomic.AddInt64(&loadManager.ActiveRequestsPerSecond, -1)
}

func (loadManager *ProviderLoadManager) getRelayCallCount() int64 {
	totalRelays := atomic.LoadInt64(&loadManager.TotalSimultaneousRelays)
	return totalRelays
}

func (loadManager *ProviderLoadManager) getProviderLoad() float64 {
	if loadManager.TotalSimultaneousRelays == 0 {
		return 0
	}

	return float64(loadManager.getRelayCallCount() / loadManager.TotalSimultaneousRelays)
}
