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
		// k.Logger(ctx).Error("!!!! bbbbbbxxxxxx !!!!!!!!!")
		return fmt.Errorf("user %s, MaxCU was not found for stake of: %d", clientEntry, clientEntry.Stake.Amount.Int64())
		// panic(fmt.Sprintf("user %s, MaxCU was not found for stake of: %d", clientEntry, clientEntry.Stake.Amount.Int64()))
	}
	allowedCUProvider := allowedCU / k.ServicersToPairCount(ctx)
	if totalCUInEpochForUserProvider > allowedCUProvider {
		return k.LimitClientPairingsAndMarkForPenalty(ctx, chainID, clientEntry, totalCUInEpochForUserProvider, allowedCU, allowedCUProvider, providerAddr)
	}

	return nil
}

func getMaxCULimitsPercentage() (float64, float64) {
	// TODO: Get from param
	slashLimitP, unpayLimitP := 0.1, 0.2 // 10% , 20%
	return slashLimitP, unpayLimitP
}

type ClientOverusedCU struct {
	totalOverusedPercent float64
	providers            map[string]float64
}

func (k Keeper) GetOverusedCUSumPercentage(ctx sdk.Context, chainID string, clientEntry *epochstoragetypes.StakeEntry, providerAddr sdk.AccAddress, totalCUInEpochForUserProvider uint64) (clientOverusedCU ClientOverusedCU, err error) {
	//TODO: Caching will save a lot of time...
	clientAddr := sdk.AccAddress(clientEntry.Address)
	epochLast := k.epochStorageKeeper.GetEpochStart(ctx)
	clientOveruseMap := ClientOverusedCU{0.0, make(map[string]float64)}

	// for every epoch in memory
	for epoch := k.epochStorageKeeper.GetEarliestEpochStart(ctx); epoch <= epochLast; epoch = k.epochStorageKeeper.GetNextEpoch(ctx, epoch) {
		// get epochPayments for this client
		clientPaymentStorage, found := k.GetClientPaymentStorage(ctx, k.GetClientPaymentStorageKey(ctx, chainID, epoch, sdk.AccAddress(clientAddr)))
		if !found { // no payments this epoch, continue + advance epoch
			continue
		}
		// for every unique payment of client for this epoch
		uniquePaymentStoragesClientProvider := clientPaymentStorage.UniquePaymentStorageClientProvider
		for _, uniquePaymentStorageClientProvider := range uniquePaymentStoragesClientProvider {
			paymentProviderAddr := k.GetProviderFromUniquePayment(ctx, *uniquePaymentStorageClientProvider)

			// get current stake of client for this epoch
			currentStakeEntry, stakeErr := k.epochStorageKeeper.GetStakeEntryForClientEpoch(ctx, chainID, clientAddr, epoch)
			if stakeErr != nil {
				panic(fmt.Sprintf("Got payment but not stake - epoch %s for client %s, chainID %s ", epoch, clientAddr, chainID))
			}
			// get allowed of client for this epoch
			allowedCU, allowedCUErr := k.GetAllowedCU(ctx, currentStakeEntry)
			if allowedCUErr != nil {
				panic(fmt.Sprintf("Could not find allowedCU for client %s , epoch %s, stakeEntry %s", clientAddr, epoch, currentStakeEntry))
			}

			// TODO: ? what happens if servicer count changes ? this needs to be based on a selected epoch
			allowedCUProvider := allowedCU / k.ServicersToPairCount(ctx)
			overusedCU := totalCUInEpochForUserProvider - allowedCUProvider
			overusedProviderPercent := float64(overusedCU / allowedCUProvider)
			overusedTotalPercent := float64(overusedCU / allowedCU)

			//TODO: check for negatives

			// var overusedPercent float64 = 0.0
			// var providersCount float64 = 0.0 // Get from param
			// ? do we use UsedCU for this payment or do we calculate this over the sum for this epoch
			// ? do we divide the allowedCU by providerCount ? how do we calc this percentage
			// overusedPercent = float64(uniquePaymentStorageClientProvider.UsedCU) / (float64(allowedCU) / float64(providersCount))
			// TODO: do the math! - calc precentage of overused for allowedCU

			// overused cu found !
			if overusedProviderPercent > 0 {
				// add overused percentage to Total
				clientOveruseMap.totalOverusedPercent += overusedTotalPercent
				// overusedSumTotalP += overusedPercent

				// ? pretty sure no need to filter by provider actually, later will pull this provider specifically
				// if paymentProviderAddr == providerAddr.String() { //TODO: compare AccAddress not strings
				// add overused percentage for the selected Provider
				if val, ok := clientOveruseMap.providers[paymentProviderAddr]; ok {
					//do something here
					clientOveruseMap.providers[paymentProviderAddr] = overusedProviderPercent + val
				} else {
					clientOveruseMap.providers[paymentProviderAddr] = overusedProviderPercent
				}
				// overusedSumProviderP += overusedPercent
				// }
			}
		}
		epoch = k.epochStorageKeeper.GetNextEpoch(ctx, epoch)
	}
	return clientOveruseMap, nil
	// return overusedSumProviderP, overusedSumTotalP, nil
}

func (k Keeper) LimitClientPairingsAndMarkForPenalty(ctx sdk.Context, chainID string, clientEntry *epochstoragetypes.StakeEntry, totalCUInEpochForUserProvider uint64, allowedCU uint64, allowedCUProvider uint64, providerAddr sdk.AccAddress) error {
	implementationFinished := false
	if !implementationFinished {

		k.Logger(ctx).Error("lava_LimitClientPairingsAndMarkForPenalty not fully implemented")
		return fmt.Errorf("lava_LimitClientPairingsAndMarkForPenalty not paying provider")
	} else {

		// ? is this payment already accounted for ? if yes no need for these, if not, sum it up with the rest - use the commented out if statements below
		// currentOverusedProviderPercent := float64(0)
		// overusedCU := totalCUInEpochForUserProvider - allowedCUProvider
		// overusedProviderPercent := float64(overusedCU / allowedCUProvider)
		// overusedTotalPercent := float64(overusedCU / allowedCU)

		slashLimitPercent, unpayLimitPercent := getMaxCULimitsPercentage()
		// overusedSumProviderP, overusedSumTotalP, err := k.GetOverusedCUSumPercentage(ctx, chainID, clientEntry, providerAddr, totalCUInEpochForUserProvider)
		clientOverusedCU, err := k.GetOverusedCUSumPercentage(ctx, chainID, clientEntry, providerAddr, totalCUInEpochForUserProvider)
		if err != nil {
			panic(fmt.Sprintf("user %s, could not calculate overusedCU from memory: %s", clientEntry, clientEntry.Stake.Amount))
		}
		overusedSumTotalPercent := clientOverusedCU.totalOverusedPercent
		overusedSumProviderPercent := clientOverusedCU.providers[providerAddr.String()]
		providerCount := float64(k.ServicersToPairCount(ctx))
		// if overusedSumTotalPercent+currentOverusedProviderPercent > slashLimitP || overusedSumProviderPercent+currentOverusedProviderPercent > slashLimitP/providerCount {
		if overusedSumTotalPercent > slashLimitPercent || overusedSumProviderPercent > slashLimitPercent/providerCount {
			k.SlashUser(ctx, clientEntry.Address)
			// TODO: what about slashing the provider
		}

		// if overusedSumTotalPercent+currentOverusedProviderPercent < unpayLimitP && overusedSumProviderPercent+currentOverusedProviderPercent < unpayLimitP/providerCount {
		if overusedSumTotalPercent < unpayLimitPercent && overusedSumProviderPercent < unpayLimitPercent/providerCount {
			// overuse is under the limit - will allow provider to get payment
			return nil
		}
		// overused is not over the slashLimit, but over the unpayLimit - not paying provider
		return fmt.Errorf("user %s bypassed allowed CU %d by using: %d", clientEntry, allowedCU, totalCUInEpochForUserProvider)
	}
}

func (k Keeper) SlashUser(ctx sdk.Context, clientAddr string) {
	//TODO: jail user, and count problems
}
