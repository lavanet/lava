package keeper

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
)

func (k msgServer) UnstakeServicer(goCtx context.Context, msg *types.MsgUnstakeServicer) (*types.MsgUnstakeServicerResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	specName := msg.Spec
	err := specName.ValidateBasic() //TODO: basic validation, we dont want to read the entire spec list here
	if err != nil {
		return nil, err
	}

	// we can unstake disabled specs, but not missing ones
	_, found := k.Keeper.specKeeper.IsSpecFoundAndActive(ctx, specName.Name)
	if found != true {
		return nil, errors.New("spec not found, can't unstake")
	}
	// receiverAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	// if err != nil {
	// 	return nil, errors.New(fmt.Sprintf("invalid creator address %s error: %s", msg.Creator, err))
	// }
	specStakeStorage, found := k.Keeper.GetSpecStakeStorage(ctx, specName.Name)
	if found != true {
		// the spec storage is empty
		return nil, errors.New(fmt.Sprintf("can't unstake empty specStakeStorage for spec name: %s", specName.Name))
	}
	stakeStorage := specStakeStorage.StakeStorage
	found_staked_entry := false
	//TODO: improve the finding logic and the way Staked is saved looping a list is slow and bad
	for idx, storageMap := range stakeStorage.Staked {
		if storageMap.Index == msg.Creator {
			// found entry
			found_staked_entry = true
			holdBlocks := k.Keeper.UnstakeHoldBlocks(ctx)
			blockHeight := uint64(ctx.BlockHeight())
			if msg.Deadline.Num < blockHeight+holdBlocks {
				// unstaking demands they wait until a cedrftain block height so we can catch frauds before they escape with the money
				storageMap.Deadline.Num = blockHeight + holdBlocks
			}
			//TODO: store this list sorted by deadline so when we go over it in the timeout, we can do this efficiently
			stakeStorage.Unstaking = append(stakeStorage.Unstaking, storageMap)
			// effeciently delete an element
			stakeStorage.Staked[idx] = stakeStorage.Staked[len(stakeStorage.Staked)-1] // replace the element at delete index with the last one
			stakeStorage.Staked = stakeStorage.Staked[:len(stakeStorage.Staked)-1]     // remove last element
			break
		}
	}
	if !found_staked_entry {
		return nil, errors.New(fmt.Sprintf("can't unstake servicer, stake entry not found for address: %s", msg.Creator))
	}

	_ = ctx

	return &types.MsgUnstakeServicerResponse{}, nil
}
