package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/rewards/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Pools(goCtx context.Context, req *types.QueryPoolsRequest) (*types.QueryPoolsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	pools := []types.PoolInfo{
		{
			Name:    string(types.ValidatorsRewardsDistributionPoolName),
			Balance: k.TotalPoolTokens(ctx, types.ValidatorsRewardsDistributionPoolName),
		},
		{
			Name:    string(types.ValidatorsRewardsAllocationPoolName),
			Balance: k.TotalPoolTokens(ctx, types.ValidatorsRewardsAllocationPoolName),
		},
		{
			Name:    string(types.ProviderRewardsDistributionPool),
			Balance: k.TotalPoolTokens(ctx, types.ProviderRewardsDistributionPool),
		},
		{
			Name:    string(types.ProvidersRewardsAllocationPool),
			Balance: k.TotalPoolTokens(ctx, types.ProvidersRewardsAllocationPool),
		},
		{
			Name:    string(types.IprpcPoolName),
			Balance: k.TotalPoolTokens(ctx, types.IprpcPoolName),
		},
	}

	estimatedBlocksToRefill := k.BlocksToNextTimerExpiry(ctx)
	timeToRefill := k.TimeToNextTimerExpiry(ctx)
	monthsLeft := k.AllocationPoolMonthsLeft(ctx)

	return &types.QueryPoolsResponse{
		Pools:                    pools,
		TimeToRefill:             timeToRefill,
		EstimatedBlocksToRefill:  estimatedBlocksToRefill,
		AllocationPoolMonthsLeft: monthsLeft,
	}, nil
}
