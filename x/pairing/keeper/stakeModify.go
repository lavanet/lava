package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k Keeper) CreditStakeEntry(ctx sdk.Context, chainID string, lookUpAddress sdk.AccAddress, creditAmount sdk.Coin) (bool, error) {
	// TODO: add a way to rewards for conflict
	return true, nil
}
