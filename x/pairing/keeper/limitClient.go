package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
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

// TODO move to types - #O where would you like these to go ?
type ClientUsedCU struct {
	TotalOverused uint64
	Providers     map[string]uint64
}
type ClientOverusedCUPercent struct {
	TotalOverusedPercent float64
	Providers            map[string]float64
}

type ClientProviderOverusedCUPercent struct {
	TotalOverusedPercent    float64
	OverusedPercentProvider float64
}

func (k Keeper) GetEpochClientProviderUsedCUMap(ctx sdk.Context, clientPaymentStorage types.ClientPaymentStorage) (clientOverusedCUMap ClientUsedCU, err error) {
	clientOverusedCUMap = ClientUsedCU{0, make(map[string]uint64)}
	// for every unique payment of client for this epoch
	uniquePaymentStoragesClientProviderList := clientPaymentStorage.UniquePaymentStorageClientProvider
	for _, uniquePaymentStorageClientProvider := range uniquePaymentStoragesClientProviderList {
		paymentProviderAddr := k.GetProviderFromUniquePayment(ctx, *uniquePaymentStorageClientProvider)
		clientOverusedCUMap.TotalOverused += uniquePaymentStorageClientProvider.UsedCU
		if _, ok := clientOverusedCUMap.Providers[paymentProviderAddr]; ok {
			clientOverusedCUMap.Providers[paymentProviderAddr] += uniquePaymentStorageClientProvider.UsedCU
		} else {
			clientOverusedCUMap.Providers[paymentProviderAddr] = uniquePaymentStorageClientProvider.UsedCU
		}
	}

	return
}
func (k Keeper) GetAllowedCUClientEpoch(ctx sdk.Context, chainID string, epoch uint64, clientAddr sdk.AccAddress) (allowedCU uint64, err error) {
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
	return

}
func (k Keeper) GetOverusedFromUsedCU(ctx sdk.Context, clientProvidersEpochUsedCUMap ClientUsedCU, allowedCU uint64, providerAddr sdk.AccAddress) (clientOverusedCUPercent ClientOverusedCUPercent, err error) {
	clientOverusedCUPercent = ClientOverusedCUPercent{0.0, make(map[string]float64)}
	// TODO: ServicersToPairCount needs epoch
	allowedCUProvider := allowedCU / k.ServicersToPairCount(ctx)
	for providerID, usedCU := range clientProvidersEpochUsedCUMap.Providers {
		overusedCU := usedCU - allowedCUProvider
		overusedProviderPercent := float64(overusedCU / allowedCUProvider)
		overusedTotalPercent := float64(overusedCU / allowedCU)
		// overused cu found !
		if overusedProviderPercent > 0 {
			// add overused percentage to Total
			clientOverusedCUPercent.TotalOverusedPercent += overusedTotalPercent
			// add overused percentage for the selected Provider
			if val, ok := clientOverusedCUPercent.Providers[providerID]; ok {
				clientOverusedCUPercent.Providers[providerID] = overusedProviderPercent + val
			} else {
				clientOverusedCUPercent.Providers[providerID] = overusedProviderPercent
			}
		}
	}
	//TODO: check for negatives ?
	return
}

func (k Keeper) GetOverusedCUSumPercentage(ctx sdk.Context, chainID string, clientEntry *epochstoragetypes.StakeEntry, providerAddr sdk.AccAddress, totalCUInEpochForUserProvider uint64) (clientProviderOverusedCUPercent ClientProviderOverusedCUPercent, err error) {
	//TODO: Caching will save a lot of time...
	//TODO: check for errors
	clientAddr, _ := sdk.AccAddressFromBech32(clientEntry.Address)
	epochLast := k.epochStorageKeeper.GetEpochStart(ctx)
	clientProviderOverusedPercentMap := ClientProviderOverusedCUPercent{0.0, 0.0}

	// for every epoch in memory
	for epoch := k.epochStorageKeeper.GetEarliestEpochStart(ctx); epoch <= epochLast; epoch = k.epochStorageKeeper.GetNextEpoch(ctx, epoch) {
		// get epochPayments for this client
		clientStoragePaymentKeyEpoch := k.GetClientPaymentStorageKey(ctx, chainID, epoch, clientAddr)
		clientPaymentStorage, found := k.GetClientPaymentStorage(ctx, clientStoragePaymentKeyEpoch)
		if !found { // no payments this epoch, continue + advance epoch
			continue
		}
		//TODO: check for errors
		clientProvidersEpochUsedCUMap, _ := k.GetEpochClientProviderUsedCUMap(ctx, clientPaymentStorage)
		allowedCU, _ := k.GetAllowedCUClientEpoch(ctx, chainID, epoch, clientAddr)
		clientProvidersEpochOverusedCUMap, _ := k.GetOverusedFromUsedCU(ctx, clientProvidersEpochUsedCUMap, allowedCU, providerAddr)

		// Check negatives ?
		if clientProvidersEpochOverusedCUMap.TotalOverusedPercent > 0 {
			clientProviderOverusedPercentMap.TotalOverusedPercent += clientProvidersEpochOverusedCUMap.TotalOverusedPercent
		}
		if val, ok := clientProvidersEpochOverusedCUMap.Providers[providerAddr.String()]; ok {
			if val > 0 {
				clientProviderOverusedPercentMap.OverusedPercentProvider += val
			}
		}

		epoch = k.epochStorageKeeper.GetNextEpoch(ctx, epoch)
	}
	return clientProviderOverusedPercentMap, nil
}

func (k Keeper) LimitClientPairingsAndMarkForPenalty(ctx sdk.Context, chainID string, clientEntry *epochstoragetypes.StakeEntry, totalCUInEpochForUserProvider uint64, allowedCU uint64, allowedCUProvider uint64, providerAddr sdk.AccAddress) error {
	slashLimitPercent, unpayLimitPercent := getMaxCULimitsPercentage()
	clientOverusedCU, err := k.GetOverusedCUSumPercentage(ctx, chainID, clientEntry, providerAddr, totalCUInEpochForUserProvider)
	if err != nil {
		panic(fmt.Sprintf("user %s, could not calculate overusedCU from memory: %s", clientEntry, clientEntry.Stake.Amount))
	}
	overusedSumTotalPercent := clientOverusedCU.TotalOverusedPercent
	overusedSumProviderPercent := clientOverusedCU.OverusedPercentProvider
	providerCount := float64(k.ServicersToPairCount(ctx))
	if overusedSumTotalPercent > slashLimitPercent || overusedSumProviderPercent > slashLimitPercent/providerCount {
		k.SlashUser(ctx, clientEntry.Address)
		// TODO: what about slashing the provider ?
	}
	if overusedSumTotalPercent < unpayLimitPercent && overusedSumProviderPercent < unpayLimitPercent/providerCount {
		// overuse is under the limit - will allow provider to get payment
		// ? maybe needs to pay the allowedCU but not pay for overuse ?
		return nil
	}
	// overused is not over the slashLimit, but over the unpayLimit - not paying provider
	// ? maybe needs to pay the allowedCU but not pay for overuse ?
	return fmt.Errorf("user %s bypassed allowed CU %d by using: %d", clientEntry, allowedCU, totalCUInEpochForUserProvider)
}

func (k Keeper) SlashUser(ctx sdk.Context, clientAddr string) {
	//TODO: jail user, and count problems
}

func (k Keeper) ClientMaxCUProvider(ctx sdk.Context, clientEntry *epochstoragetypes.StakeEntry) uint64 {
	allowedCU, err := k.GetAllowedCU(ctx, clientEntry)
	if err != nil {
		panic(fmt.Sprintf("user %s, allowedCU was not found for stake of: %d", clientEntry, clientEntry.Stake.Amount.Int64()))
	}
	allowedCU = allowedCU / k.ServicersToPairCount(ctx)

	return allowedCU
}
