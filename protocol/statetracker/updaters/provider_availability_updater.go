package updaters

import (
	"context"
	"sync"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/lavanet/lava/v3/protocol/lavasession"
	"github.com/lavanet/lava/v3/protocol/metrics"
	"github.com/lavanet/lava/v3/x/pairing/types"
	"google.golang.org/grpc"
)

type ProviderPairingStatusStateQueryInf interface {
	Providers(ctx context.Context, in *types.QueryProvidersRequest, opts ...grpc.CallOption) (*types.QueryProvidersResponse, error)
}

type AvailabilityStatus uint64

const (
	AVAILABLE AvailabilityStatus = iota
	FROZEN
)

type ProviderAvailabilityUpdater struct {
	lock                sync.RWMutex
	latestEpoch         uint64
	pairingQueryClient  ProviderPairingStatusStateQueryInf
	metricsManager      *metrics.ProviderMetricsManager
	clientCtx           client.Context
	rpcProviderEndpoint *lavasession.RPCProviderEndpoint
}

func NewProviderAvailabilityUpdater(
	stateQuery ProviderPairingStatusStateQueryInf,
	rpcProviderEndpoint *lavasession.RPCProviderEndpoint,
	clientCtx client.Context,
	metricsManager *metrics.ProviderMetricsManager,
) *ProviderAvailabilityUpdater {
	return &ProviderAvailabilityUpdater{
		pairingQueryClient:  stateQuery,
		clientCtx:           clientCtx,
		rpcProviderEndpoint: rpcProviderEndpoint,
		metricsManager:      metricsManager,
	}
}

func (pau *ProviderAvailabilityUpdater) UpdateEpoch(epoch uint64) {
	go pau.runProviderAvailabilityUpdate(epoch)
}

func (pau *ProviderAvailabilityUpdater) runProviderAvailabilityUpdate(epoch uint64) {
	pau.lock.Lock()
	defer pau.lock.Unlock()

	// get jail
	if epoch <= pau.latestEpoch {
		return
	}
	ctx := context.Background()

	resultStatus, err := pau.clientCtx.Client.Status(ctx)
	if err != nil {
		return
	}
	currentBlock := resultStatus.SyncInfo.LatestBlockHeight

	response, err := pau.pairingQueryClient.Providers(ctx, &types.QueryProvidersRequest{
		ChainID:    pau.rpcProviderEndpoint.ChainID,
		ShowFrozen: true,
	})
	if err == nil && len(response.StakeEntry) > 0 {
		for _, provider := range response.StakeEntry {
			if !provider.IsAddressVaultOrProvider(provider.Address) {
				continue
			}
			if provider.StakeAppliedBlock > uint64(currentBlock) || provider.Jails > 0 {
				pau.setProviderAvailabilityMetric(FROZEN, provider.Chain, provider.Address)
				continue
			}
			pau.setProviderAvailabilityMetric(AVAILABLE, provider.Chain, provider.Address)
		}
	}
}

func (pau *ProviderAvailabilityUpdater) setProviderAvailabilityMetric(isFrozen AvailabilityStatus, chain string, address string) {
	pau.metricsManager.SetFrozenStatus(float64(isFrozen), chain, address)
}
