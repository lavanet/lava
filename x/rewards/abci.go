package rewards

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/rewards/keeper"
	"github.com/lavanet/lava/x/rewards/types"
)

// BeginBlocker calculates the validators block rewards and transfers them to the fee collector
func BeginBlocker(ctx sdk.Context, k keeper.Keeper) {
	// get params for validator rewards calculation
	bondedTargetFactor := k.BondedTargetFactor(ctx)
	blocksToNextTimerExpiry := k.BlocksToNextTimerExpiry(ctx)

	// get validator block pool balance
	blockPoolBalance := k.TotalPoolTokens(ctx, types.ValidatorsBlockPoolName)

	// validators bonus rewards = (blockPoolBalance * bondedTargetFactor) / blocksToNextTimerExpiry
	validatorsRewards := bondedTargetFactor.MulInt(blockPoolBalance).TruncateInt().QuoRaw(blocksToNextTimerExpiry)
	coins := sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, validatorsRewards))

	// distribute rewards to validators (same as Cosmos mint module)
	err := k.AddCollectedFees(ctx, coins)
	if err != nil {
		panic(err)
	}
}
