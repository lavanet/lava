package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
)

//first argument has all metadata, second argument is only the addresses
func (k Keeper) GetPairingForClient(ctx sdk.Context, block types.BlockNum, specID uint64, clientAddress sdk.AccAddress) ([]types.StakeMap, []sdk.AccAddress, error) {
	spec, found := k.specKeeper.GetSpec(ctx, specID)
	if !found {
		return nil, nil, fmt.Errorf("spec not found for id given: %d", specID)
	}
	specStakeStorage, found := k.GetSpecStakeStorage(ctx, spec.Name)
	if !found {
		return nil, nil, fmt.Errorf("no specStakeStorage for spec name: %s", spec.Name)
	}
	stakedServicers := specStakeStorage.StakeStorage.Staked

	verifiedUser := false
	userSpecStakeStorageForSpec, found := k.userKeeper.GetSpecStakeStorage(ctx, spec.Name)
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
	if !verifiedUser {
		return nil, nil, fmt.Errorf("client: %s isn't staked for spec %s", clientAddress, spec.Name)
	}

	return k.calculatePairingForClient(ctx, stakedServicers, block, clientAddress)
}

func (k Keeper) calculatePairingForClient(ctx sdk.Context, stakedServicers []types.StakeMap, block types.BlockNum, clientAddress sdk.AccAddress) (validServicers []types.StakeMap, addrList []sdk.AccAddress, err error) {
	//create a list of valid servicers (deadline reached)
	for _, stakeMap := range stakedServicers {
		if stakeMap.Deadline.Num > uint64(ctx.BlockHeight()) {
			//servicer deadline wasn't reached yet
			continue
		}
		validServicers = append(validServicers, stakeMap)
	}

	//calculates a hash and randomly chooses the servicers
	k.returnSubsetOfServicersByStake(validServicers, 2, uint(block.Num))

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

func (k Keeper) returnSubsetOfServicersByStake(servicersMaps []types.StakeMap, count uint, block uint) (returnedServicers []types.StakeMap) {
	//TODO: need to do the pairing function, right now i just return the first staked servicers addresses
	for _, stakedServicer := range servicersMaps {
		if uint(len(returnedServicers)) >= count {
			return returnedServicers
		}
		returnedServicers = append(returnedServicers, stakedServicer)
	}
	return returnedServicers
}
