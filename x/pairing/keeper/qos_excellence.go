package keeper

import (
	"fmt"

	"cosmossdk.io/math"
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

// AggregateQosExcellence gets a QosExcellenceReport from a relay payment and aggregates it:
//  1. Calculates the Qos score from the report (latency+sync+availability -> sdk.Dec)
//  2. Update ProviderConsumerEpochCu's QosSum with the calculated score and increase QosAmount by 1
func (k Keeper) AggregateQosExcellence(pcec types.ProviderConsumerEpochCu, qos *types.QualityOfServiceReport) (types.ProviderConsumerEpochCu, error) {
	if qos == nil {
		// no QoS report, do nothing
		return pcec, nil
	}

	qosSumToAdd, err := computeQoSExcellenceForPairing(qos)
	if err != nil {
		return pcec, utils.LavaFormatError("failed aggregating QoS excellence", err)
	}

	pcec.QosSum = pcec.QosSum.Add(qosSumToAdd)
	pcec.QosAmount += 1

	return pcec, nil
}

// computeQoSExcellenceForPairing computes the QoS excellence score for the pairing mechanism
// TODO: ComputeQoSExcellence is not the right way to calc, need to implement a different mechanism according to research
func computeQoSExcellenceForPairing(qos *types.QualityOfServiceReport) (math.LegacyDec, error) {
	return qos.ComputeQoSExcellence()
}
