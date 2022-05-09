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

func (k Keeper) EnforceClientCUsUsageInEpoch(ctx sdk.Context, clientEntry *epochstoragetypes.StakeEntry, totalCUInEpochForUserProvider uint64, providerAddr sdk.AccAddress) error {
	allowedCU, err := k.GetAllowedCU(ctx, clientEntry)
	if err != nil {
		panic(fmt.Sprintf("user %s, allowedCU was not found for stake of: %d", clientEntry, clientEntry.Stake.Amount.Int64()))
	}

	if allowedCU == 0 {
		panic(fmt.Sprintf("user %s, MaxCU was not found for stake of: %d", clientEntry, clientEntry.Stake.Amount.Int64()))
	}
	allowedCU = allowedCU / k.ServicersToPairCount(ctx)
	if totalCUInEpochForUserProvider > allowedCU {
		return k.LimitClientPairingsAndMarkForPenalty(ctx, clientEntry, totalCUInEpochForUserProvider, allowedCU, providerAddr)
	}

	return nil
}

func getMaxCULimitsPercentage() (float64, float64) {
	// TODO: Get from param
	slashLimitP, unpayLimitP := 0.1, 0.2 // 10% , 20%
	return slashLimitP, unpayLimitP
}

func (k Keeper) GetOverusedCUSumPercentage(ctx sdk.Context, clientAddr string, providerAddr sdk.AccAddress) (float64, float64, error) {
	overusedSumProviderP, overusedSumTotalP := 0.0, 0.0

	// TODO go over memory and get overused percentages
	// for block/epoch in memory
	//		for epochPayment - filter by user
	// 			stakeCurrent, allowedCU
	//			TODO: StakeEntryByAddressFromStorage
	//			TODO: get relevant allowed cu (stake Entry)
	//			TODO: calculate overused percentage
	//			TODO: append to total + append to provider if addresses match

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

func (k Keeper) LimitClientPairingsAndMarkForPenalty(ctx sdk.Context, clientEntry *epochstoragetypes.StakeEntry, totalCUInEpochForUserProvider uint64, allowedCU uint64, providerAddr sdk.AccAddress) error {
	overused := totalCUInEpochForUserProvider - allowedCU
	overusedP := float64(overused / allowedCU)
	slashLimitP, unpayLimitP := getMaxCULimitsPercentage()
	overusedSumProviderP, overusedSumTotalP, err := k.GetOverusedCUSumPercentage(ctx, clientEntry.Address, providerAddr)
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
