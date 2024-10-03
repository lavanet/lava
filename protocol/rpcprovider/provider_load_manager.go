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

	return strconv.FormatUint(loadManager.activeRequestsPerSecond.Load()/loadManager.rateLimitThreshold.Load(), 10)
}

func (loadManager *ProviderLoadManager) applyProviderLoadMetadataToContextTrailer(ctx context.Context) {
	provideRelayLoad := loadManager.getProviderLoad()
	if provideRelayLoad == "" {
		return
	}

	trailerMd := metadata.Pairs(chainlib.RpcProviderLoadRateHeader, provideRelayLoad)
	grpc.SetTrailer(ctx, trailerMd)
}
