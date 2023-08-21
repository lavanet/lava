package keeper

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

func (k Keeper) UpdateProviderQos(epochPayments pairingtypes.EpochPayments) {
}

// GetProviderQosMap uses the providerQosFS to obtain a map of providers and their QoS excellence report
// The map represents the QoS of providers for a specific chainID and a specific cluster
func (k Keeper) GetProviderQosMap(ctx sdk.Context, chainID string, cluster string) map[string]pairingtypes.QualityOfServiceReport {
	providerQosMap := map[string]pairingtypes.QualityOfServiceReport{}
	prefix := strings.Join([]string{chainID, cluster}, "/")
	indices := k.providerQosFS.GetAllEntryIndicesWithPrefix(ctx, prefix)

	for _, ind := range indices {
		var providerQos pairingtypes.ProviderQos
		found := k.providerQosFS.FindEntry(ctx, ind, uint64(ctx.BlockHeight()), &providerQos)
		if !found {
			continue
		}
		providerQosMap[providerQos.Provider] = providerQos.QosExcellence
	}

	return providerQosMap
}

// GetQos gets a provider's QoS excellence report from the providerQosFS
func (k Keeper) GetQos(ctx sdk.Context, chainID string, cluster string, provider string) pairingtypes.QualityOfServiceReport {
	var providerQos pairingtypes.ProviderQos
	key := pairingtypes.ProviderQosKey(provider, chainID, cluster)
	k.providerQosFS.FindEntry(ctx, key, uint64(ctx.BlockHeight()), &providerQos)
	return providerQos.QosExcellence
}
