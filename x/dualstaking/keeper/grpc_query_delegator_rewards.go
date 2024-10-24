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
	resProviders, err := k.DelegatorProviders(goCtx, &types.QueryDelegatorProvidersRequest{Delegator: req.Delegator})
	if err != nil {
		return nil, err
	}

	for _, delegation := range resProviders.Delegations {
		if delegation.Provider == req.Provider || req.Provider == "" {
			delegatorReward, found := k.GetDelegatorReward(ctx, delegation.Provider, delegation.Delegator)
			if found {
				reward := types.DelegatorRewardInfo{
					Provider: delegation.Provider,
					Amount:   delegatorReward.Amount,
				}
				rewards = append(rewards, reward)
			}
		}
	}

	return &types.QueryDelegatorRewardsResponse{Rewards: rewards}, nil
}

func (k Keeper) DelegatorRewardsList(goCtx context.Context, req *types.QueryDelegatorRewardsRequest) (res *types.QueryDelegatorRewardsResponse, err error) {
	return k.DelegatorRewards(goCtx, req)
}
