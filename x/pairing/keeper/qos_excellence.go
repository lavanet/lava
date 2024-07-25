package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/pairing/types"
)

// TODO: implement UpdateProviderQos(payments)

// GetQos gets a provider's QoS excellence report from the providerQosFS
func (k Keeper) GetQos(ctx sdk.Context, chainID string, cluster string, provider string) (types.QualityOfServiceReport, error) {
	var qos types.QualityOfServiceReport
	key := types.ProviderQosKey(provider, chainID, cluster)
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

// SetProviderClusterQos sets a ProviderClusterQos in the store
func (k Keeper) SetProviderClusterQos(ctx sdk.Context, chainID string, cluster string, provider string, pcq types.ProviderClusterQos) {
	key := collections.Join3(chainID, cluster, provider)
	err := k.providerClusterQos.Set(ctx, key, pcq)
	if err != nil {
		panic(err)
	}
	chainClusterKey := collections.Join(chainID, cluster)
	err = k.chainClusterQosKeys.Set(ctx, chainClusterKey)
	if err != nil {
		panic(err)
	}
}

// ConvertQosScoreToPairingQosScore get a list of ProviderClusterQos with the same chain+cluster and compares
// between all the providers to calculate a QoS pairing score (ranged between [0.5-2])
func (k Keeper) ConvertQosScoreToPairingQosScore(provider string, pcq types.ProviderClusterQos) types.QosPairingScore {
	score := pcq.Score.Score.Num.Quo(pcq.Score.Score.Denom)
	return types.QosPairingScore{Score: score}
}

// SetQosPairingScore sets the QoS pairing score in the provider QoS fixation store
func (k Keeper) SetQosPairingScore(ctx sdk.Context, chainID string, cluster string, provider string, qps types.QosPairingScore) error {
	return k.providerQosFS.AppendEntry(ctx, chainID+" "+cluster+" "+provider, uint64(ctx.BlockHeight()), &qps)
}
