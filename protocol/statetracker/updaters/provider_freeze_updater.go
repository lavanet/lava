package updaters

import (
	"context"
	"sync"
	"time"

	"github.com/lavanet/lava/v3/utils"
	"github.com/lavanet/lava/v3/x/pairing/types"
	"google.golang.org/grpc"
)

const (
	CallbackKeyForFreezeUpdate = "freeze-update"
)

type ProviderPairingStatusStateQueryInf interface {
	Providers(ctx context.Context, in *types.QueryProvidersRequest, opts ...grpc.CallOption) (*types.QueryProvidersResponse, error)
}

type ProviderMetricsManagerInf interface {
	SetFrozenStatus(float64, string, string)
	SetJailedStatus(uint64, string, string)
}

type FrozenStatus uint64

const (
	AVAILABLE FrozenStatus = iota
	FROZEN
)

type ProviderFreezeUpdater struct {
	lock               sync.RWMutex
	latestEpoch        uint64
	pairingQueryClient ProviderPairingStatusStateQueryInf
	metricsManager     ProviderMetricsManagerInf
	chainId            string
	publicAddress      string
}

func NewProviderFreezeUpdater(
	stateQuery ProviderPairingStatusStateQueryInf,
	chainId string,
	publicAddress string,
	metricsManager ProviderMetricsManagerInf,
) *ProviderFreezeUpdater {
	return &ProviderFreezeUpdater{
		pairingQueryClient: stateQuery,
		chainId:            chainId,
		publicAddress:      publicAddress,
		metricsManager:     metricsManager,
		latestEpoch:        0,
	}
}

func (pfu *ProviderFreezeUpdater) UpdaterKey() string {
	return CallbackKeyForSpecUpdate + pfu.chainId
}

func (pfu *ProviderFreezeUpdater) UpdateEpoch(epoch uint64) {
	pfu.lock.Lock()
	defer pfu.lock.Unlock()

	if epoch <= pfu.latestEpoch {
		return
	}
	pfu.latestEpoch = epoch
	ctx := context.Background()

	response, err := pfu.pairingQueryClient.Providers(ctx, &types.QueryProvidersRequest{
		ChainID:    pfu.chainId,
		ShowFrozen: true,
	})
	if err != nil {
		utils.LavaFormatError("Failed querying pairing client for providers", err, utils.LogAttr("chainId", pfu.chainId))
		return
	}
	for _, provider := range response.StakeEntry {
		if provider.Address != pfu.publicAddress || !provider.IsAddressVaultOrProvider(provider.Address) {
			continue
		}

		pfu.metricsManager.SetJailedStatus(provider.Jails, provider.Chain, provider.Address)
		if provider.StakeAppliedBlock > epoch || provider.IsJailed(time.Now().UTC().Unix()) {
			pfu.setProviderFreezeMetric(FROZEN, provider.Chain, provider.Address)
			continue
		}
		pfu.setProviderFreezeMetric(AVAILABLE, provider.Chain, provider.Address)
	}
}

func (pfu *ProviderFreezeUpdater) setProviderFreezeMetric(isFrozen FrozenStatus, chain string, address string) {
	pfu.metricsManager.SetFrozenStatus(float64(isFrozen), chain, address)
}
