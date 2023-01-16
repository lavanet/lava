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

type punishEntry struct {
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

// Function that returns a map that links between a provider that should be punished and its punishEntry
func (k Keeper) getUnresponsiveProvidersToPunish(ctx sdk.Context, currentEpoch uint64, epochsNumToCheckCUForUnresponsiveProvider uint64, epochsNumToCheckCUForComplainers uint64) (map[string][]string, error) {
	// check the epochsNum consts
	if epochsNumToCheckCUForComplainers <= 0 || epochsNumToCheckCUForUnresponsiveProvider <= 0 {
		return nil, utils.LavaError(ctx, k.Logger(ctx), "get_unresponsive_providers_to_punish", nil, "epochsNumToCheckCUForUnresponsiveProvider or epochsNumToCheckCUForComplainers are smaller or equal than zero")
	}

	providersPunishEntryMap := make(map[string]punishEntry) // map of punishEntryKey keys (string) with punishEntry values (punishEntryKey = providerAddress_chainID)
	providersToPunishMap := make(map[string][]string)       // map of punishEntryKey keys (string) with list of providerPaymentStorage indices values (string array) (punishEntryKey = providerAddress_chainID)

	// Get recommendedEpochNumToCollectPayment
	recommendedEpochNumToCollectPayment := k.RecommendedEpochNumToCollectPayment(ctx)

	// handle very early epochs (basically, don't punish)
	epochBlocks, err := k.epochStorageKeeper.EpochBlocks(ctx, currentEpoch)
	if err != nil {
		return nil, utils.LavaError(ctx, k.Logger(ctx), "get_epoch_blocks", map[string]string{"err": err.Error(), "epoch": fmt.Sprintf("%+v", currentEpoch)}, "couldn't get epoch blocks param")
	}
	if currentEpoch < (epochsNumToCheckCUForComplainers+epochsNumToCheckCUForUnresponsiveProvider+recommendedEpochNumToCollectPayment)*epochBlocks {
		// To check for punishment, we have to go back (epochsNumToCheckCUForComplainers+epochsNumToCheckCUForUnresponsiveProvider) epochs from the current epoch. If there isn't enough memory, do nothing
		return nil, types.EpochTooEarlyForUnresponsiveProviderJailError
	}

	// Go back recommendedEpochNumToCollectPayment
	for counter := uint64(0); counter < recommendedEpochNumToCollectPayment; counter++ {
		previousEpoch, err := k.epochStorageKeeper.GetPreviousEpochStartForBlock(ctx, currentEpoch)
		if err != nil {
			return nil, utils.LavaError(ctx, k.Logger(ctx), "get_previous_epoch", map[string]string{"err": err.Error(), "epoch": fmt.Sprintf("%+v", currentEpoch)}, "couldn't get previous epoch")
		}
		currentEpoch = previousEpoch
	}

	// Go over previous epochs to count the CU serviced by the provider
	epochAfterCountingProviderCu, err := k.countCuForUnresponsiveProviderAndComplainers(ctx, true, currentEpoch, epochsNumToCheckCUForUnresponsiveProvider, providersPunishEntryMap)
	if err != nil {
		return nil, utils.LavaError(ctx, k.Logger(ctx), "count_cu_unresponsive_provider", map[string]string{"err": err.Error()}, "couldn't count the CU of the unresponsive provider")
	}

	// Go over previous epochs (from the point the last loop finished) to count the CU of clients that complained about the provider
	_, err = k.countCuForUnresponsiveProviderAndComplainers(ctx, false, epochAfterCountingProviderCu, epochsNumToCheckCUForComplainers, providersPunishEntryMap)
	if err != nil {
		return nil, utils.LavaError(ctx, k.Logger(ctx), "count_cu_complainers", map[string]string{"err": err.Error()}, "couldn't count the CU of the complainers")
	}

	// Get providers that should be punished
	for punishEntryKey, punishEntry := range providersPunishEntryMap {
		// provider doesn't have payments (serviced CU = 0)
		if punishEntry.providerServicedCu == 0 {
			chainID := strings.Split(punishEntryKey, "_")[1]

			// Try getting the provider's stake entry
			_, foundStakeEntry := k.epochStorageKeeper.GetEpochStakeEntries(ctx, currentEpoch, epochstoragetypes.ProviderKey, chainID)

			// if stake entry is not found -> this is the provider's first epoch (didn't get a chance to stake yet) -> don't punish him
			if !foundStakeEntry {
				continue
			}
		}

		// The provider served less CU than the sum of the complainers CU -> should be punished
		if punishEntry.complainersUsedCu > punishEntry.providerServicedCu {
			providersToPunishMap[punishEntryKey] = punishEntry.providerPaymentStorageIndexList
		}
	}

	return providersToPunishMap, nil
}

// Function to count the CU serviced by the unresponsive provider or the CU of the complainers.
// The function counts CU from <epoch> back. The number of epochs to go back is <epochsNumToGoBack>. The function returns the last visited epoch
func (k Keeper) countCuForUnresponsiveProviderAndComplainers(ctx sdk.Context, isProvider bool, epoch uint64, epochsNumToGoBack uint64, providersPunishEntryMap map[string]punishEntry) (uint64, error) {
	epochTemp := epoch

	for counter := uint64(0); counter < epochsNumToGoBack; counter++ {
		// Get previous epoch (from epochTemp)
		previousEpoch, err := k.epochStorageKeeper.GetPreviousEpochStartForBlock(ctx, epochTemp)
		if err != nil {
			return 0, utils.LavaError(ctx, k.Logger(ctx), "get_previous_epoch", map[string]string{"err": err.Error(), "epoch": fmt.Sprintf("%+v", epochTemp)}, "couldn't get previous epoch")
		}

		// Get the current stake storages (from all chains)
		stakeStorageList := k.getCurrentProviderStakeStorageList(ctx)
		if len(stakeStorageList) == 0 {
			// no provider is staked -> CU = 0
			return 0, nil
		}

		// Get a providerPaymentStorage of all staked providers
		providerPaymentStorageList, err := k.getProviderPaymentStorageFromStakeStorageList(ctx, previousEpoch, stakeStorageList)
		if err != nil {
			return 0, utils.LavaError(ctx, k.Logger(ctx), "get_provider_payment_storage_from_stake_storage_list", map[string]string{"err": err.Error(), "epoch": fmt.Sprintf("%+v", previousEpoch)}, "couldn't get providerPaymentStorageList from stakeStorageList")
		}

		// Go over providerPaymentStorage list
		for _, providerPaymentStorage := range providerPaymentStorageList {
			cuCount := uint64(0)

			// Get provider address and chainID
			providerAddress := k.getProviderAddressFromProviderPaymentStorageKey(providerPaymentStorage.Index)
			chainID := k.getChainIDFromProviderPaymentStorageKey(providerPaymentStorage.Index)

			// Create a punishEntryKey from the provider's address and the chain ID
			punishEntryKey := providerAddress + "_" + chainID

			// find the provider's punishEntry in map
			providerPunishEntry, punishEntryFound := providersPunishEntryMap[punishEntryKey]

			if isProvider {
				// if isProvider is true, we count the CU serviced by the unersponsive provider
				for _, uniquePayment := range providerPaymentStorage.GetUniquePaymentStorageClientProvider() {
					cuCount += uniquePayment.GetUsedCU()
				}
				if punishEntryFound {
					// punishEntry found -> add the cuCount to providerServicedCu and update the map entry
					providerPunishEntry.providerServicedCu += cuCount
					providersPunishEntryMap[punishEntryKey] = providerPunishEntry
				} else {
					// punishEntry not found -> create a new one and add it to the map
					providersPunishEntryMap[punishEntryKey] = punishEntry{complainersUsedCu: 0, providerServicedCu: cuCount, providerPaymentStorageIndexList: []string{providerPaymentStorage.Index}}
				}
			} else {
				// Get servicersToPair param
				servicersToPair, err := k.ServicersToPairCount(ctx, epoch)
				if err != nil || servicersToPair == 0 {
					return 0, utils.LavaError(ctx, k.Logger(ctx), "get_servicers_to_pair", map[string]string{"err": err.Error(), "epoch": fmt.Sprintf("%+v", epoch)}, "couldn't get servicers to pair")
				}

				// Collect complainers CU (divided by servicersToPair)
				cuCount = providerPaymentStorage.ComplainersTotalCu / (servicersToPair - 1)

				if punishEntryFound {
					// punishEntry found -> add the cuCount to complainersUsedCu and update the map entry
					providerPunishEntry.complainersUsedCu += cuCount
					providersPunishEntryMap[punishEntryKey] = providerPunishEntry
				} else {
					// punishEntry not found -> create a new one and add it to the map
					providersPunishEntryMap[punishEntryKey] = punishEntry{complainersUsedCu: cuCount, providerServicedCu: 0, providerPaymentStorageIndexList: []string{providerPaymentStorage.Index}}
				}
			}
			// add the providerPaymentStorage index to the providerPaymentStorageIndexList
			providerPunishEntry.providerPaymentStorageIndexList = append(providerPunishEntry.providerPaymentStorageIndexList, providerPaymentStorage.Index)
		}

		// update epochTemp
		epochTemp = previousEpoch
	}

	return epochTemp, nil
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
	for punishEntryKey, providerPaymentStorageIndexList := range providersToPunish {
		// extract from punishEntryKey the provider address and the chain ID
		providerAddress := strings.Split(punishEntryKey, "_")[0]
		chainID := strings.Split(punishEntryKey, "_")[1]

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
