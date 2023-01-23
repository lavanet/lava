package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
)

func (k Keeper) BurnClientStake(ctx sdk.Context, chainID string, clientAddressToBurn sdk.AccAddress, burnAmount sdk.Coin, failBurnOnLeftover bool) (bool, error) {
	if burnAmount.Denom != epochstoragetypes.TokenDenom {
		return false, fmt.Errorf("burn coin isn't right denom: %s", burnAmount.Denom)
	}
	// find the user in the stake list
	clientEntry, found, indexFound := k.epochStorageKeeper.GetStakeEntryByAddressCurrent(ctx, epochstoragetypes.ClientKey, chainID, clientAddressToBurn)
	if found {
		if clientEntry.Stake.IsLT(burnAmount) {
			if failBurnOnLeftover {
				return false, fmt.Errorf("couldn't burn coins for user: %v, because insufficient stake to burn: %s", clientEntry, burnAmount)
			}
			burnAmount.Amount = clientEntry.Stake.Amount
		}
		// reduce the requested burn from the entry
		clientEntry.Stake = clientEntry.Stake.Sub(burnAmount)
		// now we need to save the entry
		k.epochStorageKeeper.ModifyStakeEntryCurrent(ctx, epochstoragetypes.ClientKey, chainID, clientEntry, indexFound)

		spec, found := k.specKeeper.GetSpec(ctx, clientEntry.Chain)
		if !found {
			return true, fmt.Errorf("could not fetch spec %s in burn client stake", clientEntry.Chain)
		}
		if clientEntry.Stake.IsLT(spec.MinStakeClient) {
			// if user doesn't have enough stake to stay staked, we will unstake him now
			// err := k.UnstakeUser(ctx, chainID, specStakeStorage.StakeStorage.StakedUsers[idx].Index, types.BlockNum{Num: 0})
			err := k.UnstakeEntry(ctx, false, chainID, clientEntry.Address, types.UnstakeDescriptionInsufficientFunds)
			if err != nil {
				return true, fmt.Errorf("error unstaking user after burn: %v , error: %s", clientEntry, err)
			}
		}
		return true, nil
	}

	// didnt find user in staked users
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
				return false, fmt.Errorf("couldn't burn coins for user: %v, because insufficient stake to burn: %s", clientEntry, burnAmount)
			}
			burnAmount.Amount = clientEntry.Stake.Amount
		}
		// reduce the requested burn from the entry
		clientEntry.Stake = clientEntry.Stake.Sub(burnAmount)
		k.epochStorageKeeper.ModifyUnstakeEntry(ctx, epochstoragetypes.ClientKey, clientEntry, indexFound)
		return true, nil
	}
	// didn't find user
	return false, nil
}

func (k Keeper) CreditStakeEntry(ctx sdk.Context, chainID string, lookUpAddress sdk.AccAddress, creditAmount sdk.Coin, isProvider bool) (bool, error) {
	if creditAmount.Denom != epochstoragetypes.TokenDenom {
		return false, fmt.Errorf("burn coin isn't right denom: %s", creditAmount.Denom)
	}
	// find the user in the stake list
	var storageType string
	switch isProvider {
	case true:
		storageType = epochstoragetypes.ProviderKey
	case false:
		storageType = epochstoragetypes.ClientKey
	}
	entry, found, indexFound := k.epochStorageKeeper.GetStakeEntryByAddressCurrent(ctx, storageType, chainID, lookUpAddress)
	if found {
		// add the requested credit to the entry
		entry.Stake = entry.Stake.Add(creditAmount)
		// now we need to save the entry
		k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(creditAmount))
		k.epochStorageKeeper.ModifyStakeEntryCurrent(ctx, storageType, chainID, entry, indexFound)
		return true, nil
	}

	// didnt find user in staked users
	entry, found, _ = k.epochStorageKeeper.UnstakeEntryByAddress(ctx, epochstoragetypes.ClientKey, lookUpAddress)
	if found {
		// add the requested credit to the entry
		// appending new unstake entry in order to delay liquidity of the reward
		entry.Stake = creditAmount

		// now we need to save the entry
		k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(creditAmount))

		unstakeHoldBlocks, err := k.unstakeHoldBlocks(ctx, entry.Chain, isProvider)
		if err != nil {
			return false, err
		}

		return true, k.epochStorageKeeper.AppendUnstakeEntry(ctx, storageType, entry, unstakeHoldBlocks)
	}
	// didn't find user
	return false, nil
}
