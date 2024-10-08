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

func (loadManager *ProviderLoadManager) getProviderLoad() string {
	if loadManager == nil {
		return ""
	}
	rateLimitThreshold := loadManager.rateLimitThreshold.Load()
	if rateLimitThreshold == 0 {
		return ""
	}
	activeRequests := loadManager.activeRequestsPerSecond.Load()
	return strconv.FormatFloat(float64(activeRequests)/float64(rateLimitThreshold), 'f', -1, 64)
}

func (loadManager *ProviderLoadManager) applyProviderLoadMetadataToContextTrailer(ctx context.Context) bool {
	provideRelayLoad := loadManager.getProviderLoad()
	if provideRelayLoad == "" {
		return false
	}

	trailerMd := metadata.Pairs(chainlib.RpcProviderLoadRateHeader, provideRelayLoad)
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
