package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

func (k Keeper) GetAllowedCU(ctx sdk.Context, entry *epochstoragetypes.StakeEntry) (uint64, error) {
	var allowedCU uint64 = 0
	found := false
	stakeToMaxCUMap := k.StakeToMaxCUList(ctx).List

	for _, stakeToCU := range stakeToMaxCUMap {
		if entry.Stake.IsGTE(stakeToCU.StakeThreshold) {
			allowedCU = stakeToCU.MaxComputeUnits
			found = true
		} else {
			break
		}
	}
	if found {
		return allowedCU, nil
	}
	return 0, nil
}

func (k Keeper) EnforceClientCUsUsageInEpoch(ctx sdk.Context, chainID string, clientEntry *epochstoragetypes.StakeEntry, totalCUInEpochForUserProvider uint64, providerAddr sdk.AccAddress) error {
	allowedCU, err := k.GetAllowedCU(ctx, clientEntry)
	if err != nil {
		panic(fmt.Sprintf("user %s, allowedCU was not found for stake of: %d", clientEntry, clientEntry.Stake.Amount.Int64()))
	}

	if allowedCU == 0 {
		panic(fmt.Sprintf("user %s, MaxCU was not found for stake of: %d", clientEntry, clientEntry.Stake.Amount.Int64()))
	}
	allowedCU = allowedCU / k.ServicersToPairCount(ctx)
	if totalCUInEpochForUserProvider > allowedCU {
		return k.LimitClientPairingsAndMarkForPenalty(ctx, chainID, clientEntry, totalCUInEpochForUserProvider, allowedCU, providerAddr)
	}

	return nil
}

func getMaxCULimitsPercentage() (float64, float64) {
	// TODO: Get from param
	slashLimitP, unpayLimitP := 0.1, 0.2 // 10% , 20%
	return slashLimitP, unpayLimitP
}

func (k Keeper) GetOverusedCUSumPercentage(ctx sdk.Context, chainID string, clientAddr string, providerAddr sdk.AccAddress) (overusedSumTotalP float64, overusedSumProviderP float64, err error) {
	//TODO: Caching will save a lot of time...
	//#omer
	epoch := k.epochStorageKeeper.GetEarliestEpochStart(ctx)
	epochLast := k.epochStorageKeeper.GetEpochStart(ctx)
	// for every epoch in memory
	for epoch < epochLast {
		// epochPayments, found, key := k.GetEpochPaymentsFromBlock(ctx, block)
		epochPayments, found, _ := k.GetEpochPaymentsFromBlock(ctx, epoch)
		if !found { // return fmt.Errorf("did not find any epochPayments for block %d", blockForDelete.Num)
			// return nil,nil
		}
		userPaymentsStorages := epochPayments.ClientsPayments
		// for every epochPayments
		for _, userPaymentStorage := range userPaymentsStorages {
			client, provider, _ := k.DecodeUniquePaymentKey(ctx, userPaymentStorage.Index)
			// only payments by the selected client
			if client == clientAddr { //TODO: compare AccAddress not strings
				uniquePaymentStoragesClientProvider := userPaymentStorage.UniquePaymentStorageClientProvider
				for _, uniquePaymentStorageClientProvider := range uniquePaymentStoragesClientProvider {
					// get current stake - clientStakeEntry (epoch)
					var allowedCU uint64 = 1.0
					// get allowedCU (currentStake)
					var overusedPercent float64 = 0.0
					var providersCount float64 = 0.0 // Get from param
					// ? do we use UsedCU for this payment or do we calculate this over the sum for this epoch
					// ? do we divide the allowedCU by providerCount ? how do we calc this percentage
					overusedPercent = float64(uniquePaymentStorageClientProvider.UsedCU) / (float64(allowedCU) / float64(providersCount))
					// calc precentage of overused for allowedCU
					// overused cu found !
					if overusedPercent > 0 {
						// add overused percentage to Total
						overusedSumTotalP += overusedPercent
						if provider == providerAddr.String() {
							// add overused percentage for the selected Provider
							overusedSumProviderP += overusedPercent
						}
					}
				}
			}
		}
		if false { //code for getting clientStakeEntry
			userStakedEntries, found := k.epochStorageKeeper.GetEpochStakeEntries(ctx, epoch, epochstoragetypes.ClientKey, chainID)
			// loop over stake entries
			for _, clientStakeEntry := range userStakedEntries {
				clientAddr, err := sdk.AccAddressFromBech32(clientStakeEntry.Address)
				if err != nil {
					panic(fmt.Sprintf("invalid user address saved in keeper %s, err: %s", clientStakeEntry.Address, err))
				}
				if clientAddr.Equals(clientAddr) {
					// if clientStakeEntry.Deadline > block {
					// 	//client is not valid for new pairings yet, or was jailed
					// 	return nil, fmt.Errorf("found staked user %+v, but his deadline %d, was bigger than checked block: %d", clientStakeEntry, clientStakeEntry.Deadline, block)
					// }
					print(found)
					// verifiedUser = true
					// clientStakeEntryRet = &clientStakeEntry
					break
				}
			}
		}

		epochBlocks := k.epochStorageKeeper.GetEpochBlocks(ctx, epoch)
		epoch += epochBlocks
	}
	// go over memory and get overused percentages
	// for epoch in memory
	//		for epochPayment - filter by user
	//			payment - > stakeStorage
	// 			stakeCurrent stakeEntry
	// 			, allowedCU
	//			TODO: StakeEntryByAddressFromStorage
	//			TODO: get relevant allowed cu (stake Entry)
	//			TODO: calculate overused percentage
	//			TODO: append to total + append to provider if addresses match
	//
	// this code is useful to go over epochPayments
	// epochPayments, found, key := k.GetEpochPaymentsFromBlock(ctx, blockForDelete)
	// if !found {
	// 	// return fmt.Errorf("did not find any epochPayments for block %d", blockForDelete.Num)
	// 	return nil
	// }
	// userPaymentsStorages := epochPayments.ClientsPayments
	// for _, userPaymentStorage := range userPaymentsStorages {
	// 	uniquePaymentStoragesCliPro := userPaymentStorage.UniquePaymentStorageClientProvider
	// 	for _, uniquePaymentStorageCliPro := range uniquePaymentStoragesCliPro {
	// 		//validate its an old entry, for sanity
	// 		if uniquePaymentStorageCliPro.Block > blockForDelete {
	// 			errMsg := "trying to delete a new entry in epoch payments for block"
	// 			k.Logger(ctx).Error(errMsg)
	// 			panic(errMsg)
	// 		}
	// 		//delete all payment storages
	// 		k.RemoveUniquePaymentStorageClientProvider(ctx, uniquePaymentStorageCliPro.Index)
	// 	}
	// 	k.RemoveClientPaymentStorage(ctx, userPaymentStorage.Index)
	// }
	// k.RemoveEpochPayments(ctx, key)

	return overusedSumProviderP, overusedSumTotalP, nil
}

func (k Keeper) LimitClientPairingsAndMarkForPenalty(ctx sdk.Context, chainID string, clientEntry *epochstoragetypes.StakeEntry, totalCUInEpochForUserProvider uint64, allowedCU uint64, providerAddr sdk.AccAddress) error {
	overused := totalCUInEpochForUserProvider - allowedCU
	overusedP := float64(overused / allowedCU)
	slashLimitP, unpayLimitP := getMaxCULimitsPercentage()
	overusedSumProviderP, overusedSumTotalP, err := k.GetOverusedCUSumPercentage(ctx, chainID, clientEntry.Address, providerAddr)
	if err != nil {
		panic(fmt.Sprintf("user %s, could not calculate overusedCU from memory: %s", clientEntry, clientEntry.Stake.Amount))
	}

	if overusedSumTotalP+overusedP > slashLimitP || overusedSumProviderP+overusedP > slashLimitP/float64(k.ServicersToPairCount(ctx)) {
		k.SlashUser(ctx, clientEntry.Address)
	}

	if overusedSumTotalP+overusedP < unpayLimitP && overusedSumProviderP+overusedP < unpayLimitP/float64(k.ServicersToPairCount(ctx)) {
		return nil // overuse is under the limit - will allow provider to get payment
	}
	// overused is not over the slashLimit, but over the unpayLimit - not paying provider
	return fmt.Errorf("user %s bypassed allowed CU %d by using: %d", clientEntry, allowedCU, totalCUInEpochForUserProvider)
}

func (k Keeper) SlashUser(ctx sdk.Context, clientAddr string) {
	//TODO: jail user, and count problems
}
