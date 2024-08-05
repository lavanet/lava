package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/rewards/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) BlockReward(goCtx context.Context, req *types.QueryBlockRewardRequest) (*types.QueryBlockRewardResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// get params for validator rewards calculation
	bondedTargetFactor := k.BondedTargetFactor(ctx)
	blocksToNextTimerExpiry := k.BlocksToNextTimerExpiry(ctx)

	// get validator block pool balance
	coins := k.TotalPoolTokens(ctx, types.ValidatorsRewardsDistributionPoolName)
	blockPoolBalance := coins.AmountOf(k.stakingKeeper.BondDenom(ctx))
	if blocksToNextTimerExpiry == 0 {
		return nil, utils.LavaFormatWarning("blocksToNextTimerExpiry is zero", fmt.Errorf("critical: Attempt to divide by zero"),
			utils.LogAttr("blocksToNextTimerExpiry", blocksToNextTimerExpiry),
			utils.LogAttr("blockPoolBalance", blockPoolBalance),
		)
	}

	// validators bonus rewards = (blockPoolBalance * bondedTargetFactor) / blocksToNextTimerExpiry
	validatorsRewards := bondedTargetFactor.MulInt(blockPoolBalance).QuoInt64(blocksToNextTimerExpiry).TruncateInt()
	reward := sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), validatorsRewards)

	return &types.QueryBlockRewardResponse{Reward: reward}, nil
}
