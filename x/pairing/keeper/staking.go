package keeper

import (
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
)

func (k Keeper) StakeNewEntry(ctx sdk.Context, creator string, chainID string, amount sdk.Coin, endpoints []epochstoragetypes.Endpoint, geolocation uint64, moniker string) error {
	logger := k.Logger(ctx)

	// TODO: basic validation for chain ID
	specChainID := chainID

	spec, found := k.specKeeper.GetSpec(ctx, specChainID)
	if !found || !spec.Enabled {
		return utils.LavaFormatWarning("spec not found or not active", fmt.Errorf("invalid spec ID"),
			utils.Attribute{Key: "spec", Value: specChainID},
		)
	}

	// if we get here, the spec is active and supported
	if amount.IsLT(spec.MinStakeProvider) { // we count on this to also check the denom
		return utils.LavaFormatWarning("insufficient stake amount", fmt.Errorf("stake amount smaller than minStake"),
			utils.Attribute{Key: "spec", Value: specChainID},
			utils.Attribute{Key: "provider", Value: creator},
			utils.Attribute{Key: "stake", Value: amount},
			utils.Attribute{Key: "minStake", Value: spec.MinStakeProvider.String()},
		)
	}
	senderAddr, err := sdk.AccAddressFromBech32(creator)
	if err != nil {
		return utils.LavaFormatWarning("invalid address", err,
			utils.Attribute{Key: "provider", Value: creator},
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

	if !planstypes.IsValidGeoEnum(int32(geolocation)) {
		return utils.LavaFormatWarning("can't register for no geolocation or geolocation outside zones", fmt.Errorf("invalid geolocation"),
			utils.Attribute{Key: "geolocation", Value: geolocation},
		)
	}

	endpointsVerified, err := k.validateGeoLocationAndApiInterfaces(ctx, endpoints, geolocation, specChainID)
	if err != nil {
		return utils.LavaFormatWarning("invalid endpoints implementation for the given spec", err,
			utils.Attribute{Key: "provider", Value: creator},
			utils.Attribute{Key: "endpoints", Value: endpoints},
			utils.Attribute{Key: "chain", Value: chainID},
			utils.Attribute{Key: "geolocation", Value: geolocation},
		)
	}
	// new staking takes effect from the next block
	stakeAppliedBlock := uint64(ctx.BlockHeight()) + 1

	if len(moniker) > 50 {
		moniker = moniker[:50]
	}

	existingEntry, entryExists, indexInStakeStorage := k.epochStorageKeeper.GetStakeEntryByAddressCurrent(ctx, chainID, senderAddr)
	if entryExists {
		// modify the entry
		if existingEntry.Address != creator {
			return utils.LavaFormatWarning("returned stake entry by address doesn't match sender address", fmt.Errorf("sender and stake entry address mismatch"),
				utils.Attribute{Key: "spec", Value: specChainID},
				utils.Attribute{Key: "provider", Value: senderAddr.String()},
			)
		}
		details := []utils.Attribute{
			{Key: "spec", Value: specChainID},
			{Key: "provider", Value: senderAddr.String()},
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
			// we dont change stakeAppliedBlocks and chain once they are set, if they need to change, unstake first
			existingEntry.Geolocation = geolocation
			existingEntry.Endpoints = endpointsVerified
			existingEntry.Moniker = moniker
			k.epochStorageKeeper.ModifyStakeEntryCurrent(ctx, chainID, existingEntry, indexInStakeStorage)
			detailsMap := map[string]string{}
			for _, val := range details {
				detailsMap[val.Key] = fmt.Sprint(val.Value)
			}
			utils.LogLavaEvent(ctx, logger, types.ProviderStakeUpdateEventName, detailsMap, "Changing Stake")
			return nil
		}
		details = append(details, utils.Attribute{Key: "existingStake", Value: existingEntry.Stake.String()})
		return utils.LavaFormatWarning("can't decrease stake for existing provider", fmt.Errorf("cant decrease stake"),
			details...,
		)
	}

	// entry isn't staked so add him
	details := []utils.Attribute{
		{Key: "spec", Value: specChainID},
		{Key: "provider", Value: senderAddr.String()},
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

	stakeEntry := epochstoragetypes.StakeEntry{Stake: amount, Address: creator, StakeAppliedBlock: stakeAppliedBlock, Endpoints: endpointsVerified, Geolocation: geolocation, Chain: chainID, Moniker: moniker}
	k.epochStorageKeeper.AppendStakeEntryCurrent(ctx, chainID, stakeEntry)
	appended := false

	details = append(details, utils.Attribute{Key: "effectiveImmediately", Value: appended})
	details = append(details, utils.Attribute{Key: "moniker", Value: moniker})

	detailsMap := map[string]string{}
	for _, atr := range details {
		detailsMap[atr.Key] = fmt.Sprint(atr.Value)
	}
	utils.LogLavaEvent(ctx, logger, types.ProviderStakeEventName, detailsMap, "Adding Staked provider")
	return err
}

func (k Keeper) validateGeoLocationAndApiInterfaces(ctx sdk.Context, endpoints []epochstoragetypes.Endpoint, geolocation uint64, chainID string) (endpointsFormatted []epochstoragetypes.Endpoint, err error) {
	expectedInterfaces, err := k.specKeeper.GetExpectedInterfacesForSpec(ctx, chainID, true)
	if err != nil {
		return nil, fmt.Errorf("expected interfaces: %w", err)
	}
	allowedInterfaces, err := k.specKeeper.GetExpectedInterfacesForSpec(ctx, chainID, false)
	if err != nil {
		return nil, fmt.Errorf("allowed interfaces: %w", err)
	}

	geolocMapRequired := map[epochstoragetypes.EndpointService]struct{}{}
	geolocMapAllowed := map[epochstoragetypes.EndpointService]struct{}{}
	geolocations := k.specKeeper.GeolocationCount(ctx)

	geolocKey := func(intefaceName string, geolocation uint64, addon string) epochstoragetypes.EndpointService {
		return epochstoragetypes.EndpointService{
			ApiInterface: intefaceName + "_" + strconv.FormatUint(geolocation, 10),
			Addon:        addon,
		}
	}

	for idx := uint64(0); idx < geolocations; idx++ {
		// geolocation is a bit mask for areas, each bit turns support for an area
		geolocZone := geolocation & (1 << idx)
		if geolocZone != 0 {
			for expectedEndpointService := range expectedInterfaces {
				key := geolocKey(expectedEndpointService.ApiInterface, geolocZone, expectedEndpointService.Addon)
				geolocMapRequired[key] = struct{}{}
			}
			for expectedEndpointService := range allowedInterfaces {
				key := geolocKey(expectedEndpointService.ApiInterface, geolocZone, expectedEndpointService.Addon)
				geolocMapAllowed[key] = struct{}{}
			}
		}
	}

	// check all endpoints only implement expected interfaces
	for idx, endpoint := range endpoints {
		endpoint.SetApiInterfacesFromAddons(allowedInterfaces) // support apiInterfaces inside addons list
		endpoint.SetDefaultApiInterfaces(expectedInterfaces)   // support empty apiInterfaces list
		endpoints[idx] = endpoint
		for _, endpointService := range endpoint.GetSupportedServices() {
			key := geolocKey(endpointService.ApiInterface, endpoint.Geolocation, endpointService.Addon)
			if _, ok := geolocMapAllowed[key]; !ok {
				return nil, fmt.Errorf("servicer implements api interfaces not allowed in the spec: %s, current allowed: %+v", key, geolocMapAllowed)
			}
		}
	}

	// check all expected api interfaces are implemented
	for _, endpoint := range endpoints {
		for _, endpointService := range endpoint.GetSupportedServices() {
			key := geolocKey(endpointService.ApiInterface, endpoint.Geolocation, endpointService.Addon)
			delete(geolocMapRequired, key) // remove this from expected implementations
		}
	}

	if len(geolocMapRequired) != 0 {
		return nil, fmt.Errorf("servicer does not implement all expected interfaces for all geolocations: %+v, missing implementation count: %d", geolocMapRequired, len(geolocMapRequired))
	}

	// all interfaces and geolocations were implemented
	return endpoints, nil
}
