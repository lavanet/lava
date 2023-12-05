package rewards

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/rewards/keeper"
	"github.com/lavanet/lava/x/rewards/types"
)

func BeginBlocker(ctx sdk.Context, k keeper.Keeper) {
	// get params for validator rewards calculation
	bondedTargetFactor := k.BondedTargetFactor(ctx)
	blocksToNextEmission := k.BlocksToNextEmission(ctx)

	// get validator block pool balance
	blockPoolBalance := k.TotalPoolTokens(ctx, types.ValidatorsBlockPoolName)

	// validators bonus rewards = (blockPoolBalance * bondedTargetFactor) / blocksToNextEmission
	validatorsRewards := bondedTargetFactor.MulInt(blockPoolBalance).TruncateInt().QuoRaw(blocksToNextEmission)
	coins := sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, validatorsRewards))

	// distribute rewards to validators (same as Cosmos mint module)
	err := k.AddCollectedFees(ctx, coins)
	if err != nil {
		panic(err)
	}
}
