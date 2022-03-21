package keeper

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/user/types"
)

func (k msgServer) UnstakeUser(goCtx context.Context, msg *types.MsgUnstakeUser) (*types.MsgUnstakeUserResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	specName := msg.Spec
	err := specName.ValidateBasic() //TODO: basic validation, we dont want to read the entire spec list here
	if err != nil {
		return nil, err
	}

	// we can unstake disabled specs, but not missing ones
	_, found := k.Keeper.specKeeper.IsSpecFoundAndActive(ctx, specName.Name)
	if !found {
		return nil, errors.New("spec not found, can't unstake")
	}

	specStakeStorage, found := k.Keeper.GetSpecStakeStorage(ctx, specName.Name)
	if !found {
		// the spec storage is empty
		return nil, fmt.Errorf("can't unstake empty specStakeStorage for spec name: %s", specName.Name)
	}
	stakeStorage := specStakeStorage.StakeStorage
	found_staked_entry := false
	//TODO: improve the finding logic and the way Staked is saved looping a list is slow and bad
	for idx, stakedUser := range stakeStorage.StakedUsers {
		if stakedUser.Index == msg.Creator {
			// found entry
			found_staked_entry = true
			holdBlocks := k.Keeper.UnstakeHoldBlocks(ctx)
			blockHeight := uint64(ctx.BlockHeight())
			if msg.Deadline.Num < blockHeight+holdBlocks {
				// unstaking demands they wait until a cedrftain block height so we can catch frauds before they escape with the money
				stakedUser.Deadline.Num = blockHeight + holdBlocks
			}
			//TODO: store this list sorted by deadline so when we go over it in the timeout, we can do this efficiently
			unstakingUserAllSpecs := types.UnstakingUsersAllSpecs{
				Id:               0,
				Unstaking:        stakedUser,
				SpecStakeStorage: specStakeStorage,
			}
			k.Keeper.AppendUnstakingUsersAllSpecs(ctx, unstakingUserAllSpecs)
			currentDeadline, found := k.GetBlockDeadlineForCallback(ctx)
			if !found {
				panic("didn't find single variable BlockDeadlineForCallback")
			}
			if currentDeadline.Deadline.Num == 0 || currentDeadline.Deadline.Num > stakedUser.Deadline.Num {
				currentDeadline.Deadline.Num = stakedUser.Deadline.Num
				k.SetBlockDeadlineForCallback(ctx, currentDeadline)
			}
			// effeciently delete stakedUser from stakeStorage.Staked
			stakeStorage.StakedUsers[idx] = stakeStorage.StakedUsers[len(stakeStorage.StakedUsers)-1] // replace the element at delete index with the last one
			stakeStorage.StakedUsers = stakeStorage.StakedUsers[:len(stakeStorage.StakedUsers)-1]     // remove last element
			//should be unique so there's no reason to keep iterating
			break
		}
	}
	if !found_staked_entry {
		return nil, fmt.Errorf("can't unstake User, stake entry not found for address: %s", msg.Creator)
	}
	k.Keeper.SetSpecStakeStorage(ctx, specStakeStorage)
	_ = ctx

	return &types.MsgUnstakeUserResponse{}, nil
}
