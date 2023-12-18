package keeper

import (
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TotalPoolTokens gets the total tokens supply from a pool
func (k Keeper) TotalPoolTokens(ctx sdk.Context, pool string) math.Int {
	poolAddr := k.accountKeeper.GetModuleAddress(pool)
	return k.bankKeeper.GetBalance(ctx, poolAddr, k.stakingKeeper.BondDenom(ctx)).Amount
}

// BurnPoolTokens removes coins from a pool module account
func (k Keeper) BurnPoolTokens(ctx sdk.Context, pool string, amt math.Int) error {
	if !amt.IsPositive() {
		// skip as no coins need to be burned
		return nil
	}

	coins := sdk.NewCoins(sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), amt))

	return k.bankKeeper.BurnCoins(ctx, pool, coins)
}
