package keeper

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/tendermint/tendermint/libs/log"
	"golang.org/x/exp/slices"
)

func (k msgServer) UnstakeUnresponsiveProviders(ctx sdk.Context, epochsNumToCheckCUForUnresponsiveProvider uint64, epochsNumToCheckCUForComplainers uint64) error {
	// Get current epoch
	currentEpoch := k.epochStorageKeeper.GetEpochStart(ctx)

	// Get epoch payments relevant to unresponsiveness from previous blocks
	unresponsiveProviders := k.getPaymentsRelatedToUnresponiveness(ctx, currentEpoch, epochsNumToCheckCUForUnresponsiveProvider, epochsNumToCheckCUForComplainers)

	// Get CU serviced by the provider

	// deal with unresponsive providers
	err := k.dealWithUnresponsiveProviders(ctx, relay.UnresponsiveProviders, logger, clientAddr, epochStart, relay.ChainID)
	if err != nil {
		utils.LogLavaEvent(ctx, logger, types.UnresponsiveProviderUnstakeFailedEventName, map[string]string{"err:": err.Error()}, "Error Unresponsive Providers could not unstake")
	}
}

// Function that returns the payments objects relevant to the unresponsiveness punishment condition.
// The relevant payments are all the payments from the previous epoch that are determined by
// epochsNumToCheckCUForUnresponsiveProvider and epochsNumToCheckCUForComplainers.
func (k msgServer) getPaymentsRelatedToUnresponiveness(ctx sdk.Context, currentEpoch uint64, epochsNumToCheckCUForUnresponsiveProvider uint64, epochsNumToCheckCUForComplainers uint64) (uint64, uint64, error) {
	tempEpoch := currentEpoch
	complainerUsedCU := uint64(0)
	providerServicedCU := uint64(0)

	// Find which of the constants is larger to know when to stop iterating
	maxEpochsBack := uint64(math.Max(float64(epochsNumToCheckCUForUnresponsiveProvider), float64(epochsNumToCheckCUForComplainers)))

	// Go over previous epoch payments
	for counter := uint64(0); counter < maxEpochsBack; counter++ {
		// Get previous epoch
		previousEpoch, err := k.epochStorageKeeper.GetPreviousEpochStartForBlock(ctx, tempEpoch)
		if err != nil {
			return 0, 0, err
		}

		// Get epoch payments from block. If the epoch doesn't contain payments, continue
		epochPayments, found, _ := k.GetEpochPaymentsFromBlock(ctx, tempEpoch)
		if !found {
			continue
		}

		if counter < epochsNumToCheckCUForComplainers {
			for _, providerPaymentStorage := range epochPayments.GetClientsPayments() {
				if len(providerPaymentStorage.UnresponsivenessComplaints) == 0 {
					continue
				}
				for _, payment := range providerPaymentStorage.GetUniquePaymentStorageClientProvider() {
					payment.GetUsedCU()
				}
			}
		}

		// Update tempEpoch
		tempEpoch = previousEpoch
	}
	// check if lists are empty (no epochs had payments in them)

	return complainerUsedCU, providerServicedCU, nil
}

func (k msgServer) dealWithUnresponsiveProviders(ctx sdk.Context, unresponsiveData []byte, logger log.Logger, clientAddr sdk.AccAddress, epoch uint64, chainID string) error {
	var unresponsiveProviders []string
	if len(unresponsiveData) == 0 {
		return nil
	}
	err := json.Unmarshal(unresponsiveData, &unresponsiveProviders)
	if err != nil {
		return utils.LavaFormatError("unable to unmarshal unresponsive providers", err, &map[string]string{"UnresponsiveProviders": string(unresponsiveData), "dataLength": strconv.Itoa(len(unresponsiveData))})
	}
	if len(unresponsiveProviders) == 0 {
		// nothing to do.
		return nil
	}
	for _, unresponsiveProvider := range unresponsiveProviders {
		sdkUnresponsiveProviderAddress, err := sdk.AccAddressFromBech32(unresponsiveProvider)
		if err != nil { // if bad data was given, we cant parse it so we ignote it and continue this protects from spamming wrong information.
			utils.LavaFormatError("unable to sdk.AccAddressFromBech32(unresponsive_provider)", err, &map[string]string{"unresponsive_provider_address": unresponsiveProvider})
			continue
		}
		existingEntry, entryExists, indexInStakeStorage := k.epochStorageKeeper.GetStakeEntryByAddressCurrent(ctx, epochstoragetypes.ProviderKey, chainID, sdkUnresponsiveProviderAddress)
		// if !entryExists provider is alraedy unstaked
		if !entryExists {
			continue // if provider is not staked, nothing to do.
		}

		providerStorageKey := k.GetProviderPaymentStorageKey(ctx, chainID, epoch, sdkUnresponsiveProviderAddress)
		providerPaymentStorage, found := k.GetProviderPaymentStorage(ctx, providerStorageKey)

		if !found {
			// currently this provider has not payments in this epoch and also no complaints, we need to add one complaint here
			emptyProviderPaymentStorageWithComplaint := types.ProviderPaymentStorage{
				Index:                              providerStorageKey,
				UniquePaymentStorageClientProvider: []*types.UniquePaymentStorageClientProvider{},
				Epoch:                              epoch,
				UnresponsivenessComplaints:         []string{clientAddr.String()},
			}
			k.SetProviderPaymentStorage(ctx, emptyProviderPaymentStorageWithComplaint)
			continue
		}
		// providerPaymentStorage was found for epoch, start analyzing if unstake is necessary
		// check if the same consumer is trying to complain a second time for this epoch
		if slices.Contains(providerPaymentStorage.UnresponsivenessComplaints, clientAddr.String()) {
			continue
		}

		providerPaymentStorage.UnresponsivenessComplaints = append(providerPaymentStorage.UnresponsivenessComplaints, clientAddr.String())

		// Get the amount of CU used by the complainers (consumers that complained on the provider)
		// complainersCU :=
		for _, complaint := range providerPaymentStorage.UnresponsivenessComplaints {
			complaint
		}

		// Get the amount of CU serviced by the unresponsive provider
		unresponsiveProviderCU, err := k.getTotalCUServicedByProviderFromPreviousEpochs(ctx, epochsNumToCheckCUForUnresponsiveProvider, epoch, chainID, sdkUnresponsiveProviderAddress)

		// now we check if we have more UnresponsivenessComplaints than maxComplaintsPerEpoch
		if len(providerPaymentStorage.UnresponsivenessComplaints) >= maxComplaintsPerEpoch {
			// we check if we have double complaints than previous "collectPaymentsFromNumberOfPreviousEpochs" epochs (including this one) payment requests
			totalPaymentsInPreviousEpochs, err := k.getTotalPaymentsForPreviousEpochs(ctx, collectPaymentsFromNumberOfPreviousEpochs, epoch, chainID, sdkUnresponsiveProviderAddress)
			totalPaymentRequests := totalPaymentsInPreviousEpochs + len(providerPaymentStorage.UniquePaymentStorageClientProvider) // get total number of payments including this epoch
			if err != nil {
				utils.LavaFormatError("lava_unresponsive_providers: couldnt fetch getTotalPaymentsForPreviousEpochs", err, nil)
			} else if totalPaymentRequests*providerPaymentMultiplier < len(providerPaymentStorage.UnresponsivenessComplaints) {
				// unstake provider
				utils.LogLavaEvent(ctx, logger, types.ProviderJailedEventName, map[string]string{"provider_address": sdkUnresponsiveProviderAddress.String(), "chain_id": chainID}, "Unresponsive provider was unstaked from the chain due to unresponsiveness")
				err = k.unSafeUnstakeProviderEntry(ctx, epochstoragetypes.ProviderKey, chainID, indexInStakeStorage, existingEntry)
				if err != nil {
					utils.LavaFormatError("unable to unstake provider entry (unsafe method)", err, &map[string]string{"chainID": chainID, "indexInStakeStorage": strconv.FormatUint(indexInStakeStorage, 10), "existingEntry": existingEntry.GetStake().String()})
					continue
				}
			}
		}
		// set the final provider payment storage state including the complaints
		k.SetProviderPaymentStorage(ctx, providerPaymentStorage)
	}
	return nil
}

func (k msgServer) getTotalCUServicedByProviderFromPreviousEpochs(ctx sdk.Context, numberOfEpochs int, currentEpoch uint64, chainID string, sdkUnresponsiveProviderAddress sdk.AccAddress) (uint64, error) {
	providerServicedCU := uint64(0)
	epochTemp := currentEpoch
	for payment := 0; payment < numberOfEpochs; payment++ {
		previousEpoch, err := k.epochStorageKeeper.GetPreviousEpochStartForBlock(ctx, epochTemp)
		if err != nil {
			return 0, err
		}
		previousEpochProviderStorageKey := k.GetProviderPaymentStorageKey(ctx, chainID, previousEpoch, sdkUnresponsiveProviderAddress)
		previousEpochProviderPaymentStorage, providerPaymentStorageFound := k.GetProviderPaymentStorage(ctx, previousEpochProviderStorageKey)
		if !providerPaymentStorageFound {
			return 0, utils.LavaError(ctx, k.Logger(ctx), "get_provider_payment_storage", map[string]string{"providerStorageKey": fmt.Sprintf("%+v", previousEpochProviderStorageKey), "epoch": fmt.Sprintf("%+v", previousEpoch)}, "couldn't get provider payment storage with given key")
		}
		for _, uniquePayment := range previousEpochProviderPaymentStorage.GetUniquePaymentStorageClientProvider() {
			providerServicedCU += uniquePayment.GetUsedCU()
		}

		epochTemp = previousEpoch
	}
	return providerServicedCU, nil
}

func (k msgServer) unSafeUnstakeProviderEntry(ctx sdk.Context, providerKey string, chainID string, indexInStakeStorage uint64, existingEntry epochstoragetypes.StakeEntry) error {
	err := k.epochStorageKeeper.RemoveStakeEntryCurrent(ctx, epochstoragetypes.ProviderKey, chainID, indexInStakeStorage)
	if err != nil {
		return utils.LavaError(ctx, k.Logger(ctx), "relay_payment_unstake", map[string]string{"existingEntry": fmt.Sprintf("%+v", existingEntry)}, "tried to unstake unsafe but didnt find entry")
	}
	k.epochStorageKeeper.AppendUnstakeEntry(ctx, epochstoragetypes.ProviderKey, existingEntry)
	return nil
}
