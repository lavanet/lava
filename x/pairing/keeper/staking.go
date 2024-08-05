package keeper

import (
	"fmt"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/v2/utils"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	"github.com/lavanet/lava/v2/x/pairing/types"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
)

const (
	MAX_CHANGE_RATE = 1
	CHANGE_WINDOW   = time.Hour * 24
)

func (k Keeper) StakeNewEntry(ctx sdk.Context, validator, creator, chainID string, amount sdk.Coin, endpoints []epochstoragetypes.Endpoint, geolocation int32, delegationLimit sdk.Coin, delegationCommission uint64, provider string, description stakingtypes.Description) error {
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
			utils.LogAttr("description", description.String()),
			utils.LogAttr("geolocation", geolocation),
			utils.LogAttr("max_endpoints_allowed", types.MAX_ENDPOINTS_AMOUNT_PER_GEO),
		)
	}

	// new staking takes effect from the next block
	stakeAppliedBlock := uint64(ctx.BlockHeight()) + 1

	existingEntry, entryExists := k.epochStorageKeeper.GetStakeEntryByAddressCurrent(ctx, chainID, creator)
	if entryExists {
		// modify the entry (check who's modifying - vault/provider)
		isProvider := false
		isVault := false
		if creator == existingEntry.Address {
			isProvider = true
		}
		if creator == existingEntry.Vault {
			isVault = true
		}

		if !isVault {
			// verify that the provider only tries to change non-stake related traits of the stake entry
			// a provider can be the same as the vault, so we verify it's not the case
			if isProvider {
				if delegationCommission != existingEntry.DelegateCommission ||
					!delegationLimit.Equal(existingEntry.DelegateLimit) ||
					!amount.Equal(existingEntry.Stake) {
					return utils.LavaFormatWarning("only vault address can change stake/delegation related properties of the stake entry", fmt.Errorf("invalid modification request for stake entry"),
						utils.LogAttr("creator", creator),
						utils.LogAttr("vault", existingEntry.Vault),
						utils.LogAttr("provider", provider),
						utils.LogAttr("description", description.String()),
						utils.LogAttr("current_delegation_limit", existingEntry.DelegateLimit),
						utils.LogAttr("req_delegation_limit", delegationLimit),
						utils.LogAttr("current_delegation_commission", existingEntry.DelegateCommission),
						utils.LogAttr("req_delegation_commission", delegationCommission),
						utils.LogAttr("current_stake", existingEntry.Stake.String()),
						utils.LogAttr("req_stake", amount.String()),
					)
				}
			} else {
				return utils.LavaFormatWarning("stake entry modification request was not created by provider/vault", fmt.Errorf("invalid creator address"),
					utils.LogAttr("spec", specChainID),
					utils.LogAttr("creator", creator),
					utils.LogAttr("provider", existingEntry.Address),
					utils.LogAttr("vault", existingEntry.Vault),
					utils.LogAttr("description", description.String()),
				)
			}
		}

		details := []utils.Attribute{
			{Key: "spec", Value: specChainID},
			{Key: "provider", Value: senderAddr.String()},
			{Key: "stakeAppliedBlock", Value: stakeAppliedBlock},
			{Key: "stake", Value: amount},
			utils.LogAttr("description", description.String()),
		}
		details = append(details, utils.Attribute{Key: "moniker", Value: description.Moniker})

		// if the provider has no delegations then we dont limit the changes
		if !existingEntry.DelegateTotal.IsZero() {
			// if there was a change in the last 24h than we dont allow changes
			if ctx.BlockTime().UTC().Unix()-int64(existingEntry.LastChange) < int64(CHANGE_WINDOW.Seconds()) {
				if delegationCommission != existingEntry.DelegateCommission || !existingEntry.DelegateLimit.IsEqual(delegationLimit) {
					return utils.LavaFormatWarning(fmt.Sprintf("stake entry commmision or delegate limit can only be changes once in %s", CHANGE_WINDOW), nil,
						utils.LogAttr("last_change_time", existingEntry.LastChange))
				}
			}

			// check that the change is not mode than MAX_CHANGE_RATE
			if int64(delegationCommission)-int64(existingEntry.DelegateCommission) > MAX_CHANGE_RATE {
				return utils.LavaFormatWarning("stake entry commission increase too high", fmt.Errorf("commission change cannot increase by more than %d at a time", MAX_CHANGE_RATE),
					utils.LogAttr("original_commission", existingEntry.DelegateCommission),
					utils.LogAttr("wanted_commission", delegationCommission),
				)
			}

			// check that the change in delegation limit is decreasing and that new_limit*100/old_limit < (100-MAX_CHANGE_RATE)
			if delegationLimit.IsLT(existingEntry.DelegateLimit) && delegationLimit.Amount.MulRaw(100).Quo(existingEntry.DelegateLimit.Amount).LT(sdk.NewInt(100-MAX_CHANGE_RATE)) {
				return utils.LavaFormatWarning("stake entry DelegateLimit decrease too high", fmt.Errorf("DelegateLimit change cannot decrease by more than %d at a time", MAX_CHANGE_RATE),
					utils.LogAttr("change_percentage", delegationLimit.Amount.MulRaw(100).Quo(existingEntry.DelegateLimit.Amount)),
					utils.LogAttr("original_limit", existingEntry.DelegateLimit),
					utils.LogAttr("wanted_limit", delegationLimit),
				)
			}
		}

		// we dont change stakeAppliedBlocks and chain once they are set, if they need to change, unstake first
		existingEntry.Geolocation = geolocation
		existingEntry.Endpoints = endpointsVerified
		existingEntry.Description = description
		existingEntry.DelegateCommission = delegationCommission
		existingEntry.DelegateLimit = delegationLimit
		existingEntry.LastChange = uint64(ctx.BlockTime().UTC().Unix())

		k.epochStorageKeeper.ModifyStakeEntryCurrent(ctx, chainID, existingEntry)

		if amount.Amount.GT(existingEntry.Stake.Amount) {
			// delegate the difference
			diffAmount := amount.Sub(existingEntry.Stake)
			err = k.dualstakingKeeper.DelegateFull(ctx, existingEntry.Vault, validator, existingEntry.Address, chainID, diffAmount)
			if err != nil {
				details = append(details, utils.Attribute{Key: "neededStake", Value: amount.Sub(existingEntry.Stake).String()})
				return utils.LavaFormatWarning("insufficient funds to pay for difference in stake", err,
					details...,
				)
			}
		} else if amount.Amount.LT(existingEntry.Stake.Amount) {
			// unbond the difference
			diffAmount := existingEntry.Stake.Sub(amount)
			err = k.dualstakingKeeper.UnbondFull(ctx, existingEntry.Vault, validator, existingEntry.Address, chainID, diffAmount, false)
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
		utils.LogLavaEvent(ctx, logger, types.ProviderStakeUpdateEventName, detailsMap, "Changing Stake Entry")
		return nil
	}

	// check that the configured provider is not used by another vault (when the provider and creator (vault) addresses are not equal)
	if provider != creator {
		providerStakeEntry, entryExists := k.epochStorageKeeper.GetStakeEntryByAddressCurrent(ctx, chainID, provider)
		if entryExists {
			return utils.LavaFormatWarning("configured provider exists", fmt.Errorf("new provider not staked"),
				utils.LogAttr("provider", provider),
				utils.LogAttr("chain_id", chainID),
				utils.LogAttr("existing_vault", providerStakeEntry.Vault),
			)
		}
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

	delegations, err := k.dualstakingKeeper.GetProviderDelegators(ctx, provider, nextEpoch)
	if err != nil {
		utils.LavaFormatWarning("cannot get provider's delegators", err,
			utils.LogAttr("provider", provider),
			utils.LogAttr("block", nextEpoch),
		)
	}

	for _, d := range delegations {
		if (d.Delegator == creator && d.Provider == provider) || d.ChainID != chainID {
			// ignore provider self delegation (delegator = vault, provider = provider) or delegations from other chains
			continue
		}
		delegateTotal = delegateTotal.Add(d.Amount.Amount)
	}

	stakeEntry := epochstoragetypes.StakeEntry{
		Stake:              sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), sdk.ZeroInt()), // we set this to 0 since the delegate will take care of this
		Address:            provider,
		StakeAppliedBlock:  stakeAppliedBlock,
		Endpoints:          endpointsVerified,
		Geolocation:        geolocation,
		Chain:              chainID,
		Description:        description,
		DelegateTotal:      sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), delegateTotal),
		DelegateLimit:      delegationLimit,
		DelegateCommission: delegationCommission,
		Vault:              creator, // the stake-provider TX creator is always regarded as the vault address
		LastChange:         uint64(ctx.BlockTime().UTC().Unix()),
	}

	k.epochStorageKeeper.AppendStakeEntryCurrent(ctx, chainID, stakeEntry)

	err = k.dualstakingKeeper.DelegateFull(ctx, stakeEntry.Vault, validator, stakeEntry.Address, chainID, amount)
	if err != nil {
		return utils.LavaFormatWarning("provider self delegation failed", err,
			details...,
		)
	}

	details = append(details, utils.Attribute{Key: "moniker", Value: description.Moniker})
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
	stakeEntry, found := k.epochStorageKeeper.GetStakeEntryByAddressCurrent(ctx, chainID, provider)
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
