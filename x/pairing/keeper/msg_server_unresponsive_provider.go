package keeper

import (
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
)

type punishEntry struct {
	chainID            string
	complainersUsedCu  uint64
	providerServicedCu uint64
}

func (k Keeper) UnstakeUnresponsiveProviders(ctx sdk.Context, epochsNumToCheckCUForUnresponsiveProvider uint64, epochsNumToCheckCUForComplainers uint64) error {
	// Get current epoch
	currentEpoch := k.epochStorageKeeper.GetEpochStart(ctx)

	// Get unreponsive providers that should be punished
	providersToPunishMap, err := k.getUnresponsiveProvidersToPunish(ctx, currentEpoch, epochsNumToCheckCUForUnresponsiveProvider, epochsNumToCheckCUForComplainers)
	if err != nil {
		return err
	}

	// Punish unresponsive providers (currently unstake)
	err = k.punishUnresponsiveProviders(ctx, providersToPunishMap)
	if err != nil {
		return err
	}

	return nil
}

// Function that punishes providers. Current punishment is unstake
func (k Keeper) punishUnresponsiveProviders(ctx sdk.Context, providersToPunish map[string]string) error {
	// if providersToPunish map is empty, do nothing
	if len(providersToPunish) == 0 {
		return nil
	}

	// Go over providers
	for providerAddress, chainID := range providersToPunish {
		// Get provider's sdk.Account address
		sdkUnresponsiveProviderAddress, err := sdk.AccAddressFromBech32(providerAddress)
		if err != nil {
			// if bad data was given, we cant parse it so we ignore it and continue this protects from spamming wrong information.
			utils.LavaFormatError("unable to sdk.AccAddressFromBech32(unresponsive_provider)", err, &map[string]string{"unresponsive_provider_address": providerAddress})
			continue
		}

		// Get provider's stake entry
		existingEntry, entryExists, indexInStakeStorage := k.epochStorageKeeper.GetStakeEntryByAddressCurrent(ctx, epochstoragetypes.ProviderKey, chainID, sdkUnresponsiveProviderAddress)
		if !entryExists {
			// if provider is not staked, nothing to do.
			continue
		}

		utils.LogLavaEvent(ctx, k.Logger(ctx), types.ProviderJailedEventName, map[string]string{"provider_address": providerAddress, "chain_id": chainID}, "Unresponsive provider was unstaked from the chain due to unresponsiveness")
		err = k.unsafeUnstakeProviderEntry(ctx, epochstoragetypes.ProviderKey, chainID, indexInStakeStorage, existingEntry)
		if err != nil {
			utils.LavaFormatError("unable to unstake provider entry (unsafe method)", err, &map[string]string{"chainID": chainID, "indexInStakeStorage": strconv.FormatUint(indexInStakeStorage, 10), "existingEntry": existingEntry.GetStake().String()})
		}
	}

	return nil
}

// Function that returns a map that links between a provider that should be punished and its punishEntry
func (k Keeper) getUnresponsiveProvidersToPunish(ctx sdk.Context, currentEpoch uint64, epochsNumToCheckCUForUnresponsiveProvider uint64, epochsNumToCheckCUForComplainers uint64) (map[string]string, error) {
	// check the epochsNum consts
	if epochsNumToCheckCUForComplainers <= 0 || epochsNumToCheckCUForUnresponsiveProvider <= 0 {
		return nil, utils.LavaError(ctx, k.Logger(ctx), "get_unresponsive_providers_to_punish", nil, "epochsNumToCheckCUForUnresponsiveProvider or epochsNumToCheckCUForComplainers are smaller or equal than zero")
	}

	providersPunishEntryMap := make(map[string]punishEntry)
	providersToPunishMap := make(map[string]string)
	epochTemp := currentEpoch

	// Get servicersToPair param
	servicersToPair, err := k.ServicersToPairCount(ctx, currentEpoch)
	if err != nil || servicersToPair == 0 {
		return nil, utils.LavaError(ctx, k.Logger(ctx), "get_servicers_to_pair", map[string]string{"err": err.Error(), "epoch": fmt.Sprintf("%+v", currentEpoch)}, "couldn't get servicers to pair")
	}

	// Go over previous epochs
	counter := uint64(0)
	for {
		// We've finished going back epochs, break the loop
		if counter >= epochsNumToCheckCUForComplainers && counter >= epochsNumToCheckCUForUnresponsiveProvider {
			break
		}

		// Get previous epoch (from epochTemp)
		previousEpoch, err := k.epochStorageKeeper.GetPreviousEpochStartForBlock(ctx, epochTemp)
		if err != nil {
			return nil, utils.LavaError(ctx, k.Logger(ctx), "get_previous_epoch", map[string]string{"err": err.Error(), "epoch": fmt.Sprintf("%+v", epochTemp)}, "couldn't get previous epoch")
		}

		// Get previous epoch's payments
		epochPayments, found, _ := k.GetEpochPaymentsFromBlock(ctx, previousEpoch)
		if !found {
			epochTemp = previousEpoch
			counter++
			continue
		}

		// Get providerPaymentStorage list
		providerPaymentStorageList := epochPayments.GetClientsPayments()

		// Go over providerPaymentStorage
		for _, providerPaymentStorage := range providerPaymentStorageList {
			providerServicedCU := uint64(0)
			complainersUsedCu := uint64(0)

			// Collect serviced CU for provider
			if counter < epochsNumToCheckCUForUnresponsiveProvider {
				// Go over the unique payments collected by the provider. Sum the serviced CU
				for _, uniquePayment := range providerPaymentStorage.GetUniquePaymentStorageClientProvider() {
					providerServicedCU += uniquePayment.GetUsedCU()
				}
			}

			// Collect complainers CU (divided by servicersToPair)
			if counter < epochsNumToCheckCUForComplainers {
				complainersUsedCu = providerPaymentStorage.ComplainersTotalCu / (servicersToPair - 1)
			}

			// Get provider address
			providerAddress := k.getProviderAddressFromProviderPaymentStorageKey(providerPaymentStorage.Index)
			chainID := k.getChainIDFromProviderPaymentStorageKey(providerPaymentStorage.Index)

			// find the provider's punishEntry in map
			providerPunishEntry, found := providersPunishEntryMap[providerAddress]

			// update provider's punish entry
			if found {
				providerPunishEntry.complainersUsedCu += complainersUsedCu
				providerPunishEntry.providerServicedCu += providerServicedCU
			} else {
				providersPunishEntryMap[providerAddress] = punishEntry{complainersUsedCu: complainersUsedCu, providerServicedCu: providerServicedCU, chainID: chainID}
			}
		}

		// update epochTemp
		epochTemp = previousEpoch
	}

	// Get providers that should be punished
	for providerAddress, punishEntry := range providersPunishEntryMap {
		// provider doesn't have payments (serviced CU = 0)
		if punishEntry.providerServicedCu == 0 {
			// Try getting the provider's stake entry
			_, foundStakeEntry := k.epochStorageKeeper.GetEpochStakeEntries(ctx, currentEpoch, epochstoragetypes.ProviderKey, punishEntry.chainID)

			// if stake entry is not found -> this is provider's first epoch (didn't get a chance to stake yet) -> don't punish him
			if !foundStakeEntry {
				continue
			}
		}

		// provider's served less CU than the sum of the complainers CU -> should be punished
		if punishEntry.complainersUsedCu > punishEntry.providerServicedCu {
			providersToPunishMap[providerAddress] = punishEntry.chainID
		}
	}

	return providersToPunishMap, nil
}

// Function that unstakes a provider
func (k Keeper) unsafeUnstakeProviderEntry(ctx sdk.Context, providerKey string, chainID string, indexInStakeStorage uint64, existingEntry epochstoragetypes.StakeEntry) error {
	// Remove the provider's stake entry
	err := k.epochStorageKeeper.RemoveStakeEntryCurrent(ctx, epochstoragetypes.ProviderKey, chainID, indexInStakeStorage)
	if err != nil {
		return utils.LavaError(ctx, k.Logger(ctx), "relay_payment_unstake", map[string]string{"existingEntry": fmt.Sprintf("%+v", existingEntry)}, "tried to unstake unsafe but didnt find entry")
	}

	// Appened the provider's stake entry to the unstake entry list
	k.epochStorageKeeper.AppendUnstakeEntry(ctx, epochstoragetypes.ProviderKey, existingEntry)

	return nil
}
