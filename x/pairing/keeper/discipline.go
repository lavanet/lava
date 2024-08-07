package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
)

func (k Keeper) JailEntry(ctx sdk.Context, address string, chainID string, jailStartBlock, jailBlocks uint64, bail sdk.Coin) error {
	// todo - provider will not get pairing and payment for this period
	if !utils.IsBech32Address(address) {
		return fmt.Errorf("invalid address")
	}
	return nil
}

func (k Keeper) BailEntry(ctx sdk.Context, address string, chainID string, bail sdk.Coin) error {
	// todo - remove provider from jail and remove bail amount from address and add to stake
	if !utils.IsBech32Address(address) {
		return fmt.Errorf("invalid address")
	}
	return nil
}

func (k Keeper) SlashEntry(ctx sdk.Context, address string, chainID string, percentage sdk.Dec) (sdk.Coin, error) {
	// TODO: jail user, and count problems
	if !utils.IsBech32Address(address) {
		return sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), sdk.ZeroInt()), fmt.Errorf("invalid address")
	}
	return sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), sdk.ZeroInt()), nil
}
