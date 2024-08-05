package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/pairing/types"
)

// AddEpochPayment adds a new epoch payment and returns the updated CU used between provider and project
func (k EpochCuCache) AddEpochPayment(ctx sdk.Context, chainID string, epoch uint64, project string, provider string, cu uint64, sessionID uint64) uint64 {
	// register new epoch session (not checking double spend because it's already checked before calling this function)
	k.SetUniqueEpochSession(ctx, epoch, provider, project, chainID, sessionID)

	// update provider serviced CU
	pec, found := k.GetProviderEpochCuCached(ctx, epoch, provider, chainID)
	if !found {
		pec = types.ProviderEpochCu{ServicedCu: cu}
	} else {
		pec.ServicedCu += cu
	}
	k.SetProviderEpochCuCached(ctx, epoch, provider, chainID, pec)

	// update provider CU for the specific project
	pcec, found := k.GetProviderConsumerEpochCuCached(ctx, epoch, provider, project, chainID)
	if !found {
		pcec = types.ProviderConsumerEpochCu{Cu: cu}
	} else {
		pcec.Cu += cu
	}
	k.SetProviderConsumerEpochCuCached(ctx, epoch, provider, project, chainID, pcec)
	return pcec.Cu
}

// Function to remove epoch payment objects from deleted epochs (older than the chain's memory)
func (k Keeper) RemoveOldEpochPayments(ctx sdk.Context) {
	epochsToDelete := k.epochStorageKeeper.GetDeletedEpochs(ctx)
	for _, epoch := range epochsToDelete {
		k.RemoveAllEpochPaymentsForBlockAppendAdjustments(ctx, epoch)
	}
}

// Function to remove all epoch payments objects from a specific epoch
func (k Keeper) RemoveAllEpochPaymentsForBlockAppendAdjustments(ctx sdk.Context, epochToDelete uint64) {
	// remove all unique epoch sessions
	k.RemoveAllUniqueEpochSession(ctx, epochToDelete)

	// remove all provider epoch cu
	k.RemoveAllProviderEpochCu(ctx, epochToDelete)

	// remove all provider epoch complainer cu
	k.RemoveAllProviderEpochComplainerCu(ctx, epochToDelete)

	pcecs := k.GetAllProviderConsumerEpochCu(ctx, epochToDelete)

	// TODO: update Qos in providerQosFS. new consumers (cluster.subUsage = 0) get default QoS (what is default?)
	consumerUsage := map[string]uint64{}
	type couplingConsumerProvider struct {
		consumer string
		provider string
	}
	// we are keeping the iteration keys to keep determinism when going over the map
	iterationOrder := []couplingConsumerProvider{}
	couplingUsage := map[couplingConsumerProvider]uint64{}
	for _, pcec := range pcecs {
		coupling := couplingConsumerProvider{consumer: pcec.Project, provider: pcec.Provider}
		if _, ok := couplingUsage[coupling]; !ok {
			// only add it if it doesn't exist
			iterationOrder = append(iterationOrder, coupling)
		}
		consumerUsage[pcec.Project] += pcec.ProviderConsumerEpochCu.Cu
		couplingUsage[coupling] += pcec.ProviderConsumerEpochCu.Cu

		k.RemoveProviderConsumerEpochCu(ctx, epochToDelete, pcec.Provider, pcec.Project, pcec.ChainId)
	}
	for _, coupling := range iterationOrder {
		k.subscriptionKeeper.AppendAdjustment(ctx, coupling.consumer, coupling.provider, consumerUsage[coupling.consumer], couplingUsage[coupling])
	}
}
