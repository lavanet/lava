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

func (k Keeper) StakeNewEntry(ctx sdk.Context, validator, creator, chainID string, amount sdk.Coin, endpoints []epochstoragetypes.Endpoint, geolocation int32, moniker string, delegationLimit sdk.Coin, delegationCommission uint64) error {
	logger := k.Logger(ctx)
	specChainID := chainID

	spec, err := k.specKeeper.GetExpandedSpec(ctx, specChainID)
	if err != nil || !spec.Enabled {
		return utils.LavaFormatWarning("spec not found or not active", err,
			utils.Attribute{Key: "spec", Value: specChainID},
		)
	}

	// if we get here, the spec is active and supported
	if amount.IsLT(k.dualstakingKeeper.MinSelfDelegation(ctx)) { // we count on this to also check the denom
		return utils.LavaFormatWarning("insufficient stake amount", fmt.Errorf("stake amount smaller than MinSelfDelegation"),
			utils.Attribute{Key: "spec", Value: specChainID},
			utils.Attribute{Key: "provider", Value: creator},
			utils.Attribute{Key: "stake", Value: amount},
			utils.Attribute{Key: "minSelfDelegation", Value: k.dualstakingKeeper.MinSelfDelegation(ctx).String()},
		)
	}
	senderAddr, err := sdk.AccAddressFromBech32(creator)
	if err != nil {
		return utils.LavaFormatWarning("invalid address", err,
			utils.Attribute{Key: "provider", Value: creator},
		)
	}

	if !planstypes.IsValidProviderGeoEnum(geolocation) {
		return utils.LavaFormatWarning(`geolocations are treated as a bitmap. To configure multiple geolocations, 
		use the uint representation of the valid geolocations`, fmt.Errorf("missing or invalid geolocation"),
			utils.Attribute{Key: "geolocation", Value: geolocation},
			utils.Attribute{Key: "valid_geolocations", Value: planstypes.PrintGeolocations()},
		)
	}

	endpointsVerified, err := k.validateGeoLocationAndApiInterfaces(endpoints, geolocation, spec)
	if err != nil {
		return utils.LavaFormatWarning("invalid endpoints implementation for the given spec", err,
			utils.Attribute{Key: "provider", Value: creator},
			utils.Attribute{Key: "endpoints", Value: endpoints},
			utils.Attribute{Key: "chain", Value: chainID},
			utils.Attribute{Key: "geolocation", Value: geolocation},
		)
	}

	// validate there are no more than 5 endpoints per geolocation
	if len(endpoints) > len(planstypes.GetGeolocationsFromUint(geolocation))*types.MAX_ENDPOINTS_AMOUNT_PER_GEO {
		return utils.LavaFormatWarning("stake provider failed", fmt.Errorf("number of endpoint for geolocation exceeded limit"),
			utils.LogAttr("creator", creator),
			utils.LogAttr("chain_id", chainID),
			utils.LogAttr("moniker", moniker),
			utils.LogAttr("geolocation", geolocation),
			utils.LogAttr("max_endpoints_allowed", types.MAX_ENDPOINTS_AMOUNT_PER_GEO),
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

		// we dont change stakeAppliedBlocks and chain once they are set, if they need to change, unstake first
		existingEntry.Geolocation = geolocation
		existingEntry.Endpoints = endpointsVerified
		existingEntry.Moniker = moniker
		existingEntry.DelegateCommission = delegationCommission
		existingEntry.DelegateLimit = delegationLimit

		k.epochStorageKeeper.ModifyStakeEntryCurrent(ctx, chainID, existingEntry, indexInStakeStorage)

		if amount.Amount.GT(existingEntry.Stake.Amount) {
			// delegate the difference
			diffAmount := amount.Sub(existingEntry.Stake)
			err = k.dualstakingKeeper.DelegateFull(ctx, senderAddr.String(), validator, senderAddr.String(), chainID, diffAmount)
			if err != nil {
				details = append(details, utils.Attribute{Key: "neededStake", Value: amount.Sub(existingEntry.Stake).String()})
				return utils.LavaFormatWarning("insufficient funds to pay for difference in stake", err,
					details...,
				)
			}
		} else if amount.Amount.LT(existingEntry.Stake.Amount) {
			// unbond the difference
			diffAmount := existingEntry.Stake.Sub(amount)
			err = k.dualstakingKeeper.UnbondFull(ctx, senderAddr.String(), validator, senderAddr.String(), chainID, diffAmount, false)
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

		detailsMap := map[string]string{}
		for _, val := range details {
			detailsMap[val.Key] = fmt.Sprint(val.Value)
		}
		utils.LogLavaEvent(ctx, logger, types.ProviderStakeUpdateEventName, detailsMap, "Changing Stake")
		return nil
	}

	// entry isn't staked so add him
	details := []utils.Attribute{
		{Key: "spec", Value: specChainID},
		{Key: "provider", Value: senderAddr.String()},
		{Key: "stakeAppliedBlock", Value: stakeAppliedBlock},
		{Key: "stake", Value: amount.String()},
		{Key: "geolocation", Value: geolocation},
	}

	// if there are registered delegations to the provider, count them in the delegateTotal
	delegateTotal := sdk.ZeroInt()
	nextEpoch, err := k.epochStorageKeeper.GetNextEpoch(ctx, uint64(ctx.BlockHeight()))
	if err != nil {
		return utils.LavaFormatWarning("cannot get next epoch to count past delegations", err,
			utils.LogAttr("provider", senderAddr.String()),
			utils.LogAttr("block", nextEpoch),
		)
	}

	delegations, err := k.dualstakingKeeper.GetProviderDelegators(ctx, senderAddr.String(), nextEpoch)
	if err != nil {
		utils.LavaFormatWarning("cannot get provider's delegators", err,
			utils.LogAttr("provider", senderAddr.String()),
			utils.LogAttr("block", nextEpoch),
		)
	}

	for _, d := range delegations {
		if d.Delegator == senderAddr.String() {
			// ignore provider self delegation
			continue
		}
		delegateTotal = delegateTotal.Add(d.Amount.Amount)
	}

	stakeEntry := epochstoragetypes.StakeEntry{
		Stake:              sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), sdk.ZeroInt()), // we set this to 0 since the delegate will take care of this
		Address:            creator,
		StakeAppliedBlock:  stakeAppliedBlock,
		Endpoints:          endpointsVerified,
		Geolocation:        geolocation,
		Chain:              chainID,
		Moniker:            moniker,
		DelegateTotal:      sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), delegateTotal),
		DelegateLimit:      delegationLimit,
		DelegateCommission: delegationCommission,
	}

	k.epochStorageKeeper.AppendStakeEntryCurrent(ctx, chainID, stakeEntry)

	err = k.dualstakingKeeper.DelegateFull(ctx, senderAddr.String(), validator, senderAddr.String(), chainID, amount)
	if err != nil {
		return utils.LavaFormatWarning("provider self delegation failed", err,
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

func (k Keeper) validateGeoLocationAndApiInterfaces(endpoints []epochstoragetypes.Endpoint, geolocation int32, spec spectypes.Spec) (endpointsFormatted []epochstoragetypes.Endpoint, err error) {
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
