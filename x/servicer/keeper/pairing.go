package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
)

func (k Keeper) verifyPairingData(ctx sdk.Context, specID uint64, clientAddress sdk.AccAddress, isNew bool) (bool, error) {

	spec, found := k.specKeeper.GetSpec(ctx, specID)
	if !found {
		return false, fmt.Errorf("spec not found for id given: %d", specID)
	}

	verifiedUser := false
	//TODO: get the spec stake storage, for the new session, and the previous one if its in the overlapping blocks

	//we get the latest user specStorage, we dont need the list of all users, so we dont load the right one at the right storage
	// to overcome unstaking users problems, we add support for unstaking users, and make sure users can't unstake sooner than savedSessions*blocksPerSession
	userSpecStakeStorageForSpec, found := k.userKeeper.GetSpecStakeStorage(ctx, spec.Name)
	if !found {
		return false, fmt.Errorf("no user spec stake storage for spec %s", spec.Name)
	}
	allStakedUsersForSpec := userSpecStakeStorageForSpec.StakeStorage.StakedUsers
	for _, stakedUser := range allStakedUsersForSpec {
		userAddr, err := sdk.AccAddressFromBech32(stakedUser.Index)
		if err != nil {
			panic(fmt.Sprintf("invalid servicer address saved in keeper %s, err: %s", stakedUser.Index, err))
		}
		if userAddr.Equals(clientAddress) {
			verifiedUser = true
			break
		}
	}
	if !isNew {
		//TODO: add verification for unstaking users too
	}
	if !verifiedUser {
		return false, fmt.Errorf("client: %s isn't staked for spec %s", clientAddress, spec.Name)
	}
	return true, nil
}

//function used to get a new pairing from relayer and client
//first argument has all metadata, second argument is only the addresses
func (k Keeper) GetPairingForClient(ctx sdk.Context, specID uint64, clientAddress sdk.AccAddress) (servicerOptions []types.StakeMap, errorRet error) {
	k.verifyPairingData(ctx, specID, clientAddress, true)
	spec, _ := k.specKeeper.GetSpec(ctx, specID)
	specStakeStorage, found := k.GetSpecStakeStorage(ctx, spec.Name)
	if !found {
		return nil, fmt.Errorf("no specStakeStorage for spec name: %s", spec.Name)
	}
	stakedServicers := specStakeStorage.StakeStorage
	sessionStart, found := k.GetCurrentSessionStart(ctx)
	if !found {
		return nil, fmt.Errorf("did not find currentSessionStart in keeper")
	}
	servicerOptions, _, errorRet = k.calculatePairingForClient(ctx, stakedServicers, clientAddress, sessionStart)
	return
}

func (k Keeper) ValidatePairingForClient(ctx sdk.Context, specID uint64, clientAddress sdk.AccAddress, block types.BlockNum) (validAddresses []sdk.AccAddress, errorRet error) {
	k.verifyPairingData(ctx, specID, clientAddress, false)
	spec, _ := k.specKeeper.GetSpec(ctx, specID)
	// sessionStartBlock := k.GetSessionStartForBlock(ctx,block)
	specStakeStorage, found := k.GetSpecStakeStorage(ctx, spec.Name)
	if !found {
		return nil, fmt.Errorf("no specStakeStorage for spec name: %s", spec.Name)
	}
	stakedServicers := specStakeStorage.StakeStorage
	sessionStart := types.CurrentSessionStart{Block: block}
	_, validAddresses, errorRet = k.calculatePairingForClient(ctx, stakedServicers, clientAddress, sessionStart)
	return
}

func (k Keeper) calculatePairingForClient(ctx sdk.Context, stakedStorage *types.StakeStorage, clientAddress sdk.AccAddress, sessionStart types.CurrentSessionStart) (validServicers []types.StakeMap, addrList []sdk.AccAddress, err error) {
	if sessionStart.Block.Num > uint64(ctx.BlockHeight()) {
		panic(fmt.Sprintf("invalid session start saved in keeper %d, current block was %d", sessionStart.Block.Num, uint64(ctx.BlockHeight())))
	}
	stakedServicers := stakedStorage.Staked
	//create a list of valid servicers (deadline reached)
	for _, stakeMap := range stakedServicers {
		if stakeMap.Deadline.Num > uint64(ctx.BlockHeight()) {
			//servicer deadline wasn't reached yet
			continue
		}
		validServicers = append(validServicers, stakeMap)
	}

	//calculates a hash and randomly chooses the servicers
	k.returnSubsetOfServicersByStake(validServicers, k.ServicersToPairCount(ctx), uint(ctx.BlockHeight()))

	for _, stakeMap := range validServicers {
		servicerAddress := stakeMap.Index
		servicerAccAddr, err := sdk.AccAddressFromBech32(servicerAddress)
		if err != nil {
			panic(fmt.Sprintf("invalid servicer address saved in keeper %s, err: %s", servicerAddress, err))
		}
		addrList = append(addrList, servicerAccAddr)
	}
	return validServicers, addrList, nil
}

func (k Keeper) returnSubsetOfServicersByStake(servicersMaps []types.StakeMap, count uint64, block uint) (returnedServicers []types.StakeMap) {
	//TODO: need to do the pairing function, right now i just return the first staked servicers addresses
	for _, stakedServicer := range servicersMaps {
		if uint64(len(returnedServicers)) >= count {
			return returnedServicers
		}
		returnedServicers = append(returnedServicers, stakedServicer)
	}
	return returnedServicers
}
