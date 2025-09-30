package keeper

import (
	"fmt"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/lavaslices"
	epochstoragetypes "github.com/lavanet/lava/v5/x/epochstorage/types"
	"github.com/lavanet/lava/v5/x/pairing/types"
	planstypes "github.com/lavanet/lava/v5/x/plans/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
)

const (
	MAX_CHANGE_RATE   = 1
	CHANGE_WINDOW     = time.Hour * 24
	STAKE_MOVE_WINDOW = time.Hour * 24 // 24-hour cooldown for stake moves
)

// checkStakeMoveRateLimit checks rate limiting for stake moves
// This method handles both individual and bulk operations (multiple stake moves in the same transaction)
func (k Keeper) checkStakeMoveRateLimit(ctx sdk.Context, provider string) error {
	// Check if this provider has already been rate-limited in this transaction
	// by checking if we already have a pending timestamp update for this block
	metadata, err := k.epochStorageKeeper.GetMetadata(ctx, provider)
	if err != nil {
		// If metadata doesn't exist, this is a fresh provider - allow the move
		return nil
	}

	currentBlockTime := uint64(ctx.BlockTime().UTC().Unix())

	// If LastStakeMove is already set to current block time, this means
	// we've already processed a stake move for this provider in this transaction
	if metadata.LastStakeMove == currentBlockTime {
		// Allow the operation - this is part of the same bulk transaction
		return nil
	}

	// Otherwise, check normal rate limiting
	if ctx.BlockTime().UTC().Unix()-int64(metadata.LastStakeMove) < int64(STAKE_MOVE_WINDOW.Seconds()) {
		return utils.LavaFormatWarning(
			fmt.Sprintf("provider must wait %s between stake moves", STAKE_MOVE_WINDOW),
			nil,
			utils.LogAttr("provider", provider),
			utils.LogAttr("last_move_time", metadata.LastStakeMove),
		)
	}

	return nil
}

// updateStakeMoveTimestamp updates the provider's timestamp
// This ensures bulk operations only update the timestamp once at the end of the transaction
func (k Keeper) updateStakeMoveTimestamp(ctx sdk.Context, provider string) error {
	metadata, err := k.epochStorageKeeper.GetMetadata(ctx, provider)
	if err != nil {
		// If metadata doesn't exist, skip timestamp update
		// The provider will be treated as fresh on the next move
		return nil
	}

	currentBlockTime := uint64(ctx.BlockTime().UTC().Unix())

	// Only update if not already updated in this block (for bulk operations)
	if metadata.LastStakeMove != currentBlockTime {
		metadata.LastStakeMove = currentBlockTime
		k.epochStorageKeeper.SetMetadata(ctx, metadata)
	}

	return nil
}

func (k Keeper) StakeNewEntry(ctx sdk.Context, validator, creator, chainID string, amount sdk.Coin, endpoints []epochstoragetypes.Endpoint, geolocation int32, delegationLimit sdk.Coin, delegationCommission uint64, provider string, description stakingtypes.Description) error {
	logger := k.Logger(ctx)
	specChainID := chainID

	metadata, err := k.epochStorageKeeper.GetMetadata(ctx, provider)
	if err != nil {
		// first provider with this address
		metadata = epochstoragetypes.ProviderMetadata{
			Provider:         provider,
			Vault:            creator,
			Chains:           []string{chainID},
			TotalDelegations: sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), sdk.ZeroInt()),
		}
	} else {
		metadata.Chains = lavaslices.AddUnique(metadata.Chains, chainID)
	}

	vaultMetadata, found := k.epochStorageKeeper.GetProviderMetadataByVault(ctx, creator)
	if found {
		if vaultMetadata.Provider != provider || vaultMetadata.Vault != creator {
			return utils.LavaFormatWarning("provider metadata mismatch", fmt.Errorf("provider metadata mismatch"),
				utils.LogAttr("provider", provider),
				utils.LogAttr("vault", creator),
			)
		}
	}

	spec, err := k.specKeeper.GetExpandedSpec(ctx, specChainID)
	if err != nil || !spec.Enabled {
		return utils.LavaFormatWarning("spec not found or not active", err,
			utils.Attribute{Key: "spec", Value: specChainID},
		)
	}

	// if we get here, the spec is active and supported
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

	nextEpoch, err := k.epochStorageKeeper.GetNextEpoch(ctx, uint64(ctx.BlockHeight()))
	if err != nil {
		return utils.LavaFormatWarning("cannot get next epoch to count past delegations", err,
			utils.LogAttr("provider", senderAddr.String()),
			utils.LogAttr("block", nextEpoch),
		)
	}

	existingEntry, entryExists := k.epochStorageKeeper.GetStakeEntryCurrent(ctx, chainID, creator)
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
				if delegationCommission != metadata.DelegateCommission ||
					!amount.Equal(existingEntry.Stake) {
					return utils.LavaFormatWarning("only vault address can change stake/delegation related properties of the stake entry", fmt.Errorf("invalid modification request for stake entry"),
						utils.LogAttr("creator", creator),
						utils.LogAttr("vault", existingEntry.Vault),
						utils.LogAttr("provider", provider),
						utils.LogAttr("description", description.String()),
						utils.LogAttr("current_delegation_commission", metadata.DelegateCommission),
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
			if ctx.BlockTime().UTC().Unix()-int64(metadata.LastChange) < int64(CHANGE_WINDOW.Seconds()) {
				if delegationCommission != metadata.DelegateCommission {
					return utils.LavaFormatWarning(fmt.Sprintf("stake entry commmision or delegate limit can only be changes once in %s", CHANGE_WINDOW), nil,
						utils.LogAttr("last_change_time", metadata.LastChange))
				}
			}

			// check that the change is not mode than MAX_CHANGE_RATE
			if int64(delegationCommission)-int64(metadata.DelegateCommission) > MAX_CHANGE_RATE {
				return utils.LavaFormatWarning("stake entry commission increase too high", fmt.Errorf("commission change cannot increase by more than %d at a time", MAX_CHANGE_RATE),
					utils.LogAttr("original_commission", metadata.DelegateCommission),
					utils.LogAttr("wanted_commission", delegationCommission),
				)
			}
		}

		// we dont change stakeAppliedBlocks and chain once they are set, if they need to change, unstake first
		beforeAmount := existingEntry.Stake
		increase := amount.Amount.GT(existingEntry.Stake.Amount)
		decrease := amount.Amount.LT(existingEntry.Stake.Amount)
		existingEntry.Geolocation = geolocation
		existingEntry.Endpoints = endpointsVerified
		metadata.Description = description
		metadata.DelegateCommission = delegationCommission
		metadata.LastChange = uint64(ctx.BlockTime().UTC().Unix())
		existingEntry.Stake = amount

		k.epochStorageKeeper.SetStakeEntryCurrent(ctx, existingEntry)
		k.epochStorageKeeper.SetMetadata(ctx, metadata)
		if increase {
			// delegate the difference
			diffAmount := amount.Sub(beforeAmount)
			err = k.dualstakingKeeper.DelegateFull(ctx, existingEntry.Vault, validator, existingEntry.Address, diffAmount, true)
			if err != nil {
				details = append(details, utils.Attribute{Key: "neededStake", Value: amount.Sub(existingEntry.Stake).String()})
				return utils.LavaFormatWarning("failed to increase stake", err,
					details...,
				)
			}

			// automatically unfreeze the provider if it was frozen due to stake below min spec stake
			minSpecStake := k.specKeeper.GetMinStake(ctx, chainID)
			if beforeAmount.IsLT(minSpecStake) && existingEntry.IsFrozen() && !existingEntry.IsJailed(ctx.BlockTime().UTC().Unix()) && amount.IsGTE(minSpecStake) {
				existingEntry.UnFreeze(nextEpoch)
				k.epochStorageKeeper.SetStakeEntryCurrent(ctx, existingEntry)
			}
		} else if decrease {
			// unbond the difference
			diffAmount := beforeAmount.Sub(amount)
			err = k.dualstakingKeeper.UnbondFull(ctx, existingEntry.Vault, validator, existingEntry.Address, diffAmount, true)
			if err != nil {
				details = append(details, utils.Attribute{Key: "neededStake", Value: amount.Sub(existingEntry.Stake).String()})
				return utils.LavaFormatWarning("failed to decrease stake", err,
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
		providerStakeEntry, entryExists := k.epochStorageKeeper.GetStakeEntryCurrent(ctx, chainID, provider)
		if entryExists {
			return utils.LavaFormatWarning("configured provider exists", fmt.Errorf("new provider not staked"),
				utils.LogAttr("provider", provider),
				utils.LogAttr("chain_id", chainID),
				utils.LogAttr("existing_vault", providerStakeEntry.Vault),
			)
		}
	}

	if creator != metadata.Vault {
		return utils.LavaFormatWarning("creator does not match the provider vault", err,
			utils.Attribute{Key: "vault", Value: metadata.Vault},
			utils.Attribute{Key: "creator", Value: creator},
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

	// if there are registered delegations to the provider, count them in the delegateTotal
	delegateTotal := sdk.ZeroInt()
	stakeAmount := amount

	// creating a new provider, fetch old delegation
	if len(metadata.Chains) == 1 {
		delegations, err := k.dualstakingKeeper.GetProviderDelegators(ctx, provider)
		if err == nil {
			for _, d := range delegations {
				if d.Delegator == creator {
					stakeAmount = stakeAmount.Add(d.Amount)
				} else {
					metadata.TotalDelegations = metadata.TotalDelegations.Add(d.Amount)
				}
			}
		}
	}

	if stakeAmount.IsLT(k.dualstakingKeeper.MinSelfDelegation(ctx)) { // we count on this to also check the denom
		return utils.LavaFormatWarning("insufficient stake amount", fmt.Errorf("stake amount smaller than MinSelfDelegation"),
			utils.Attribute{Key: "spec", Value: specChainID},
			utils.Attribute{Key: "provider", Value: creator},
			utils.Attribute{Key: "stake", Value: amount},
			utils.Attribute{Key: "minSelfDelegation", Value: k.dualstakingKeeper.MinSelfDelegation(ctx).String()},
		)
	}

	stakeEntry := epochstoragetypes.StakeEntry{
		Stake:             stakeAmount,
		Address:           provider,
		StakeAppliedBlock: stakeAppliedBlock,
		Endpoints:         endpointsVerified,
		Geolocation:       geolocation,
		Chain:             chainID,
		DelegateTotal:     sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), delegateTotal),
		Vault:             creator, // the stake-provider TX creator is always regarded as the vault address
	}

	metadata.DelegateCommission = delegationCommission
	metadata.LastChange = uint64(ctx.BlockTime().UTC().Unix())
	metadata.Description = description

	k.epochStorageKeeper.SetMetadata(ctx, metadata)
	k.epochStorageKeeper.SetStakeEntryCurrent(ctx, stakeEntry)

	err = k.dualstakingKeeper.DelegateFull(ctx, stakeEntry.Vault, validator, stakeEntry.Address, amount, true)
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
	stakeEntry, found := k.epochStorageKeeper.GetStakeEntryCurrent(ctx, chainID, provider)
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
