package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/lavanet/lava/x/user/types"
)

func (k Keeper) BurnUserStake(ctx sdk.Context, specName types.SpecName, userAddressToBurn sdk.AccAddress, burnAmount sdk.Coin, failBurnOnLeftover bool) (bool, error) {
	if burnAmount.Denom != "stake" {
		return false, fmt.Errorf("burn coin isn't stake: %s", burnAmount.Denom)
	}
	logger := k.Logger(ctx)
	//find the user in the stake list
	specStakeStorage, found := k.GetSpecStakeStorage(ctx, specName.Name)
	if !found {
		return false, fmt.Errorf("couldn't get specStakeStorage for spec name: %s", specName)
	}
	stakedUsers := specStakeStorage.StakeStorage.StakedUsers
	for idx, stakedUser := range stakedUsers {
		userAddr, err := sdk.AccAddressFromBech32(stakedUser.Index)
		if err != nil {
			panic(fmt.Sprintf("invalid user address saved on unstaking users storage in keeper %s, err: %s", stakedUser.Index, err))
		}
		if userAddr.Equals(userAddressToBurn) {
			if stakedUser.Stake.IsLT(burnAmount) {
				if failBurnOnLeftover {
					return false, fmt.Errorf("couldn't burn coins for user: %s, because insufficient stake to burn: %s", stakedUser, burnAmount)
				}
				burnAmount.Amount = stakedUser.Stake.Amount
			}
			specStakeStorage.StakeStorage.StakedUsers[idx].Stake = specStakeStorage.StakeStorage.StakedUsers[idx].Stake.Sub(burnAmount)
			k.SetSpecStakeStorage(ctx, specStakeStorage)

			if specStakeStorage.StakeStorage.StakedUsers[idx].Stake.IsLT(k.GetMinStake(ctx)) {
				//if user doesn't have enough stake to stay staked, we will unstake him now
				logger.Info("unstaking user", stakedUser.Index, "insufficient funds to stay staked", specStakeStorage.StakeStorage.StakedUsers[idx].Stake)
				err := k.UnstakeUser(ctx, specName, specStakeStorage.StakeStorage.StakedUsers[idx].Index, types.BlockNum{Num: 0})
				if err != nil {
					return true, fmt.Errorf("error unstaking user after burn: %s , error:", specStakeStorage.StakeStorage.StakedUsers[idx], err)
				}
			}
			return true, nil
		}
	}
	//didnt find user in staked users
	unstakingUsers, indexes := k.GetUnstakingUsersForSpec(ctx, specName)
	for idx, stakedUser := range unstakingUsers {
		userAddr, err := sdk.AccAddressFromBech32(stakedUser.Index)
		if err != nil {
			panic(fmt.Sprintf("invalid user address saved on unstaking users storage in keeper %s, err: %s", stakedUser.Index, err))
		}
		if userAddr.Equals(userAddressToBurn) {
			if stakedUser.Stake.IsLT(burnAmount) {
				if failBurnOnLeftover {
					return false, fmt.Errorf("couldn't burn coins for user: %s, because insufficient stake to burn: %s", stakedUser, burnAmount)
				}
				burnAmount.Amount = stakedUser.Stake.Amount
			}
			//indexes saves the index of the user in the unstaking storage
			unstakingUserIdxInKeeper := uint64(indexes[idx])
			unstakingUserEntry, found := k.GetUnstakingUsersAllSpecs(ctx, unstakingUserIdxInKeeper)
			if !found {
				panic(fmt.Sprintf("did not find unstaking user at index: %d, even though this entry was returned from unstkaing users list for spec: %s, indexes: %s", unstakingUserIdxInKeeper, unstakingUsers, indexes))
			}
			unstakingUserEntry.Unstaking.Stake = unstakingUserEntry.Unstaking.Stake.Sub(burnAmount)
			k.SetUnstakingUsersAllSpecs(ctx, unstakingUserEntry)
			return true, nil
		}
	}
	//didn't find user
	return false, nil
}
