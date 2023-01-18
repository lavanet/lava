package keeper

import (
	"fmt"
	"strconv"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
)

// This struct's goal is to keep track of the CU required for unresponsiveness for a specific provider and a specific chain ID (the CUs are counted over several epochs)
type providerCuCounterForUnreponsiveness struct {
	providerPaymentStorageIndexList []string // index list of providerPaymentStorage objects that share providerAddress + chainID (but not epoch)
	complainersUsedCu               uint64   // sum of the complainers' CU over epochsNumToCheckCUForComplainers epochs back (start counting from currentEpoch-epochsNumToCheckCUForUnresponsiveProvider)
	providerServicedCu              uint64   // sum of the unresponsive provider's serviced CU over epochsNumToCheckCUForUnresponsiveProvider epochs back (start counting from current epoch)
}

func (k Keeper) UnstakeUnresponsiveProviders(ctx sdk.Context, epochsNumToCheckCUForUnresponsiveProvider uint64, epochsNumToCheckCUForComplainers uint64) error {
	// Get current epoch
	currentEpoch := k.epochStorageKeeper.GetEpochStart(ctx)

	// Get unreponsive providers that should be punished
	providersToPunishMap, err := k.getUnresponsiveProvidersToPunish(ctx, currentEpoch, epochsNumToCheckCUForUnresponsiveProvider, epochsNumToCheckCUForComplainers)
	if err != nil {
		// if the epoch is too early, don't punish and log the event
		if err == types.EpochTooEarlyForUnresponsiveProviderJailError {
			utils.LavaFormatInfo("Epoch too early to punish unresponsive providers", &map[string]string{"epoch": fmt.Sprintf("%+v", currentEpoch)})
			return nil
		}
		return err
	}

	// Punish unresponsive providers (the punishment is currently unstake)
	err = k.punishUnresponsiveProviders(ctx, providersToPunishMap)
	if err != nil {
		return err
	}

	return nil
}

// Function that returns a map that links between a provider that should be punished and its providerCuCounterForUnreponsiveness
func (k Keeper) getUnresponsiveProvidersToPunish(ctx sdk.Context, currentEpoch uint64, epochsNumToCheckCUForUnresponsiveProvider uint64, epochsNumToCheckCUForComplainers uint64) (map[string][]string, error) {
	// check the epochsNum consts
	if epochsNumToCheckCUForComplainers <= 0 || epochsNumToCheckCUForUnresponsiveProvider <= 0 {
		return nil, utils.LavaError(ctx, k.Logger(ctx), "get_unresponsive_providers_to_punish", nil, "epochsNumToCheckCUForUnresponsiveProvider or epochsNumToCheckCUForComplainers are smaller or equal than zero")
	}

	// Get recommendedEpochNumToCollectPayment
	recommendedEpochNumToCollectPayment := k.RecommendedEpochNumToCollectPayment(ctx)

	// To check for punishment, we have to go back (epochsNumToCheckCUForComplainers+epochsNumToCheckCUForUnresponsiveProvider) epochs from the current epoch. If there isn't enough memory, do nothing
	epochTemp := currentEpoch
	for counter := uint64(0); counter < epochsNumToCheckCUForComplainers+epochsNumToCheckCUForUnresponsiveProvider+recommendedEpochNumToCollectPayment; counter++ {
		previousEpoch, err := k.epochStorageKeeper.GetPreviousEpochStartForBlock(ctx, epochTemp)
		if err != nil {
			return nil, types.EpochTooEarlyForUnresponsiveProviderJailError
		}
		epochTemp = previousEpoch
	}

	providerCuCounterForUnreponsivenessMap := make(map[string]providerCuCounterForUnreponsiveness) // map of providerCuCounterForUnreponsiveness keys (string) with providerCuCounterForUnreponsiveness values (providerCuCounterForUnreponsiveness = providerAddress_chainID)
	providersToPunishMap := make(map[string][]string)                                              // map of providerCuCounterForUnreponsiveness keys (string) with list of providerPaymentStorage indices values (string array) (providerCuCounterForUnreponsiveness = providerAddress_chainID)

	// Get the current stake storages (from all chains). stake storages contain a list of stake entries. Each stake storage is for a different chain
	providerStakeStorageList := k.getCurrentProviderStakeStorageList(ctx)
	if len(providerStakeStorageList) == 0 {
		// no provider is staked -> no one to punish
		return nil, utils.LavaError(ctx, k.Logger(ctx), "get_current_provider_stake_storage_list", nil, "no provider is staked, no one to punish")
	}

	// Go back recommendedEpochNumToCollectPayment
	epochTemp = currentEpoch
	for counter := uint64(0); counter < recommendedEpochNumToCollectPayment; counter++ {
		previousEpoch, err := k.epochStorageKeeper.GetPreviousEpochStartForBlock(ctx, epochTemp)
		if err != nil {
			return nil, utils.LavaError(ctx, k.Logger(ctx), "get_previous_epoch", map[string]string{"err": err.Error(), "epoch": fmt.Sprintf("%+v", currentEpoch)}, "couldn't get previous epoch")
		}
		epochTemp = previousEpoch
	}

	// Go over the staked provider entries (on all chains)
	for _, providerStakeStorage := range providerStakeStorageList {
		for _, providerStakeEntry := range providerStakeStorage.GetStakeEntries() {
			// update the CU count for this provider in providerCuCounterForUnreponsivenessMap
			err := k.updateCuForProviderInProviderCuCounterForUnreponsivenessMap(ctx, epochTemp, epochsNumToCheckCUForUnresponsiveProvider, epochsNumToCheckCUForComplainers, providerStakeEntry, providerCuCounterForUnreponsivenessMap)
			if err != nil {
				return nil, utils.LavaError(ctx, k.Logger(ctx), "update_cu_for_provider_in_provider_cu_counter_for_unreponsiveness_map", map[string]string{"err": err.Error()}, "couldn't update the CU count in the providerCuCounterForUnreponsivenessMap")
			}

			// get the provider's providerCuCounterForUnreponsiveness from the providerCuCounterForUnreponsivenessMap
			providerCuCounterForUnreponsiveness, found, providerCuCounterForUnreponsivenessKey := k.getProviderCuCounterForUnreponsivenessEntryFromMap(providerStakeEntry, providerCuCounterForUnreponsivenessMap)
			if !found {
				return nil, utils.LavaError(ctx, k.Logger(ctx), "get_providerCuCounterForUnreponsiveness_entry_from_map", nil, "couldn't get providerCuCounterForUnreponsiveness entry from map")
			}
			// check whether this provider should be punished
			shouldBePunished := k.checkIfProviderShouldBePunishedDueToUnresponsiveness(providerCuCounterForUnreponsiveness)
			if shouldBePunished {
				providersToPunishMap[providerCuCounterForUnreponsivenessKey] = providerCuCounterForUnreponsiveness.providerPaymentStorageIndexList
			}
		}
	}

	return providersToPunishMap, nil
}

// Function to count the CU serviced by the unresponsive provider or the CU of the complainers.
// The function counts CU from <epoch> back. The number of epochs to go back is <epochsNumToGoBack>. The function returns the last visited epoch
func (k Keeper) updateCuForProviderInProviderCuCounterForUnreponsivenessMap(ctx sdk.Context, epoch uint64, epochsNumToCheckCUForUnresponsiveProvider uint64, epochsNumToCheckCUForComplainers uint64, providerStakeEntry epochstoragetypes.StakeEntry, providerCuCounterForUnreponsivenessMap map[string]providerCuCounterForUnreponsiveness) error {
	epochTemp := epoch
	createNewproviderCuCounterForUnreponsivenessEntryFlag := false
	// get the provider's SDK account address
	sdkStakeEntryProviderAddress, err := sdk.AccAddressFromBech32(providerStakeEntry.GetAddress())
	if err != nil {
		return utils.LavaFormatError("unable to sdk.AccAddressFromBech32(provider)", err, &map[string]string{"provider_address": providerStakeEntry.Address})
	}

	// get the provider's providerCuCounterForUnreponsiveness
	providerCuCounterForUnreponsivenessEntry, found, providerCuCounterForUnreponsivenessKey := k.getProviderCuCounterForUnreponsivenessEntryFromMap(providerStakeEntry, providerCuCounterForUnreponsivenessMap)
	if !found {
		// if providerCuCounterForUnreponsivenessEntry is not found, raise a flag to create new entry in providerCuCounterForUnreponsivenessMap
		createNewproviderCuCounterForUnreponsivenessEntryFlag = true
	}

	// count the CU serviced by the unersponsive provider and used CU of the complainers
	for counter := uint64(0); counter < epochsNumToCheckCUForUnresponsiveProvider+epochsNumToCheckCUForComplainers; counter++ {
		// get providerPaymentStorageKey for epochTemp (other traits from the stake entry)
		providerPaymentStorageKey := k.GetProviderPaymentStorageKey(ctx, providerStakeEntry.GetChain(), epochTemp, sdkStakeEntryProviderAddress)

		// try getting providerPaymentStorage using the providerPaymentStorageKey
		providerPaymentStorage, found := k.GetProviderPaymentStorage(ctx, providerPaymentStorageKey)
		if !found {
			// Get previous epoch (from epochTemp)
			previousEpoch, err := k.epochStorageKeeper.GetPreviousEpochStartForBlock(ctx, epochTemp)
			if err != nil {
				return utils.LavaError(ctx, k.Logger(ctx), "get_previous_epoch", map[string]string{"err": err.Error(), "epoch": fmt.Sprintf("%+v", epochTemp)}, "couldn't get previous epoch")
			}
			// update epochTemp
			epochTemp = previousEpoch

			// there is no providerPaymentStorage for this provider in this epoch -> continue searching
			continue
		}

		// counter is smaller than epochsNumToCheckCUForUnresponsiveProvider -> only count CU serviced by the provider in the epoch
		if counter < epochsNumToCheckCUForUnresponsiveProvider {
			// count the CU by iterating through the uniquePaymentStorageClientProvider objects
			cuCount := uint64(0)
			for _, uniquePayment := range providerPaymentStorage.GetUniquePaymentStorageClientProvider() {
				cuCount += uniquePayment.GetUsedCU()
			}

			// update the CU in the providerCuCounterForUnreponsivenessMap
			if createNewproviderCuCounterForUnreponsivenessEntryFlag {
				// providerCuCounterForUnreponsiveness not found -> create a new one and add it to the map
				providerCuCounterForUnreponsivenessMap[providerCuCounterForUnreponsivenessKey] = providerCuCounterForUnreponsiveness{complainersUsedCu: 0, providerServicedCu: cuCount, providerPaymentStorageIndexList: []string{providerPaymentStorage.GetIndex()}}

				// set the createNewproviderCuCounterForUnreponsivenessEntryFlag to false
				createNewproviderCuCounterForUnreponsivenessEntryFlag = false
			} else {
				// providerCuCounterForUnreponsiveness found -> add the cuCount to providerServicedCu and update the map entry
				providerCuCounterForUnreponsivenessEntry.providerServicedCu += cuCount
				providerCuCounterForUnreponsivenessEntry.providerPaymentStorageIndexList = append(providerCuCounterForUnreponsivenessEntry.providerPaymentStorageIndexList, providerPaymentStorage.GetIndex())
				providerCuCounterForUnreponsivenessMap[providerCuCounterForUnreponsivenessKey] = providerCuCounterForUnreponsivenessEntry
			}
		} else {
			// counter is larger than epochsNumToCheckCUForUnresponsiveProvider -> only count complainer CU
			// Get servicersToPair param
			servicersToPair, err := k.ServicersToPairCount(ctx, epoch)
			if err != nil || servicersToPair == 0 {
				return utils.LavaError(ctx, k.Logger(ctx), "get_servicers_to_pair", map[string]string{"err": err.Error(), "epoch": fmt.Sprintf("%+v", epoch)}, "couldn't get servicers to pair")
			}

			// Collect complainers CU (divided by servicersToPair)
			complainersCuCount := providerPaymentStorage.ComplainersTotalCu / (servicersToPair - 1)

			// add the complainersCuCount to complainersUsedCu and update the map entry
			providerCuCounterForUnreponsivenessEntry.complainersUsedCu += complainersCuCount
			providerCuCounterForUnreponsivenessEntry.providerPaymentStorageIndexList = append(providerCuCounterForUnreponsivenessEntry.providerPaymentStorageIndexList, providerPaymentStorage.GetIndex())
			providerCuCounterForUnreponsivenessMap[providerCuCounterForUnreponsivenessKey] = providerCuCounterForUnreponsivenessEntry
		}

		// Get previous epoch (from epochTemp)
		previousEpoch, err := k.epochStorageKeeper.GetPreviousEpochStartForBlock(ctx, epochTemp)
		if err != nil {
			return utils.LavaError(ctx, k.Logger(ctx), "get_previous_epoch", map[string]string{"err": err.Error(), "epoch": fmt.Sprintf("%+v", epochTemp)}, "couldn't get previous epoch")
		}
		// update epochTemp
		epochTemp = previousEpoch
	}
	return nil
}

func (k Keeper) getProviderCuCounterForUnreponsivenessEntryFromMap(providerStakeEntry epochstoragetypes.StakeEntry, providerCuCounterForUnreponsivenessMap map[string]providerCuCounterForUnreponsiveness) (providerCuCounterForUnreponsiveness, bool, string) {
	// get ProviderCuCounterForUnreponsivenessKey
	ProviderCuCounterForUnreponsivenessKey := k.getProviderCuCounterForUnreponsivenessKey(providerStakeEntry.GetAddress(), providerStakeEntry.GetChain())

	// get ProviderCuCounterForUnreponsiveness
	providerCuCounterForUnreponsiveness, found := providerCuCounterForUnreponsivenessMap[ProviderCuCounterForUnreponsivenessKey]
	return providerCuCounterForUnreponsiveness, found, ProviderCuCounterForUnreponsivenessKey
}

func (k Keeper) checkIfProviderShouldBePunishedDueToUnresponsiveness(providerCuCounterForUnreponsiveness providerCuCounterForUnreponsiveness) bool {
	// if the provider served less CU than the sum of the complainers CU -> should be punished
	return providerCuCounterForUnreponsiveness.complainersUsedCu > providerCuCounterForUnreponsiveness.providerServicedCu
}

// Function to create a providerCuCounterForUnreponsivenessKey
func (k Keeper) getProviderCuCounterForUnreponsivenessKey(providerAddress string, chainID string) string {
	return providerAddress + "_" + chainID
}

// Function to extract the chain ID from providerCuCounterForUnreponsivenessKey
func (k Keeper) getChainIDFromProviderCuCounterForUnreponsivenessKey(providerCuCounterForUnreponsivenessKey string) string {
	return strings.Split(providerCuCounterForUnreponsivenessKey, "_")[1]
}

// Function to extract the provider address from providerCuCounterForUnreponsivenessKey
func (k Keeper) getProviderAddressFromProviderCuCounterForUnreponsivenessKey(providerCuCounterForUnreponsivenessKey string) string {
	return strings.Split(providerCuCounterForUnreponsivenessKey, "_")[0]
}

// Function that return the current stake storage for all chains
func (k Keeper) getCurrentProviderStakeStorageList(ctx sdk.Context) []epochstoragetypes.StakeStorage {
	var stakeStorageList []epochstoragetypes.StakeStorage

	// get all chain IDs
	chainIdList := k.specKeeper.GetAllChainIDs(ctx)

	// go over all chain IDs and keep their stake storage. If there is no stake storage for a specific chain, continue to the next one
	for _, chainID := range chainIdList {
		stakeStorage, found := k.epochStorageKeeper.GetStakeStorageCurrent(ctx, epochstoragetypes.ProviderKey, chainID)
		if !found {
			continue
		}
		stakeStorageList = append(stakeStorageList, stakeStorage)
	}

	return stakeStorageList
}

// Function that returns a list of pointers to ProviderPaymentStorage objects by iterating on all the staked providers
func (k Keeper) getProviderPaymentStorageFromStakeStorageList(ctx sdk.Context, epoch uint64, stakeStorageList []epochstoragetypes.StakeStorage) ([]*types.ProviderPaymentStorage, error) {
	var providerPaymentStorageList []*types.ProviderPaymentStorage

	// Go over the provider list
	for _, stakeStorage := range stakeStorageList {
		for _, stakeEntry := range stakeStorage.GetStakeEntries() {
			sdkStakeEntryProviderAddress, err := sdk.AccAddressFromBech32(stakeEntry.Address)
			if err != nil {
				// if bad data was given, we cant parse it so we ignore it and continue this protects from spamming wrong information.
				utils.LavaFormatError("unable to sdk.AccAddressFromBech32(provider)", err, &map[string]string{"provider_address": stakeEntry.Address})
				continue
			}

			// get providerPaymentStorageKey from the stake entry details
			providerPaymentStorageKey := k.GetProviderPaymentStorageKey(ctx, stakeEntry.GetChain(), epoch, sdkStakeEntryProviderAddress)

			// get providerPaymentStorage with providerPaymentStorageKey
			providerPaymentStorage, found := k.GetProviderPaymentStorage(ctx, providerPaymentStorageKey)
			if !found {
				continue
			}

			// add a pointer to the providerPaymentStorage to the providerPaymentStorageList
			providerPaymentStorageList = append(providerPaymentStorageList, &providerPaymentStorage)
		}
	}

	return providerPaymentStorageList, nil
}

// Function that punishes providers. Current punishment is unstake
func (k Keeper) punishUnresponsiveProviders(ctx sdk.Context, providersToPunish map[string][]string) error {
	// if providersToPunish map is empty, do nothing
	if len(providersToPunish) == 0 {
		return nil
	}

	// Go over providers
	for providerCuCounterForUnreponsivenessKey, providerPaymentStorageIndexList := range providersToPunish {
		// extract from providerCuCounterForUnreponsivenessKey the provider address and the chain ID
		providerAddress := strings.Split(providerCuCounterForUnreponsivenessKey, "_")[0]
		chainID := strings.Split(providerCuCounterForUnreponsivenessKey, "_")[1]

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

		// unstake the unresponsive provider
		utils.LogLavaEvent(ctx, k.Logger(ctx), types.ProviderJailedEventName, map[string]string{"provider_address": providerAddress, "chain_id": chainID}, "Unresponsive provider was unstaked from the chain due to unresponsiveness")
		err = k.unsafeUnstakeProviderEntry(ctx, epochstoragetypes.ProviderKey, chainID, indexInStakeStorage, existingEntry)
		if err != nil {
			utils.LavaFormatError("unable to unstake provider entry (unsafe method)", err, &map[string]string{"chainID": chainID, "indexInStakeStorage": strconv.FormatUint(indexInStakeStorage, 10), "existingEntry": existingEntry.GetStake().String()})
		}

		// reset the provider's complainer CU (so he won't get punished for the same complaints twice)
		k.resetComplainersCU(ctx, providerPaymentStorageIndexList)
	}

	return nil
}

// Function that reset the complainer CU of providerPaymentStorage objects that can be accessed using providerPaymentStorageIndexList
func (k Keeper) resetComplainersCU(ctx sdk.Context, providerPaymentStorageIndexList []string) {
	// go over the providerPaymentStorageIndexList
	for _, providerPaymentStorageIndex := range providerPaymentStorageIndexList {
		// get providerPaymentStorage using its index
		providerPaymentStorage, found := k.GetProviderPaymentStorage(ctx, providerPaymentStorageIndex)
		if !found {
			continue
		}

		// reset the complainer CU
		providerPaymentStorage.ComplainersTotalCu = 0
		k.SetProviderPaymentStorage(ctx, providerPaymentStorage)
	}
}

// Function that unstakes a provider (considered unsafe because it doesn't check that the provider is staked. It's ok since we check the provider is staked before we call it)
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
