package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
)

// TODO: implement UpdateProviderQos(payments)

// GetQos gets a provider's QoS excellence report from the providerQosFS
func (k Keeper) GetQos(ctx sdk.Context, chainID string, cluster string, provider string) (pairingtypes.QualityOfServiceReport, error) {
	var qos pairingtypes.QualityOfServiceReport
	key := pairingtypes.ProviderQosKey(provider, chainID, cluster)
	found := k.providerQosFS.FindEntry(ctx, key, uint64(ctx.BlockHeight()), &qos)
	if !found {
		return qos, utils.LavaFormatWarning("provider of chain and cluster was not found in the store", fmt.Errorf("qos not found"),
			utils.Attribute{Key: "provider", Value: provider},
			utils.Attribute{Key: "chainID", Value: chainID},
			utils.Attribute{Key: "cluster", Value: cluster},
		)
	}
	return qos, nil
}
