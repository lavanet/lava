package keeper

import (
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/rewards/types"
)

// TotalPoolTokens gets the total tokens supply from a pool
func (k Keeper) TotalPoolTokens(ctx sdk.Context, pool types.Pool, denom string) math.Int {
	poolAddr := k.accountKeeper.GetModuleAddress(string(pool))
	return k.bankKeeper.GetBalance(ctx, poolAddr, denom).Amount
}

// BurnPoolTokens removes coins from a pool module account
func (k Keeper) BurnPoolTokens(ctx sdk.Context, pool types.Pool, amt math.Int, denom string) error {
	if !amt.IsPositive() {
		// skip as no coins need to be burned
		return nil
	}

	coins := sdk.NewCoins(sdk.NewCoin(denom, amt))

	return k.bankKeeper.BurnCoins(ctx, string(pool), coins)
}
