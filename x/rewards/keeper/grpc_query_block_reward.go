package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/rewards/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) BlockReward(goCtx context.Context, req *types.QueryBlockRewardRequest) (*types.QueryBlockRewardResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// get params for validator rewards calculation
	bondedTargetFactor := k.bondedTargetFactor(ctx)
	blocksToNextTimerExpiry := k.BlocksToNextTimerExpiry(ctx)

	// get validator block pool balance
	blockPoolBalance := k.TotalPoolTokens(ctx, types.ValidatorsRewardsDistributionPoolName)

	// validators bonus rewards = (blockPoolBalance * bondedTargetFactor) / blocksToNextTimerExpiry
	validatorsRewards := bondedTargetFactor.MulInt(blockPoolBalance).TruncateInt().QuoRaw(blocksToNextTimerExpiry)
	reward := sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), validatorsRewards)

	return &types.QueryBlockRewardResponse{Reward: reward}, nil
}
