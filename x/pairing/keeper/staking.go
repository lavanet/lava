package keeper

import (
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

func (k Keeper) StakeNewEntry(ctx sdk.Context, creator, chainID string, amount sdk.Coin, endpoints []epochstoragetypes.Endpoint, geolocation int32, moniker string, delegationLimit sdk.Coin, delegationCommission uint64) error {
	logger := k.Logger(ctx)
	// TODO: basic validation for chain ID
	specChainID := chainID

	spec, err := k.specKeeper.GetExpandedSpec(ctx, specChainID)
	if err != nil || !spec.Enabled {
		return utils.LavaFormatWarning("spec not found or not active", err,
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

	if !planstypes.IsValidGeoEnum(geolocation) {
		return utils.LavaFormatWarning(`geolocations are treated as a bitmap. To configure multiple geolocations, 
		use the uint representation of the valid geolocations`, fmt.Errorf("missing or invalid geolocation"),
			utils.Attribute{Key: "geolocation", Value: geolocation},
			utils.Attribute{Key: "valid_geolocations", Value: planstypes.PrintGeolocations()},
		)
	}

	endpointsVerified, err := k.validateGeoLocationAndApiInterfaces(ctx, endpoints, geolocation, spec)
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
		// TODO handle this
		if amount.IsGTE(existingEntry.Stake) {
			// support modifying with the same stake or greater only
			if !amount.Equal(existingEntry.Stake) {
				// delegate the difference
				amount = amount.Sub(existingEntry.Stake)
				err = k.dualStakingKeeper.Delegate(ctx, senderAddr.String(), senderAddr.String(), chainID, amount)
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

			// we dont change stakeAppliedBlocks and chain once they are set, if they need to change, unstake first
			existingEntry.Geolocation = geolocation
			existingEntry.Endpoints = endpointsVerified
			existingEntry.Moniker = moniker
			existingEntry.DelegateCommission = delegationCommission
			existingEntry.DelegateLimit = delegationLimit

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

	stakeEntry := epochstoragetypes.StakeEntry{
		Stake:              sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.ZeroInt()),
		Address:            creator,
		StakeAppliedBlock:  stakeAppliedBlock,
		Endpoints:          endpointsVerified,
		Geolocation:        geolocation,
		Chain:              chainID,
		Moniker:            moniker,
		DelegateTotal:      sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.ZeroInt()),
		DelegateLimit:      delegationLimit,
		DelegateCommission: delegationCommission,
	}

	k.epochStorageKeeper.AppendStakeEntryCurrent(ctx, chainID, stakeEntry)

	err = k.dualStakingKeeper.Delegate(ctx, senderAddr.String(), senderAddr.String(), chainID, amount)
	if err != nil {
		utils.LavaFormatWarning("provider self delegation failed", err,
			details...,
		)
	}

	details = append(details, utils.Attribute{Key: "moniker", Value: moniker})

	detailsMap := map[string]string{}
	for _, atr := range details {
		detailsMap[atr.Key] = fmt.Sprint(atr.Value)
	}
	utils.LogLavaEvent(ctx, logger, types.ProviderStakeEventName, detailsMap, "Adding Staked provider")
	return err
}

func (k Keeper) validateGeoLocationAndApiInterfaces(ctx sdk.Context, endpoints []epochstoragetypes.Endpoint, geolocation int32, spec spectypes.Spec) (endpointsFormatted []epochstoragetypes.Endpoint, err error) {
	expectedInterfaces := k.specKeeper.GetExpectedServicesForExpandedSpec(spec, true)
	allowedInterfaces := k.specKeeper.GetExpectedServicesForExpandedSpec(spec, false)

	geolocMapRequired := map[epochstoragetypes.EndpointService]struct{}{}
	geolocMapAllowed := map[epochstoragetypes.EndpointService]struct{}{}
	geolocations := len(planstypes.GetAllGeolocations())

	geolocKey := func(intefaceName string, geolocation int32, addon, extension string) epochstoragetypes.EndpointService {
		return epochstoragetypes.EndpointService{
			ApiInterface: intefaceName + "_" + strconv.FormatInt(int64(geolocation), 10),
			Addon:        addon,
			Extension:    extension,
		}
	}

	for idx := uint64(0); idx < uint64(geolocations); idx++ {
		// geolocation is a bit mask for areas, each bit turns support for an area
		geolocZone := geolocation & (1 << idx)
		if geolocZone != 0 {
			for expectedEndpointService := range expectedInterfaces {
				key := geolocKey(expectedEndpointService.ApiInterface, geolocZone, expectedEndpointService.Addon, "")
				geolocMapRequired[key] = struct{}{}
			}
			for expectedEndpointService := range allowedInterfaces {
				key := geolocKey(expectedEndpointService.ApiInterface, geolocZone, expectedEndpointService.Addon, expectedEndpointService.Extension)
				geolocMapAllowed[key] = struct{}{}
			}
		}
	}
	addonsMap, extensionsMap := spec.ServicesMap()
	// check all endpoints only implement allowed interfaces
	for idx, endpoint := range endpoints {
		err = endpoint.SetServicesFromAddons(allowedInterfaces, addonsMap, extensionsMap) // support apiInterfaces/extensions inside addons list
		if err != nil {
			return nil, fmt.Errorf("provider implements addons not allowed in the spec: %s, endpoint: %+v, allowed interfaces: %+v", spec.Index, endpoint, allowedInterfaces)
		}
		endpoint.SetDefaultApiInterfaces(expectedInterfaces) // support empty apiInterfaces list
		endpoints[idx] = endpoint
		for _, endpointService := range endpoint.GetSupportedServices() {
			key := geolocKey(endpointService.ApiInterface, endpoint.Geolocation, endpointService.Addon, endpointService.Extension)
			if _, ok := geolocMapAllowed[key]; !ok {
				return nil, fmt.Errorf("provider implements api interfaces not allowed in the spec: %s, current allowed: %+v", key, geolocMapAllowed)
			}
		}
	}

	// check all expected api interfaces are implemented
	for _, endpoint := range endpoints {
		for _, endpointService := range endpoint.GetSupportedServices() {
			key := geolocKey(endpointService.ApiInterface, endpoint.Geolocation, endpointService.Addon, "")
			delete(geolocMapRequired, key) // remove this from expected implementations
		}
	}

	if len(geolocMapRequired) != 0 {
		return nil, fmt.Errorf("servicer does not implement all expected interfaces for all geolocations: %+v, missing implementation count: %d", geolocMapRequired, len(geolocMapRequired))
	}

	// all interfaces and geolocations were implemented
	return endpoints, nil
}

func (k Keeper) GetStakeEntry(ctx sdk.Context, chainID string, provider string) (epochstoragetypes.StakeEntry, error) {
	providerAcc, err := sdk.AccAddressFromBech32(provider)
	if err != nil {
		return epochstoragetypes.StakeEntry{}, utils.LavaFormatWarning("invalid provider address", fmt.Errorf("cannot get stake entry"),
			utils.Attribute{Key: "provider", Value: provider},
		)
	}

	stakeEntry, found, _ := k.epochStorageKeeper.GetStakeEntryByAddressCurrent(ctx, chainID, providerAcc)
	if !found {
		return epochstoragetypes.StakeEntry{}, utils.LavaFormatWarning("provider not staked on chain", fmt.Errorf("cannot get stake entry"),
			utils.Attribute{Key: "chainID", Value: chainID},
			utils.Attribute{Key: "provider", Value: provider},
		)
	}

	return stakeEntry, nil
}

func (k Keeper) GetAllChainIDs(ctx sdk.Context) []string {
	return k.specKeeper.GetAllChainIDs(ctx)
}
