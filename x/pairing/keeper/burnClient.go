package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

func (k Keeper) BurnClientStake(ctx sdk.Context, chainID string, clientAddressToBurn sdk.AccAddress, burnAmount sdk.Coin, failBurnOnLeftover bool) (bool, error) {
	if burnAmount.Denom != "stake" {
		return false, fmt.Errorf("burn coin isn't stake: %s", burnAmount.Denom)
	}
	logger := k.Logger(ctx)
	//find the user in the stake list
	clientEntry, found, indexFound := k.epochStorageKeeper.StakeEntryByAddress(ctx, epochstoragetypes.ClientKey, chainID, clientAddressToBurn)
	if found {
		if clientEntry.Stake.IsLT(burnAmount) {
			if failBurnOnLeftover {
				return false, fmt.Errorf("couldn't burn coins for user: %s, because insufficient stake to burn: %s", clientEntry, burnAmount)
			}
			burnAmount.Amount = clientEntry.Stake.Amount
		}
		//reduce the requested burn from the entry
		clientEntry.Stake = clientEntry.Stake.Sub(burnAmount)
		//now we need to save the entry
		k.epochStorageKeeper.ModifyStakeEntry(ctx, epochstoragetypes.ClientKey, chainID, clientEntry, indexFound)

		if clientEntry.Stake.IsLT(k.GetMinStakeClient(ctx)) {
			//if user doesn't have enough stake to stay staked, we will unstake him now
			logger.Info("unstaking client", clientEntry.Address, "insufficient funds to stay staked", clientEntry.Stake)
			// err := k.UnstakeUser(ctx, chainID, specStakeStorage.StakeStorage.StakedUsers[idx].Index, types.BlockNum{Num: 0})
			err := k.UnstakeEntry(ctx, false, chainID, clientEntry.Address)
			if err != nil {
				return true, fmt.Errorf("error unstaking user after burn: %s , error:", clientEntry, err)
			}
		}
		return true, nil
	}

	//didnt find user in staked users
	clientEntry, found, indexFound = k.epochStorageKeeper.UnstakeEntryByAddress(ctx, epochstoragetypes.ClientKey, clientAddressToBurn)
	if found {
		userAddr, err := sdk.AccAddressFromBech32(clientEntry.Address)
		if err != nil {
			panic(fmt.Sprintf("invalid user address saved on unstaking users storage in keeper %s, err: %s", clientEntry.Address, err))
		}
		if !userAddr.Equals(clientAddressToBurn) {
			panic(fmt.Sprintf("invalid user address found! %s != %s", clientEntry.Address, clientAddressToBurn))
		}
		if clientEntry.Stake.IsLT(burnAmount) {
			if failBurnOnLeftover {
				return false, fmt.Errorf("couldn't burn coins for user: %s, because insufficient stake to burn: %s", clientEntry, burnAmount)
			}
			burnAmount.Amount = clientEntry.Stake.Amount
		}
		//reduce the requested burn from the entry
		clientEntry.Stake = clientEntry.Stake.Sub(burnAmount)
		k.epochStorageKeeper.ModifyUnstakeEntry(ctx, epochstoragetypes.ClientKey, clientEntry, indexFound)
		return true, nil
	}
	//didn't find user
	return false, nil
}
