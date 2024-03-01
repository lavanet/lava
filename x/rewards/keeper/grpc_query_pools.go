package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/rewards/types"
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
			Balance: sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), k.TotalPoolTokens(ctx, types.ValidatorsRewardsDistributionPoolName, k.stakingKeeper.BondDenom(ctx))),
		},
		{
			Name:    string(types.ValidatorsRewardsAllocationPoolName),
			Balance: sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), k.TotalPoolTokens(ctx, types.ValidatorsRewardsAllocationPoolName, k.stakingKeeper.BondDenom(ctx))),
		},
		{
			Name:    string(types.ProviderRewardsDistributionPool),
			Balance: sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), k.TotalPoolTokens(ctx, types.ProviderRewardsDistributionPool, k.stakingKeeper.BondDenom(ctx))),
		},
		{
			Name:    string(types.ProvidersRewardsAllocationPool),
			Balance: sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), k.TotalPoolTokens(ctx, types.ProvidersRewardsAllocationPool, k.stakingKeeper.BondDenom(ctx))),
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
