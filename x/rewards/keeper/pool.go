package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

// TotalPoolTokens gets the total tokens supply from a pool
func (k Keeper) TotalPoolTokens(ctx sdk.Context, pool string) math.Int {
	poolAddr := k.accountKeeper.GetModuleAddress(pool)
	return k.bankKeeper.GetBalance(ctx, poolAddr, epochstoragetypes.TokenDenom).Amount
}

// burnPoolTokens removes coins from a pool module account
func (k Keeper) burnPoolTokens(ctx sdk.Context, pool string, amt math.Int) error {
	if !amt.IsPositive() {
		// skip as no coins need to be burned
		return nil
	}

	coins := sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, amt))

	return k.bankKeeper.BurnCoins(ctx, pool, coins)
}
