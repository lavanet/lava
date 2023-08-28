package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

func (k Keeper) UpdateProviderQos(epochPayments pairingtypes.EpochPayments) {
}

// GetQos gets a provider's QoS excellence report from the providerQosFS
func (k Keeper) GetQos(ctx sdk.Context, chainID string, cluster string, provider string) pairingtypes.QualityOfServiceReport {
	var qos pairingtypes.QualityOfServiceReport
	key := pairingtypes.ProviderQosKey(provider, chainID, cluster)
	k.providerQosFS.FindEntry(ctx, key, uint64(ctx.BlockHeight()), &qos)
	return qos
}
