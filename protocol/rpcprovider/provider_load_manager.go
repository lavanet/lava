package rpcprovider

import (
	"context"
	"strconv"
	"sync/atomic"

	"github.com/lavanet/lava/v4/protocol/chainlib"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type ProviderLoadManager struct {
	rateLimitThreshold      uint64
	activeRequestsPerSecond atomic.Uint64
}

func NewProviderLoadManager(rateLimitThreshold uint64) *ProviderLoadManager {
	if rateLimitThreshold == 0 {
		return nil
	}
	loadManager := &ProviderLoadManager{rateLimitThreshold: rateLimitThreshold}
	return loadManager
}

func (loadManager *ProviderLoadManager) subtractRelayCall() {
	if loadManager == nil {
		return
	}
	loadManager.activeRequestsPerSecond.Add(^uint64(0))
}

func (loadManager *ProviderLoadManager) getProviderLoad(activeRequests uint64) float64 {
	rateLimitThreshold := loadManager.rateLimitThreshold
	if rateLimitThreshold == 0 {
		return 0
	}
	return float64(activeRequests) / float64(rateLimitThreshold)
}

// Add relay count, calculate current load
func (loadManager *ProviderLoadManager) addAndSetRelayLoadToContextTrailer(ctx context.Context) float64 {
	if loadManager == nil {
		return 0
	}
	activeRequestsPerSecond := loadManager.activeRequestsPerSecond.Add(1)
	provideRelayLoad := loadManager.getProviderLoad(activeRequestsPerSecond)
	if provideRelayLoad == 0 {
		return provideRelayLoad
	}
	formattedProviderLoad := strconv.FormatFloat(provideRelayLoad, 'f', -1, 64)
	trailerMd := metadata.Pairs(chainlib.RpcProviderLoadRateHeader, formattedProviderLoad)
	grpc.SetTrailer(ctx, trailerMd)
	return provideRelayLoad
}
