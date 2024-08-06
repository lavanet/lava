package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/rewards/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) IprpcSpecReward(goCtx context.Context, req *types.QueryIprpcSpecRewardRequest) (*types.QueryIprpcSpecRewardResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	iprpcRewards := k.GetAllIprpcReward(ctx)
	currentMonthId := k.GetIprpcRewardsCurrentId(ctx)

	if req.Spec == "" {
		return &types.QueryIprpcSpecRewardResponse{IprpcRewards: iprpcRewards, CurrentMonthId: currentMonthId}, nil
	}

	specIprpcRewards := []types.IprpcReward{}
	for _, iprpcReward := range iprpcRewards {
		for _, specFund := range iprpcReward.SpecFunds {
			if specFund.Spec == req.Spec {
				specIprpcReward := types.IprpcReward{Id: iprpcReward.Id, SpecFunds: []types.Specfund{specFund}}
				specIprpcRewards = append(specIprpcRewards, specIprpcReward)
				break
			}
		}
	}

	return &types.QueryIprpcSpecRewardResponse{IprpcRewards: specIprpcRewards, CurrentMonthId: currentMonthId}, nil
}
