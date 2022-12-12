package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

func (k Keeper) JailEntry(ctx sdk.Context, account sdk.AccAddress, isProvider bool, chainID string, jailStartBlock uint64, jailBlocks uint64, bail sdk.Coin) error {
	// todo - provider will not get pairing and payment for this period
	return nil
}

func (k Keeper) BailEntry(ctx sdk.Context, account sdk.AccAddress, isProvider bool, chainID string, bail sdk.Coin) error {
	// todo - remove provider from jail and remove bail amount from account and add to stake
	return nil
}

func (k Keeper) SlashEntry(ctx sdk.Context, account sdk.AccAddress, isProvider bool, chainID string, percentage sdk.Dec) (sdk.Coin, error) {
	// TODO: jail user, and count problems
	return sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.ZeroInt()), nil
}
