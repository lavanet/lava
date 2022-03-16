package keeper

import (
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
)

func (k Keeper) GetPairingForClient(ctx sdk.Context, block types.BlockNum, specID uint64, clientAddress sdk.AccAddress) ([]sdk.AccAddress, error) {
	//TODO: client stake needs to be verified
	spec, found := k.specKeeper.GetSpec(ctx, specID)
	if !found {
		return nil, errors.New(fmt.Sprintf("spec not found for id given: %s", specID))
	}
	specStakeStorage, found := k.GetSpecStakeStorage(ctx, spec.Name)
	if !found {
		return nil, errors.New(fmt.Sprintf("no specStakeStorage for spec name: %s", spec.Name))
	}
	stakedServicers := specStakeStorage.StakeStorage.Staked
	return k.calculatePairingForClient(ctx, stakedServicers, block, clientAddress)
}

func (k Keeper) calculatePairingForClient(ctx sdk.Context, stakedServicers []types.StakeMap, block types.BlockNum, clientAddress sdk.AccAddress) (addrList []sdk.AccAddress, err error) {
	//TODO: need to do the pairing function, right now i just return the staked servicers addresses
	for _, stakeMap := range stakedServicers {
		servicerAddress := stakeMap.Index
		servicerAccAddr, err := sdk.AccAddressFromBech32(servicerAddress)
		if err != nil {
			panic(fmt.Sprintf("invalid servicer address saved in keeper %s", servicerAddress))
		}
		addrList = append(addrList, servicerAccAddr)
	}
	return addrList, nil
}
