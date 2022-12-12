package keeper

import (
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
)

func (k Keeper) StakeNewEntry(ctx sdk.Context, provider bool, creator string, chainID string, amount sdk.Coin, endpoints []epochstoragetypes.Endpoint, geolocation uint64, vrfpk string) error {
	logger := k.Logger(ctx)
	stake_type := func() string {
		if provider {
			return epochstoragetypes.ProviderKey
		}
		return epochstoragetypes.ClientKey
	}

	// TODO: basic validation for chain ID
	specChainID := chainID

	foundAndActive, _ := k.specKeeper.IsSpecFoundAndActive(ctx, specChainID)
	if !foundAndActive {
		details := map[string]string{"spec": specChainID}
		return utils.LavaError(ctx, logger, "stake_"+stake_type()+"_spec", details, "spec not found or not active")
	}
	var minStake sdk.Coin
	if provider {
		minStake = k.MinStakeProvider(ctx)
	} else {
		minStake = k.MinStakeClient(ctx)
	}
	// if we get here, the spec is active and supported

	if amount.IsLT(minStake) { // we count on this to also check the denom
		details := map[string]string{"spec": specChainID, stake_type(): creator, "stake": amount.String(), "minStake": minStake.String()}
		return utils.LavaError(ctx, logger, "stake_"+stake_type()+"_amount", details, "insufficient "+stake_type()+" stake amount")
	}
	senderAddr, err := sdk.AccAddressFromBech32(creator)
	if err != nil {
		details := map[string]string{stake_type(): creator, "error": err.Error()}
		return utils.LavaError(ctx, logger, "stake_"+stake_type()+"_addr", details, "invalid "+stake_type()+" address")
	}
	// define the function here for later use
	verifySufficientAmountAndSendToModule := func(ctx sdk.Context, k Keeper, addr sdk.AccAddress, neededAmount sdk.Coin) error {
		if k.bankKeeper.GetBalance(ctx, addr, epochstoragetypes.TokenDenom).IsLT(neededAmount) {
			return fmt.Errorf("insufficient balance for staking %s current balance: %s", neededAmount, k.bankKeeper.GetBalance(ctx, addr, epochstoragetypes.TokenDenom))
		}
		err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, addr, types.ModuleName, []sdk.Coin{neededAmount})
		if err != nil {
			return fmt.Errorf("invalid transfer coins to module, %s", err)
		}
		return nil
	}
	geolocations := k.specKeeper.GeolocationCount(ctx)
	if geolocation == 0 || geolocation > (1<<geolocations) {
		details := map[string]string{"geolocation": strconv.FormatUint(geolocation, 10)}
		return utils.LavaError(ctx, logger, "stake_"+stake_type()+"_geolocation", details, "can't register for no geolocation or geolocation outside zones")
	}
	if provider {
		err := k.validateGeoLocationAndApiInterfaces(ctx, endpoints, geolocation, specChainID)
		if err != nil {
			details := map[string]string{stake_type(): creator, "error": err.Error(), "endpoints": fmt.Sprintf("%v", endpoints), "Chain": specChainID, "geolocation": strconv.FormatUint(geolocation, 10)}
			return utils.LavaError(ctx, logger, "stake_"+stake_type()+"_endpoints", details, "invalid "+stake_type()+" endpoints implementation for the given spec")
		}
	} else {
		// clients need to provide their VRF PK before running to limit brute forcing the random functions
		err := utils.VerifyVRF(vrfpk)
		if err != nil {
			details := map[string]string{stake_type(): creator, "error": err.Error()}
			return utils.LavaError(ctx, logger, "stake_"+stake_type()+"_vrfpk", details, "invalid "+stake_type()+" stake: invalid vrf pk, must provide a valid verification key")
		}
	}

	// new staking takes effect from the next block
	blockDeadline := uint64(ctx.BlockHeight()) + 1

	existingEntry, entryExists, indexInStakeStorage := k.epochStorageKeeper.GetStakeEntryByAddressCurrent(ctx, stake_type(), chainID, senderAddr)
	if entryExists {
		// modify the entry
		if existingEntry.Address != creator {
			details := map[string]string{"spec": specChainID, stake_type(): senderAddr.String(), stake_type(): creator}
			utils.LavaError(ctx, logger, "stake_"+stake_type()+"_panic", details, "returned stake entry by address doesn't match sender address!")
		}
		details := map[string]string{"spec": specChainID, stake_type(): senderAddr.String(), "deadline": strconv.FormatUint(blockDeadline, 10), "stake": amount.String()}
		if existingEntry.Stake.IsLT(amount) {
			// increasing stake is allowed
			err := verifySufficientAmountAndSendToModule(ctx, k, senderAddr, amount.Sub(existingEntry.Stake))
			if err != nil {
				details["error"] = err.Error()
				details["neededStake"] = amount.Sub(existingEntry.Stake).String()
				return utils.LavaError(ctx, logger, "stake_"+stake_type()+"_update_amount", details, "insufficient funds to pay for difference in stake")
			}

			// TODO: create a new entry entirely because then we can keep the copies of this list as pointers only
			// then we need to change the Copy of StoreCurrentEpochStakeStorage to copy of the pointers only
			// must also change the unstaking to create a new entry entirely

			// paid the difference to module
			existingEntry.Stake = amount
			// we dont change vrfpk, deadlines and chain once they are set, if they need to change, unstake first
			existingEntry.Geolocation = geolocation
			existingEntry.Endpoints = endpoints
			k.epochStorageKeeper.ModifyStakeEntryCurrent(ctx, stake_type(), chainID, existingEntry, indexInStakeStorage)
			utils.LogLavaEvent(ctx, logger, stake_type()+"_stake_update", details, "Changing Staked "+stake_type())
			return nil
		}
		details["existingStake"] = existingEntry.Stake.String()
		return utils.LavaError(ctx, logger, "stake_"+stake_type()+"_stake", details, "can't decrease stake for existing "+stake_type())
	}

	// entry isn't staked so add him
	details := map[string]string{"spec": specChainID, stake_type(): senderAddr.String(), "deadline": strconv.FormatUint(blockDeadline, 10), "stake": amount.String(), "geolocation": strconv.FormatUint(geolocation, 10)}
	err = verifySufficientAmountAndSendToModule(ctx, k, senderAddr, amount)
	if err != nil {
		details["error"] = err.Error()
		return utils.LavaError(ctx, logger, "stake_"+stake_type()+"_new_amount", details, "insufficient amount to pay for stake")
	}

	stakeEntry := epochstoragetypes.StakeEntry{Stake: amount, Address: creator, Deadline: blockDeadline, Endpoints: endpoints, Geolocation: geolocation, Chain: chainID, Vrfpk: vrfpk}
	k.epochStorageKeeper.AppendStakeEntryCurrent(ctx, stake_type(), chainID, stakeEntry)
	appended := false
	if !provider {
		// this is done so consumers can use services upon staking for the first time and dont have to wait for the next epoch
		appended, err = k.epochStorageKeeper.BypassCurrentAndAppendNewEpochStakeEntry(ctx, stake_type(), chainID, stakeEntry)

		if err != nil {
			details["error"] = err.Error()
			return utils.LavaError(ctx, logger, "stake_"+stake_type()+"_epoch", details, "could not append epoch stake entries")
		}
	}
	details["effectiveImmediately"] = strconv.FormatBool(appended)
	utils.LogLavaEvent(ctx, logger, stake_type()+"_stake_new", details, "Adding Staked "+stake_type())
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
