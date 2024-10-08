package rpcprovider

import (
	"context"
	"strconv"
	"sync/atomic"

	"github.com/lavanet/lava/v3/protocol/chainlib"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
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

func (loadManager *ProviderLoadManager) getProviderLoad() float64 {
	if loadManager == nil {
		return 0
	}
	rateLimitThreshold := loadManager.rateLimitThreshold.Load()
	if rateLimitThreshold == 0 {
		return 0
	}
	activeRequests := loadManager.activeRequestsPerSecond.Load()
	return float64(activeRequests) / float64(rateLimitThreshold)
}

func (loadManager *ProviderLoadManager) applyProviderLoadMetadataToContextTrailer(ctx context.Context) bool {
	provideRelayLoad := loadManager.getProviderLoad()
	if provideRelayLoad == 0 {
		return false
	}
	formattedProviderLoad := strconv.FormatFloat(provideRelayLoad, 'f', -1, 64)

	trailerMd := metadata.Pairs(chainlib.RpcProviderLoadRateHeader, formattedProviderLoad)
	grpc.SetTrailer(ctx, trailerMd)
	return true
}

func (loadManager *ProviderLoadManager) addAndSetRelayLoadToContextTrailer(ctx context.Context) bool {
	loadManager.addRelayCall()
	return loadManager.applyProviderLoadMetadataToContextTrailer(ctx)
}

func (loadManager *ProviderLoadManager) getActiveRequestsPerSecond() uint64 {
	return loadManager.activeRequestsPerSecond.Load()
}
