package keeper

import (
	"context"
	"errors"
	"fmt"
	"strconv"

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
	_, found, _ := k.Keeper.specKeeper.IsSpecFoundAndActive(ctx, specName.Name)
	if !found {
		return nil, errors.New("spec not found, can't unstake")
	}
	// receiverAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	// if err != nil {
	// 	return nil, fmt.Errorf("invalid creator address %s error: %s", msg.Creator, err))
	// }
	specStakeStorage, found := k.Keeper.GetSpecStakeStorage(ctx, specName.Name)
	if !found {
		// the spec storage is empty
		return nil, fmt.Errorf("can't unstake empty specStakeStorage for spec name: %s", specName.Name)
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
			storageMap.Deadline.Num = msg.Deadline.Num
			if storageMap.Deadline.Num < blockHeight+holdBlocks {
				// unstaking demands they wait until a certain block height so we can catch frauds before they escape with the money
				storageMap.Deadline.Num = blockHeight + holdBlocks
			}
			if storageMap.Deadline.Num < blockHeight+k.userKeeper.BlocksToSave(ctx) {
				// protocol demands the stake stays in deposit until proofsOfWork for older blocks are no longer valid,
				// this is to prevent fraud and escaping with the money
				storageMap.Deadline.Num = blockHeight + k.userKeeper.BlocksToSave(ctx)
			}
			//TODO: store this list sorted by deadline so when we go over it in the timeout, we can do this efficiently
			unstakingServicerAllSpecs := types.UnstakingServicersAllSpecs{
				Id:               0,
				Unstaking:        &storageMap,
				SpecStakeStorage: &specStakeStorage,
			}
			k.Keeper.AppendUnstakingServicersAllSpecs(ctx, unstakingServicerAllSpecs)
			currentDeadline, found := k.GetBlockDeadlineForCallback(ctx)
			if !found {
				panic("didn't find single variable BlockDeadlineForCallback")
			}
			if currentDeadline.Deadline.Num == 0 || currentDeadline.Deadline.Num > storageMap.Deadline.Num {
				currentDeadline.Deadline.Num = storageMap.Deadline.Num
				k.SetBlockDeadlineForCallback(ctx, currentDeadline)
			}
			// effeciently delete storageMap from stakeStorage.Staked
			stakeStorage.Staked[idx] = stakeStorage.Staked[len(stakeStorage.Staked)-1] // replace the element at delete index with the last one
			stakeStorage.Staked = stakeStorage.Staked[:len(stakeStorage.Staked)-1]     // remove last element
			//should be unique so there's no reason to keep iterating
			eventAttributes := []sdk.Attribute{sdk.NewAttribute("servicer", msg.Creator), sdk.NewAttribute("deadline", strconv.FormatUint(storageMap.Deadline.Num, 10)), sdk.NewAttribute("stake", storageMap.Stake.String()), sdk.NewAttribute("requestedDeadline", strconv.FormatUint(msg.Deadline.Num, 10))}
			ctx.EventManager().EmitEvent(sdk.NewEvent("lava_servicer_unstake_schedule", eventAttributes...))
			break
		}
	}
	if !found_staked_entry {
		return nil, fmt.Errorf("can't unstake servicer, stake entry not found for address: %s", msg.Creator)
	}
	k.Keeper.SetSpecStakeStorage(ctx, specStakeStorage)

	return &types.MsgUnstakeServicerResponse{}, nil
}
