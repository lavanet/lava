package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v4/x/dualstaking/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) DelegatorRewards(goCtx context.Context, req *types.QueryDelegatorRewardsRequest) (res *types.QueryDelegatorRewardsResponse, err error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	var rewards []types.DelegatorRewardInfo
	allRewards := k.GetAllDelegatorReward(ctx)
	for _, delegatorReward := range allRewards {
		if delegatorReward.Delegator == req.Delegator && (delegatorReward.Provider == req.Provider || req.Provider == "") {
			rewards = append(rewards, types.DelegatorRewardInfo{
				Provider: delegatorReward.Provider,
				Amount:   delegatorReward.Amount,
			})
		}
	}

	return &types.QueryDelegatorRewardsResponse{Rewards: rewards}, nil
}

func (k Keeper) DelegatorRewardsList(goCtx context.Context, req *types.QueryDelegatorRewardsRequest) (res *types.QueryDelegatorRewardsResponse, err error) {
	return k.DelegatorRewards(goCtx, req)
}
