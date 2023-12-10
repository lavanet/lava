package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/rewards/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Pools(goCtx context.Context, req *types.QueryPoolsRequest) (*types.QueryPoolsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	details := []types.PoolInfo{
		{
			Name:           string(types.ValidatorsBlockRewardsPoolName),
			Balance:        sdk.NewCoin(epochstoragetypes.TokenDenom, k.TotalPoolTokens(ctx, types.ValidatorsBlockRewardsPoolName)),
			BlocksToRefill: k.BlocksToNextTimerExpiry(ctx),
		},
		{
			Name:           string(types.ValidatorsRewardsPoolName),
			Balance:        sdk.NewCoin(epochstoragetypes.TokenDenom, k.TotalPoolTokens(ctx, types.ValidatorsRewardsPoolName)),
			BlocksToRefill: 0, // the main validators pool never refills
		},
	}

	return &types.QueryPoolsResponse{Details: details}, nil
}
