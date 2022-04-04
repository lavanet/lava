package keeper

import (
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/user/types"
)

func (k Keeper) EnforceUserCUsUsageInSession(ctx sdk.Context, userStake *types.UserStake, totalCU uint64) error {
	var allowedCU uint64 = 0
	type stakeToCU struct {
		stake sdk.Coin
		cu    uint64
	}
	//TODO: create param dictionary for max CU per session per stake and compare it to totalCU
	// stakeToMaxCUMap := k.GetStakeToMaxCUInSessionMap(ctx)
	stakeToMaxCUMap := []stakeToCU{}
	stakeToMaxCUMap = append(stakeToMaxCUMap, stakeToCU{stake: sdk.Coin{Denom: "stake", Amount: sdk.NewIntFromUint64(0)}, cu: 5000})
	stakeToMaxCUMap = append(stakeToMaxCUMap, stakeToCU{stake: sdk.Coin{Denom: "stake", Amount: sdk.NewIntFromUint64(500)}, cu: 15000})
	stakeToMaxCUMap = append(stakeToMaxCUMap, stakeToCU{stake: sdk.Coin{Denom: "stake", Amount: sdk.NewIntFromUint64(2000)}, cu: 50000})
	stakeToMaxCUMap = append(stakeToMaxCUMap, stakeToCU{stake: sdk.Coin{Denom: "stake", Amount: sdk.NewIntFromUint64(5000)}, cu: 250000})
	stakeToMaxCUMap = append(stakeToMaxCUMap, stakeToCU{stake: sdk.Coin{Denom: "stake", Amount: sdk.NewIntFromUint64(100000)}, cu: 500000})
	stakeToMaxCUMap = append(stakeToMaxCUMap, stakeToCU{stake: sdk.Coin{Denom: "stake", Amount: sdk.NewIntFromUint64(9999900000)}, cu: 9999999999})
	for _, stakeToCU := range stakeToMaxCUMap {
		if userStake.Stake.IsGTE(stakeToCU.stake) {
			allowedCU = stakeToCU.cu
		} else {
			break
		}
	}
	if totalCU > allowedCU {
		k.LimitUserPairingsAndMarkForPenalty(ctx, userStake)
		return fmt.Errorf("user %s bypassed allowed CU %d by using: %d", userStake, allowedCU, totalCU)
	}
	return nil
}

func (k Keeper) LimitUserPairingsAndMarkForPenalty(ctx sdk.Context, userStake *types.UserStake) {
	//TODO: jail user, and count problems
}

func (k Keeper) UnstakeUser(ctx sdk.Context, specName types.SpecName, unstakingUser string, deadline types.BlockNum) error {
	specStakeStorage, found := k.GetSpecStakeStorage(ctx, specName.Name)
	logger := k.Logger(ctx)
	if !found {
		// the spec storage is empty
		details := map[string]string{"spec": specName.Name}
		return utils.LavaError(ctx, logger, "user_unstake_spec", details, "can't unstake empty specStakeStorage for spec name")
	}
	stakeStorage := specStakeStorage.StakeStorage
	found_staked_entry := false
	//TODO: improve the finding logic and the way Staked is saved looping a list is slow and bad
	for idx, stakedUser := range stakeStorage.StakedUsers {
		if stakedUser.Index == unstakingUser {
			// found entry
			found_staked_entry = true
			holdBlocks := k.UnstakeHoldBlocks(ctx)
			blockHeight := uint64(ctx.BlockHeight())

			if deadline.Num < blockHeight+holdBlocks {
				// unstaking demands they wait until a cedrftain block height so we can catch frauds before they escape with the money
				stakedUser.Deadline.Num = blockHeight + holdBlocks
			} else {
				stakedUser.Deadline.Num = deadline.Num
			}
			if stakedUser.Deadline.Num < blockHeight+k.BlocksToSave(ctx) {
				// protocol demands the stake stays in deposit until proofsOfWork for older blocks are no longer valid,
				// this is to prevent escaping with the money without paying
				stakedUser.Deadline.Num = blockHeight + k.BlocksToSave(ctx)
			}
			//TODO: store this list sorted by deadline so when we go over it in the timeout, we can do this efficiently
			unstakingUserAllSpecs := types.UnstakingUsersAllSpecs{
				Id:               0, //append overwrites the Id
				Unstaking:        stakedUser,
				SpecStakeStorage: specStakeStorage,
			}
			k.AppendUnstakingUsersAllSpecs(ctx, unstakingUserAllSpecs)
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

			details := map[string]string{"user": unstakingUser, "deadline": strconv.FormatUint(stakedUser.Deadline.Num, 10), "stake": stakedUser.Stake.String(), "requestedDeadline": strconv.FormatUint(deadline.Num, 10)}
			utils.LogLavaEvent(ctx, logger, "lava_user_unstake_schedule", details, "Scheduling Unstaking for User")
			break
		}
	}
	if !found_staked_entry {
		details := map[string]string{"user": unstakingUser}
		return utils.LavaError(ctx, logger, "user_unstake_entry", details, "can't unstake User, stake entry not found for address")
	}
	k.SetSpecStakeStorage(ctx, specStakeStorage)
	return nil
}
