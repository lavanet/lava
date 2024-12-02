package updaters

import (
	"context"
	"time"

	"github.com/lavanet/lava/v4/utils"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
	grpc "google.golang.org/grpc"
)

const (
	CallbackKeyForFreezeUpdate = "freeze-update"
)

type ProviderQueryGetter interface {
	GetPairingQueryClient() pairingtypes.QueryClient
}

type ProviderMetricsManagerInf interface {
	SetFrozenStatus(string, bool)
	SetJailStatus(string, bool)
	SetJailedCount(string, uint64)
}

type FrozenStatus uint64

const (
	AVAILABLE FrozenStatus = iota
	FROZEN
)

type ProviderPairingStatusStateQueryInf interface {
	Provider(ctx context.Context, in *pairingtypes.QueryProviderRequest, opts ...grpc.CallOption) (*pairingtypes.QueryProviderResponse, error)
}

type ProviderFreezeJailUpdater struct {
	querier        ProviderPairingStatusStateQueryInf
	metricsManager ProviderMetricsManagerInf
	publicAddress  string
}

func NewProviderFreezeJailUpdater(
	querier ProviderPairingStatusStateQueryInf,
	publicAddress string,
	metricsManager ProviderMetricsManagerInf,
) *ProviderFreezeJailUpdater {
	return &ProviderFreezeJailUpdater{
		querier:        querier,
		publicAddress:  publicAddress,
		metricsManager: metricsManager,
	}
}

func (pfu *ProviderFreezeJailUpdater) UpdateEpoch(epoch uint64) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	response, err := pfu.querier.Provider(ctx, &pairingtypes.QueryProviderRequest{Address: pfu.publicAddress})
	cancel()

	if err != nil {
		utils.LavaFormatError("Failed querying pairing client for provider", err)
		return
	}

	for _, provider := range response.StakeEntries {
		if provider.Address != pfu.publicAddress || !provider.IsAddressVaultOrProvider(provider.Address) {
			// should never happen, but just in case
			continue
		}

		pfu.metricsManager.SetJailedCount(provider.Chain, provider.Jails)
		pfu.metricsManager.SetJailStatus(provider.Chain, provider.IsJailed(time.Now().UTC().Unix()))
		pfu.metricsManager.SetFrozenStatus(provider.Chain, provider.IsFrozen() || provider.StakeAppliedBlock > epoch)
	}
}
