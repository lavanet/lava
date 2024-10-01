package rpcprovider

import (
	"strconv"
	"sync/atomic"
)

type ProviderLoadManager struct {
	rateLimitThreshold      atomic.Uint64
	activeRequestsPerSecond atomic.Uint64
}

func NewProviderLoadManager(rateLimitThreshold uint64) *ProviderLoadManager {
	if rateLimitThreshold == 0 {
		return nil
	}
	loadManager := &ProviderLoadManager{}

	loadManager.rateLimitThreshold.Store(rateLimitThreshold)
	loadManager.activeRequestsPerSecond.Store(0)

	return loadManager
}

func (loadManager *ProviderLoadManager) addRelayCall() {
	if loadManager == nil {
		return
	}
	loadManager.activeRequestsPerSecond.Add(1)
}

func (loadManager *ProviderLoadManager) subtractRelayCall() {
	if loadManager == nil {
		return
	}
	loadManager.activeRequestsPerSecond.Add(^uint64(0))
}

func (loadManager *ProviderLoadManager) getProviderLoad() string {
	if loadManager == nil {
		return ""
	}
	loadedRateLimitThreshold := loadManager.rateLimitThreshold.Load()
	if loadedRateLimitThreshold == 0 {
		return "0"
	}

	return strconv.FormatUint(loadManager.activeRequestsPerSecond.Load()/loadedRateLimitThreshold, 10)
}
