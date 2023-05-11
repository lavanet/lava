package keeper

import (
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
)

func (k Keeper) StakeNewEntry(ctx sdk.Context, provider bool, creator string, chainID string, amount sdk.Coin, endpoints []epochstoragetypes.Endpoint, geolocation uint64, vrfpk string, moniker string) error {
	logger := k.Logger(ctx)
	var stake_type string
	if provider {
		stake_type = epochstoragetypes.ProviderKey
	} else {
		stake_type = epochstoragetypes.ClientKey
	}
	// TODO: basic validation for chain ID
	specChainID := chainID

	spec, found := k.specKeeper.GetSpec(ctx, specChainID)
	if !found || !spec.Enabled {
		return utils.LavaFormatWarning("spec not found or not active", fmt.Errorf("invalid spec ID"),
			utils.Attribute{Key: "spec", Value: specChainID},
		)
	}
	var minStake sdk.Coin
	if provider {
		minStake = spec.MinStakeProvider
	} else {
		minStake = spec.MinStakeClient
	}
	// if we get here, the spec is active and supported

	if amount.IsLT(minStake) { // we count on this to also check the denom
		return utils.LavaFormatWarning("insufficient "+stake_type+" stake amount", fmt.Errorf("stake amount smaller than minStake"),
			utils.Attribute{Key: "spec", Value: specChainID},
			utils.Attribute{Key: stake_type, Value: creator},
			utils.Attribute{Key: "stake", Value: amount},
			utils.Attribute{Key: "minStake", Value: minStake.String()},
		)
	}
	senderAddr, err := sdk.AccAddressFromBech32(creator)
	if err != nil {
		return utils.LavaFormatWarning("invalid "+stake_type+" address", err,
			utils.Attribute{Key: stake_type, Value: creator},
		)
	}
	// define the function here for later use
	verifySufficientAmountAndSendToModule := func(ctx sdk.Context, k Keeper, addr sdk.AccAddress, neededAmount sdk.Coin) error {
		if k.bankKeeper.GetBalance(ctx, addr, epochstoragetypes.TokenDenom).IsLT(neededAmount) {
			return utils.LavaFormatWarning("insufficient balance for staking", fmt.Errorf("insufficient funds"),
				utils.Attribute{Key: "current balance", Value: k.bankKeeper.GetBalance(ctx, addr, epochstoragetypes.TokenDenom)},
				utils.Attribute{Key: "stake amount", Value: neededAmount},
			)
		}
		if neededAmount.Amount == sdk.ZeroInt() {
			return nil
		}
		err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, addr, types.ModuleName, []sdk.Coin{neededAmount})
		if err != nil {
			return utils.LavaFormatError("invalid transfer coins to module", err)
		}
		return nil
	}
	geolocations := k.specKeeper.GeolocationCount(ctx)
	if geolocation == 0 || geolocation > (1<<geolocations) {
		return utils.LavaFormatWarning("can't register for no geolocation or geolocation outside zones", fmt.Errorf("invalid geolocation"),
			utils.Attribute{Key: "geolocation", Value: geolocation},
		)
	}
	if provider {
		err := k.validateGeoLocationAndApiInterfaces(ctx, endpoints, geolocation, specChainID)
		if err != nil {
			return utils.LavaFormatWarning("invalid "+stake_type+" endpoints implementation for the given spec", err,
				utils.Attribute{Key: stake_type, Value: creator},
				utils.Attribute{Key: "endpoints", Value: endpoints},
				utils.Attribute{Key: "Chain", Value: chainID},
				utils.Attribute{Key: "geolocation", Value: geolocation},
			)
		}
	} else {
		// clients need to provide their VRF PK before running to limit brute forcing the random functions
		err := utils.VerifyVRF(vrfpk)
		if err != nil {
			return utils.LavaFormatWarning("invalid "+stake_type+" stake: invalid vrf pk, must provide a valid verification key", err,
				utils.Attribute{Key: stake_type, Value: creator},
			)
		}
	}

	// new staking takes effect from the next block
	stakeAppliedBlock := uint64(ctx.BlockHeight()) + 1

	if len(moniker) > 50 {
		moniker = moniker[:50]
	}

	existingEntry, entryExists, indexInStakeStorage := k.epochStorageKeeper.GetStakeEntryByAddressCurrent(ctx, stake_type, chainID, senderAddr)
	if entryExists {
		// modify the entry
		if existingEntry.Address != creator {
			return utils.LavaFormatWarning("returned stake entry by address doesn't match sender address", fmt.Errorf("sender and stake entry address mismatch"),
				utils.Attribute{Key: "spec", Value: specChainID},
				utils.Attribute{Key: stake_type, Value: senderAddr.String()},
			)
		}
		details := []utils.Attribute{
			{Key: "spec", Value: specChainID},
			{Key: stake_type, Value: senderAddr.String()},
			{Key: "stakeAppliedBlock", Value: stakeAppliedBlock},
			{Key: "stake", Value: amount},
		}
		details = append(details, utils.Attribute{Key: "moniker", Value: moniker})
		if amount.IsGTE(existingEntry.Stake) {
			// support modifying with the same stake or greater only
			if !amount.Equal(existingEntry.Stake) {
				// needs to charge additional tokens
				err := verifySufficientAmountAndSendToModule(ctx, k, senderAddr, amount.Sub(existingEntry.Stake))
				if err != nil {
					details = append(details, utils.Attribute{Key: "neededStake", Value: amount.Sub(existingEntry.Stake).String()})
					return utils.LavaFormatWarning("insufficient funds to pay for difference in stake", err,
						details...,
					)
				}
			}
			// TODO: create a new entry entirely because then we can keep the copies of this list as pointers only
			// then we need to change the Copy of StoreCurrentEpochStakeStorage to copy of the pointers only
			// must also change the unstaking to create a new entry entirely

			// paid the difference to module
			existingEntry.Stake = amount
			// we dont change vrfpk, stakeAppliedBlocks and chain once they are set, if they need to change, unstake first
			existingEntry.Geolocation = geolocation
			existingEntry.Endpoints = endpoints
			existingEntry.Moniker = moniker
			k.epochStorageKeeper.ModifyStakeEntryCurrent(ctx, stake_type, chainID, existingEntry, indexInStakeStorage)
			detailsMap := map[string]string{}
			for _, val := range details {
				detailsMap[val.Key] = fmt.Sprint(val.Value)
			}
			utils.LogLavaEvent(ctx, logger, types.StakeUpdateEventName(provider), detailsMap, "Changing Staked "+stake_type)
			return nil
		}
		details = append(details, utils.Attribute{Key: "existingStake", Value: existingEntry.Stake.String()})
		return utils.LavaFormatWarning("can't decrease stake for existing "+stake_type, fmt.Errorf("cant decrease stake"),
			details...,
		)
	}

	// entry isn't staked so add him
	details := []utils.Attribute{
		{Key: "spec", Value: specChainID},
		{Key: stake_type, Value: senderAddr.String()},
		{Key: "stakeAppliedBlock", Value: stakeAppliedBlock},
		{Key: "stake", Value: amount.String()},
		{Key: "geolocation", Value: geolocation},
	}
	err = verifySufficientAmountAndSendToModule(ctx, k, senderAddr, amount)
	if err != nil {
		utils.LavaFormatWarning("insufficient amount to pay for stake", err,
			details...,
		)
	}

	stakeEntry := epochstoragetypes.StakeEntry{Stake: amount, Address: creator, StakeAppliedBlock: stakeAppliedBlock, Endpoints: endpoints, Geolocation: geolocation, Chain: chainID, Vrfpk: vrfpk, Moniker: moniker}
	k.epochStorageKeeper.AppendStakeEntryCurrent(ctx, stake_type, chainID, stakeEntry)
	appended := false
	if !provider {
		// this is done so consumers can use services upon staking for the first time and dont have to wait for the next epoch
		appended, err = k.epochStorageKeeper.BypassCurrentAndAppendNewEpochStakeEntry(ctx, stake_type, chainID, stakeEntry)

		if err != nil {
			return utils.LavaFormatError("could not append epoch stake entries", err,
				details...,
			)
		}
	}
	details = append(details, utils.Attribute{Key: "effectiveImmediately", Value: appended})
	details = append(details, utils.Attribute{Key: "moniker", Value: moniker})

	detailsMap := map[string]string{}
	for _, atr := range details {
		detailsMap[atr.Key] = fmt.Sprint(atr.Value)
	}
	utils.LogLavaEvent(ctx, logger, types.StakeNewEventName(provider), detailsMap, "Adding Staked "+stake_type)
	return err
}

func (k Keeper) validateGeoLocationAndApiInterfaces(ctx sdk.Context, endpoints []epochstoragetypes.Endpoint, geolocation uint64, chainID string) (err error) {
	expectedInterfaces := k.specKeeper.GetExpectedInterfacesForSpec(ctx, chainID)
	geolocMap := map[string]bool{} // TODO: turn this into spectypes.ApiInterface
	geolocations := k.specKeeper.GeolocationCount(ctx)
	geolocKey := func(intefaceName string, geolocation uint64) string {
		return intefaceName + "_" + strconv.FormatUint(geolocation, 10)
	}
	for idx := uint64(0); idx < geolocations; idx++ {
		// geolocation is a bit mask for areas, each bit turns support for an area
		geolocZone := geolocation & (1 << idx)
		if geolocZone != 0 {
			for expectedApiInterface := range expectedInterfaces {
				geolocMap[geolocKey(expectedApiInterface, geolocZone)] = true
			}
		}
	}
	// check all endpoints only implement expected interfaces
	for _, endpoint := range endpoints {
		key := geolocKey(endpoint.UseType, endpoint.Geolocation)
		if geolocMap[key] {
			continue
		} else {
			return fmt.Errorf("servicer implemented api interfaces that are not in the spec: %s, current expected: %+v", key, geolocMap)
		}
	}
	// check all expected api interfaces are implemented
	for _, endpoint := range endpoints {
		key := geolocKey(endpoint.UseType, endpoint.Geolocation)
		if geolocMap[key] {
			// interface is implemented and expected
			delete(geolocMap, key) // remove this from expected implementations
		}
	}
	if len(geolocMap) == 0 {
		// all interfaces and geolocations were implemented
		return nil
	}
	return fmt.Errorf("not all expected interfaces are implemented for all geolocations: %+v, missing implementation count: %d", geolocMap, len(geolocMap))
}
