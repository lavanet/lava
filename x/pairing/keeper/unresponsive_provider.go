package keeper

import (
	"fmt"
	"math"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
)

// Function that returns a map that links between a provider that should be punished and its providerCuCounterForUnreponsiveness
func (k Keeper) UnstakeUnresponsiveProviders(ctx sdk.Context, epochsNumToCheckCUForUnresponsiveProvider, epochsNumToCheckCUForComplainers uint64) {
	// check the epochsNum consts
	if epochsNumToCheckCUForComplainers <= 0 || epochsNumToCheckCUForUnresponsiveProvider <= 0 {
		utils.LavaFormatError("epoch to check CU for unresponsive provider or for complainer is zero",
			fmt.Errorf("invalid unresponsive provider consts"),
			utils.Attribute{Key: "epochsNumToCheckCUForUnresponsiveProvider", Value: epochsNumToCheckCUForUnresponsiveProvider},
			utils.Attribute{Key: "epochsNumToCheckCUForComplainers", Value: epochsNumToCheckCUForComplainers},
		)
	}

	// Get current epoch
	currentEpoch := k.epochStorageKeeper.GetEpochStart(ctx)

	// Get recommendedEpochNumToCollectPayment
	recommendedEpochNumToCollectPayment := k.RecommendedEpochNumToCollectPayment(ctx)

	// check which of the consts is larger
	largerEpochsNumConst := epochsNumToCheckCUForComplainers
	if epochsNumToCheckCUForUnresponsiveProvider > epochsNumToCheckCUForComplainers {
		largerEpochsNumConst = epochsNumToCheckCUForUnresponsiveProvider
	}

	// To check for punishment, we have to go back:
	//   recommendedEpochNumToCollectPayment+
	//   max(epochsNumToCheckCUForComplainers,epochsNumToCheckCUForUnresponsiveProvider)
	// epochs from the current epoch.
	minHistoryBlock, err := k.getBlockEpochsAgo(ctx, currentEpoch, largerEpochsNumConst+recommendedEpochNumToCollectPayment)
	if err != nil {
		// not enough history, do nothing
		return
	}

	// Get the current stake storages (from all chains).
	// Stake storages contain a list of stake entries (each for a different chain).
	providerStakeStorageList := k.getCurrentProviderStakeStorageList(ctx)
	if len(providerStakeStorageList) == 0 {
		// no provider is staked -> no one to punish
		return
	}

	// Go back recommendedEpochNumToCollectPayment
	minPaymentBlock, err := k.getBlockEpochsAgo(ctx, currentEpoch, recommendedEpochNumToCollectPayment)
	if err != nil {
		// not enough history, do nothiing
		return
	}

	// find the minimum number of providers in all the plans
	minProviders := uint64(math.MaxUint64)
	planIndices := k.planKeeper.GetAllPlanIndices(ctx)
	for _, index := range planIndices {
		plan, found := k.planKeeper.FindPlan(ctx, index, uint64(ctx.BlockHeight()))
		if found && plan.PlanPolicy.MaxProvidersToPair < minProviders {
			minProviders = plan.PlanPolicy.MaxProvidersToPair
		}
	}

	// check all supported geolocations form all providers prior to making decisions
	existingProviders := map[int32]map[string]struct{}{}
	for _, providerStakeStorage := range providerStakeStorageList {
		providerStakeEntriesForChain := providerStakeStorage.GetStakeEntries()
		// count providers per geolocation
		for _, providerStakeEntry := range providerStakeEntriesForChain {
			for _, endpoint := range providerStakeEntry.Endpoints {
				_, ok := existingProviders[endpoint.Geolocation]
				if !ok {
					existingProviders[endpoint.Geolocation] = map[string]struct{}{}
				}
				existingProviders[endpoint.Geolocation][providerStakeEntry.Address] = struct{}{}
			}
		}
	}
	// Go over the staked provider entries (on all chains)
	for _, providerStakeStorage := range providerStakeStorageList {
		providerStakeEntriesForChain := providerStakeStorage.GetStakeEntries()
		for _, providerStakeEntry := range providerStakeEntriesForChain {
			if minHistoryBlock < providerStakeEntry.StakeAppliedBlock {
				// this staked provider has too short history (either since staking
				// or since it was last unfrozen) - do not consider for jailing
				continue
			}
			// update the CU count for this provider in providerCuCounterForUnreponsivenessMap
			providerPaymentStorageKeyList, complaintCU, servicedCU, err := k.countCuForUnresponsiveness(ctx, minPaymentBlock, epochsNumToCheckCUForUnresponsiveProvider, epochsNumToCheckCUForComplainers, providerStakeEntry)
			if err != nil {
				utils.LavaFormatError("unstake unresponsive providers failed to count CU", err,
					utils.Attribute{Key: "provider", Value: providerStakeEntry.Address},
				)
				continue
			}

			// providerPaymentStorageKeyList is not empty -> provider should be punished
			if len(providerPaymentStorageKeyList) != 0 && uint64(len(existingProviders[providerStakeEntry.Geolocation])) > minProviders {
				err = k.punishUnresponsiveProvider(ctx, minPaymentBlock, providerPaymentStorageKeyList, providerStakeEntry.GetAddress(), providerStakeEntry.GetChain(), complaintCU, servicedCU)
				delete(existingProviders[providerStakeEntry.Geolocation], providerStakeEntry.Address)
				if err != nil {
					utils.LavaFormatError("unstake unresponsive providers failed to punish provider", err,
						utils.Attribute{Key: "provider", Value: providerStakeEntry.Address},
					)
				}
			}
		}
	}
}

// getBlockEpochsAgo returns the block numEpochs back from the given blockHeight
func (k Keeper) getBlockEpochsAgo(ctx sdk.Context, blockHeight, numEpochs uint64) (uint64, error) {
	for counter := 0; counter < int(numEpochs); counter++ {
		var err error
		blockHeight, err = k.epochStorageKeeper.GetPreviousEpochStartForBlock(ctx, blockHeight)
		if err != nil {
			// too early in the chain life: bail without an error
			return uint64(0), err
		}
	}
	return blockHeight, nil
}

// Function to count the CU serviced by the unresponsive provider and the CU of the complainers. The function returns a list of the found providerPaymentStorageKey
func (k Keeper) countCuForUnresponsiveness(ctx sdk.Context, epoch, epochsNumToCheckCUForUnresponsiveProvider, epochsNumToCheckCUForComplainers uint64, providerStakeEntry epochstoragetypes.StakeEntry) (keysToResetIfJail []string, complaints uint64, servicedCU uint64, errRet error) {
	epochTemp := epoch
	providerServicedCu := uint64(0)
	complainersCu := uint64(0)
	providerPaymentStorageKeyList := []string{}

	// get the provider's SDK account address
	sdkStakeEntryProviderAddress, err := sdk.AccAddressFromBech32(providerStakeEntry.GetAddress())
	if err != nil {
		return nil, 0, 0, utils.LavaFormatError("unable to sdk.AccAddressFromBech32(provider)", err, utils.Attribute{Key: "provider_address", Value: providerStakeEntry.Address})
	}

	// check which of the consts is larger
	largerEpochsNumConst := epochsNumToCheckCUForComplainers
	if epochsNumToCheckCUForUnresponsiveProvider > epochsNumToCheckCUForComplainers {
		largerEpochsNumConst = epochsNumToCheckCUForUnresponsiveProvider
	}

	// count the CU serviced by the unersponsive provider and used CU of the complainers
	for counter := uint64(0); counter < largerEpochsNumConst; counter++ {
		// get providerPaymentStorageKey for epochTemp (other traits from the stake entry)
		providerPaymentStorageKey := k.GetProviderPaymentStorageKey(ctx, providerStakeEntry.GetChain(), epochTemp, sdkStakeEntryProviderAddress)

		// try getting providerPaymentStorage using the providerPaymentStorageKey
		providerPaymentStorage, found := k.GetProviderPaymentStorage(ctx, providerPaymentStorageKey)
		if found {
			// counter is smaller than epochsNumToCheckCUForUnresponsiveProvider -> count CU serviced by the provider in the epoch
			if counter < epochsNumToCheckCUForUnresponsiveProvider {
				// count the CU by iterating through the uniquePaymentStorageClientProvider objects
				for _, uniquePaymentKey := range providerPaymentStorage.GetUniquePaymentStorageClientProviderKeys() {
					uniquePayment, _ := k.GetUniquePaymentStorageClientProvider(ctx, uniquePaymentKey)
					providerServicedCu += uniquePayment.GetUsedCU()
				}
			}

			// counter is smaller than epochsNumToCheckCUForComplainers -> count complainer CU
			if counter < epochsNumToCheckCUForComplainers {
				// update complainersCu
				complainersCu += providerPaymentStorage.ComplainersTotalCu
			}

			// save the providerPaymentStorageKey in the providerPaymentStorageKeyList
			providerPaymentStorageKeyList = append(providerPaymentStorageKeyList, providerPaymentStorageKey)
		}

		// Get previous epoch (from epochTemp)
		previousEpoch, err := k.epochStorageKeeper.GetPreviousEpochStartForBlock(ctx, epochTemp)
		if err != nil {
			return nil, 0, 0, utils.LavaFormatWarning("couldn't get previous epoch", err,
				utils.Attribute{Key: "epoch", Value: epoch},
			)
		}
		// update epochTemp
		epochTemp = previousEpoch
	}

	// the complainers' CU is larger than the provider serviced CU -> should be punished (return providerPaymentStorageKeyList so the complainers' CU can be reset after the punishment)
	if complainersCu > providerServicedCu {
		return providerPaymentStorageKeyList, complainersCu, providerServicedCu, nil
	}

	return nil, complainersCu, providerServicedCu, nil
}

// Function that return the current stake storage for all chains
func (k Keeper) getCurrentProviderStakeStorageList(ctx sdk.Context) []epochstoragetypes.StakeStorage {
	var stakeStorageList []epochstoragetypes.StakeStorage

	// get all chain IDs
	chainIdList := k.specKeeper.GetAllChainIDs(ctx)

	// go over all chain IDs and keep their stake storage. If there is no stake storage for a specific chain, continue to the next one
	for _, chainID := range chainIdList {
		stakeStorage, found := k.epochStorageKeeper.GetStakeStorageCurrent(ctx, chainID)
		if !found {
			continue
		}
		stakeStorageList = append(stakeStorageList, stakeStorage)
	}

	return stakeStorageList
}

// Function that punishes providers. Current punishment is unstake
func (k Keeper) punishUnresponsiveProvider(ctx sdk.Context, epoch uint64, providerPaymentStorageKeyList []string, providerAddress, chainID string, complaintCU uint64, servicedCU uint64) error {
	// Get provider's sdk.Account address
	sdkUnresponsiveProviderAddress, err := sdk.AccAddressFromBech32(providerAddress)
	if err != nil {
		// if bad data was given, we cant parse it so we ignore it and continue this protects from spamming wrong information.
		return utils.LavaFormatError("unable to sdk.AccAddressFromBech32(unresponsive_provider)", err, utils.Attribute{Key: "unresponsive_provider_address", Value: providerAddress})
	}

	// Get provider's stake entry
	existingEntry, entryExists, indexInStakeStorage := k.epochStorageKeeper.GetStakeEntryByAddressCurrent(ctx, chainID, sdkUnresponsiveProviderAddress)
	if !entryExists {
		// if provider is not staked, nothing to do.
		return nil
	}

	// unstake the unresponsive provider
	utils.LogLavaEvent(ctx, k.Logger(ctx), types.ProviderJailedEventName, map[string]string{"provider_address": providerAddress, "chain_id": chainID, "complaint_cu": strconv.FormatUint(complaintCU, 10), "serviced_cu": strconv.FormatUint(servicedCU, 10)}, "Unresponsive provider was unstaked from the chain due to unresponsiveness")
	err = k.unsafeUnstakeProviderEntry(ctx, epoch, chainID, indexInStakeStorage, existingEntry)
	if err != nil {
		utils.LavaFormatError("unable to unstake provider entry (unsafe method)", err, []utils.Attribute{{Key: "chainID", Value: chainID}, {Key: "indexInStakeStorage", Value: indexInStakeStorage}, {Key: "existingEntry", Value: existingEntry.GetStake()}}...)
	}

	// reset the provider's complainer CU (so he won't get punished for the same complaints twice)
	k.resetComplainersCU(ctx, providerPaymentStorageKeyList)

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
func (k Keeper) unsafeUnstakeProviderEntry(ctx sdk.Context, epoch uint64, chainID string, indexInStakeStorage uint64, existingEntry epochstoragetypes.StakeEntry) error {
	// Remove the provider's stake entry
	err := k.epochStorageKeeper.RemoveStakeEntryCurrent(ctx, chainID, indexInStakeStorage)
	if err != nil {
		return utils.LavaFormatWarning("tried to unstake unsafe but didnt find entry", err,
			utils.Attribute{Key: "existingEntry", Value: fmt.Sprintf("%+v", existingEntry)},
		)
	}

	// get unstakeHoldBlocks
	unstakeHoldBlocks := k.epochStorageKeeper.UnstakeHoldBlocks(ctx, epoch)

	// Appened the provider's stake entry to the unstake entry list
	k.epochStorageKeeper.AppendUnstakeEntry(ctx, existingEntry, unstakeHoldBlocks)

	return nil
}
