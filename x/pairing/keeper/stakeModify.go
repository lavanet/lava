package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

func (k Keeper) CreditStakeEntry(ctx sdk.Context, chainID string, lookUpAddress sdk.AccAddress, creditAmount sdk.Coin) (bool, error) {
	if creditAmount.Denom != epochstoragetypes.TokenDenom {
		return false, fmt.Errorf("burn coin isn't right denom: %s", creditAmount.Denom)
	}
	// TODO need to find another solution for this
	// // find the user in the stake list
	// entry, found, indexFound := k.epochStorageKeeper.GetStakeEntryByAddressCurrent(ctx, chainID, lookUpAddress)
	// if found {
	// 	// add the requested credit to the entry
	// 	entry.Stake = entry.Stake.Add(creditAmount)
	// 	// now we need to save the entry
	// 	k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(creditAmount))
	// 	k.epochStorageKeeper.ModifyStakeEntryCurrent(ctx, chainID, entry, indexFound)
	// 	return true, nil
	// }

	// // didnt find user in staked users
	// entry, found, _ = k.epochStorageKeeper.UnstakeEntryByAddress(ctx, lookUpAddress)
	// if found {
	// 	// add the requested credit to the entry
	// 	// appending new unstake entry in order to delay liquidity of the reward
	// 	entry.Stake = creditAmount

	// 	// now we need to save the entry
	// 	k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(creditAmount))

	// 	unstakeHoldBlocks := k.getUnstakeHoldBlocks(ctx, entry.Chain)
	// 	return true, k.epochStorageKeeper.AppendUnstakeEntry(ctx, entry, unstakeHoldBlocks)
	// }
	// didn't find user
	return false, nil
}
