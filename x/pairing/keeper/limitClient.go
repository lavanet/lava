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

func (k Keeper) GetAllowedCU(ctx sdk.Context, entry *epochstoragetypes.StakeEntry) (uint64, error) {
	var allowedCU uint64 = 0
	stakeToMaxCUMap := k.StakeToMaxCUList(ctx).List

	for _, stakeToCU := range stakeToMaxCUMap {
		if entry.Stake.IsGTE(stakeToCU.StakeThreshold) {
			allowedCU = stakeToCU.MaxComputeUnits
		} else {
			break
		}
	}
	return allowedCU, nil
}

func (k Keeper) EnforceClientCUsUsageInEpoch(ctx sdk.Context, relay *types.RelayRequest, clientEntry *epochstoragetypes.StakeEntry, clientAddr sdk.AccAddress, totalCUInEpochForUserProvider uint64, providerAddr sdk.AccAddress) (ammountToPay uint64, err error) {
	allowedCU, err := k.GetAllowedCU(ctx, clientEntry)
	if err != nil {
		panic(fmt.Sprintf("user %s, allowedCU was not found for stake of: %d", clientEntry, clientEntry.Stake.Amount.Int64()))
	}
	if allowedCU == 0 {
		return 0, fmt.Errorf("user %s, MaxCU was not found for stake of: %d", clientEntry, clientEntry.Stake.Amount.Int64())
		// panic(fmt.Sprintf("user %s, MaxCU was not found for stake of: %d", clientEntry, clientEntry.Stake.Amount.Int64()))
	}
	allowedCUProvider := allowedCU / k.ServicersToPairCount(ctx)
	if totalCUInEpochForUserProvider > allowedCUProvider {
		return k.LimitClientPairingsAndMarkForPenalty(ctx, relay, clientEntry, clientAddr, totalCUInEpochForUserProvider, allowedCU, allowedCUProvider, providerAddr)
	}

	return relay.CuSum, nil
}

func getMaxCULimitsPercentage() (float64, float64) {
	// TODO: Get from param
	slashLimitP, unpayLimitP := 0.2, 0.1 // 20% , 10%
	return slashLimitP, unpayLimitP
}

// TODO move to types - #O where would you like these to go ?

func (k Keeper) GetEpochClientProviderUsedCUMap(ctx sdk.Context, clientPaymentStorage types.ClientPaymentStorage) (clientOverusedCUMap types.ClientUsedCU, err error) {
	clientOverusedCUMap = types.ClientUsedCU{0, make(map[string]uint64)}
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
		return 0, stakeErr
	}
	// get allowed of client for this epoch
	allowedCU, allowedCUErr := k.GetAllowedCU(ctx, currentStakeEntry)
	if allowedCUErr != nil {
		return 0, allowedCUErr
	}
	return
}

func maxU(x uint64, y uint64) uint64 {
	if x > y {
		return x
	}
	return y
}
func maxF(x float64, y float64) float64 {
	if x > y {
		return x
	}
	return y
}
func (k Keeper) GetOverusedFromUsedCU(ctx sdk.Context, clientProvidersEpochUsedCUMap types.ClientUsedCU, allowedCU uint64, providerAddr sdk.AccAddress) (float64, float64) {
	if allowedCU <= 0 {
		panic(fmt.Sprintf("lava_GetOverusedFromUsedCU was called with %d allowedCU", allowedCU))
	}
	overusedProviderPercent := float64(0.0)
	totalOverusedPercent := float64(clientProvidersEpochUsedCUMap.TotalOverused / allowedCU)
	if usedCU, exist := clientProvidersEpochUsedCUMap.Providers[providerAddr.String()]; exist {
		// TODO: ServicersToPairCount needs epoch !
		allowedCUProvider := maxU(0, allowedCU/k.ServicersToPairCount(ctx))
		overusedCU := maxU(0, usedCU-allowedCUProvider)
		overusedProviderPercent = float64(overusedCU / allowedCUProvider)
	}
	return totalOverusedPercent, overusedProviderPercent
}
func (k Keeper) getOverusedCUPercentageAllEpochs(ctx sdk.Context, chainID string, clientAddr sdk.AccAddress, providerAddr sdk.AccAddress) (clientProviderOverusedCUPercent types.ClientProviderOverusedCUPercent, err error) {
	//TODO: Caching will save a lot of time...
	epochLast := k.epochStorageKeeper.GetEpochStart(ctx)
	clientProviderOverusedPercentMap := types.ClientProviderOverusedCUPercent{0.0, 0.0}

	// for every epoch in memory
	for epoch := k.epochStorageKeeper.GetEarliestEpochStart(ctx); epoch <= epochLast; epoch = k.epochStorageKeeper.GetNextEpoch(ctx, epoch) {
		// get epochPayments for this client
		clientStoragePaymentKeyEpoch := k.GetClientPaymentStorageKey(ctx, chainID, epoch, clientAddr)
		clientPaymentStorage, found := k.GetClientPaymentStorage(ctx, clientStoragePaymentKeyEpoch)
		if !found { // no payments this epoch, continue + advance epoch
			continue
		}
		clientProvidersEpochUsedCUMap, errPaymentStorage := k.GetEpochClientProviderUsedCUMap(ctx, clientPaymentStorage)
		if errPaymentStorage != nil || clientProvidersEpochUsedCUMap.TotalOverused == 0 {
			// no payments this epoch - continue
			continue
		}
		allowedCU, allowedCUErr := k.GetAllowedCUClientEpoch(ctx, chainID, epoch, clientAddr)
		if allowedCUErr != nil || allowedCU == 0 {
			// user has no stake this epoch - continue
			continue
		}
		totalOverusedPercent, providerOverusedPercent := k.GetOverusedFromUsedCU(ctx, clientProvidersEpochUsedCUMap, allowedCU, providerAddr)

		clientProviderOverusedPercentMap.TotalOverusedPercent += totalOverusedPercent
		clientProviderOverusedPercentMap.OverusedPercentProvider += providerOverusedPercent

		epoch = k.epochStorageKeeper.GetNextEpoch(ctx, epoch)
	}
	return clientProviderOverusedPercentMap, nil
}

func (k Keeper) LimitClientPairingsAndMarkForPenalty(ctx sdk.Context, relay *types.RelayRequest, clientEntry *epochstoragetypes.StakeEntry, clientAddr sdk.AccAddress, totalCUInEpochForUserProvider uint64, allowedCU uint64, allowedCUProvider uint64, providerAddr sdk.AccAddress) (amountToPay uint64, err error) {
	logger := k.Logger(ctx)
	chainID := relay.ChainID
	slashLimitPercent, unpayLimitPercent := getMaxCULimitsPercentage()
	// clientOverusedCU, err := k.getOverusedCUPercentageAllEpochs(ctx, chainID, sdk.AccAddress(clientEntry.Address), providerAddr)
	clientOverusedCU, err := k.getOverusedCUPercentageAllEpochs(ctx, chainID, clientAddr, providerAddr)
	if err != nil {
		utils.LavaError(ctx, logger, "lava_get_overused_cu", map[string]string{"block": strconv.FormatUint(k.epochStorageKeeper.GetEpochStart(ctx), 10),
			"relay.CuSum":  strconv.FormatUint(relay.CuSum, 10),
			"clientAddr":   clientAddr.String(),
			"providerAddr": providerAddr.String()},
			fmt.Sprintf("user %s, could not calculate overusedCU from memory: %s", clientEntry, clientEntry.Stake.Amount))
		return 0, fmt.Errorf("user %s, could not calculate overusedCU from memory: %s", clientEntry, clientEntry.Stake.Amount)
	}
	overusedSumTotalPercent := clientOverusedCU.TotalOverusedPercent
	overusedSumProviderPercent := clientOverusedCU.OverusedPercentProvider
	if overusedSumTotalPercent > slashLimitPercent || overusedSumProviderPercent > slashLimitPercent {
		k.SlashUser(ctx, clientEntry.Address)
		utils.LogLavaEvent(ctx, logger, "lava_slash_user", map[string]string{"block": strconv.FormatUint(k.epochStorageKeeper.GetEpochStart(ctx), 10),
			"relay.CuSum":                strconv.FormatUint(relay.CuSum, 10),
			"overusedSumTotalPercent":    strconv.FormatFloat(overusedSumTotalPercent, 'f', 6, 64),
			"overusedSumProviderPercent": strconv.FormatFloat(overusedSumProviderPercent, 'f', 6, 64),
			"slashLimitPercent":          strconv.FormatFloat(slashLimitPercent, 'f', 6, 64)},
			"overuse is above the slashLimit - slashing user - not paying provider")
		return uint64(0), nil
	}
	if overusedSumTotalPercent < unpayLimitPercent && overusedSumProviderPercent < unpayLimitPercent {
		// overuse is under the limit - will allow provider to get payment
		// ? maybe needs to pay the allowedCU but not pay for overuse ?
		utils.LogLavaEvent(ctx, logger, "lava_slash_user", map[string]string{"block": strconv.FormatUint(k.epochStorageKeeper.GetEpochStart(ctx), 10),
			"relay.CuSum":                strconv.FormatUint(relay.CuSum, 10),
			"overusedSumTotalPercent":    strconv.FormatFloat(overusedSumTotalPercent, 'f', 6, 64),
			"overusedSumProviderPercent": strconv.FormatFloat(overusedSumProviderPercent, 'f', 6, 64),
			"unpayLimitPercent":          strconv.FormatFloat(unpayLimitPercent, 'f', 6, 64)},
			"overuse is under the unpayLimit - paying provider")
		return relay.CuSum, nil
	}
	// overused over the unpayLimit (under slashLimit) - paying provider upto the unpayLimit
	utils.LogLavaEvent(ctx, logger, "lava_slash_user", map[string]string{"block": strconv.FormatUint(k.epochStorageKeeper.GetEpochStart(ctx), 10),
		"relay.CuSum":                strconv.FormatUint(relay.CuSum, 10),
		"overusedSumTotalPercent":    strconv.FormatFloat(overusedSumTotalPercent, 'f', 6, 64),
		"overusedSumProviderPercent": strconv.FormatFloat(overusedSumProviderPercent, 'f', 6, 64),
		"unpayLimitPercent":          strconv.FormatFloat(unpayLimitPercent, 'f', 6, 64)},
		"overuse is above the unpayLimit - paying provider only ")
	return uint64(math.Round(float64(relay.CuSum) * unpayLimitPercent)), nil
}

func (k Keeper) SlashUser(ctx sdk.Context, clientAddr string) {
	//TODO: jail user, and count problems
}

func (k Keeper) ClientMaxCUProvider(ctx sdk.Context, clientEntry *epochstoragetypes.StakeEntry) (uint64, error) {
	allowedCU, err := k.GetAllowedCU(ctx, clientEntry)
	if err != nil {
		return 0, fmt.Errorf("user %s, MaxCU was not found for stake of: %d", clientEntry, clientEntry.Stake.Amount.Int64())
	}
	allowedCU = allowedCU / k.ServicersToPairCount(ctx)

	return allowedCU, nil
}
