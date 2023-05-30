package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
)

func (k Keeper) CreditStakeEntry(ctx sdk.Context, chainID string, lookUpAddress sdk.AccAddress, creditAmount sdk.Coin) (bool, error) {
	if creditAmount.Denom != epochstoragetypes.TokenDenom {
		return false, fmt.Errorf("burn coin isn't right denom: %s", creditAmount.Denom)
	}
	// find the user in the stake list

	storageType := epochstoragetypes.ProviderKey

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

		unstakeHoldBlocks, err := k.unstakeHoldBlocks(ctx, entry.Chain)
		if err != nil {
			return false, err
		}

		return true, k.epochStorageKeeper.AppendUnstakeEntry(ctx, storageType, entry, unstakeHoldBlocks)
	}
	// didn't find user
	return false, nil
}
