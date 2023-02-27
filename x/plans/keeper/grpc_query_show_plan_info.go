package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/plans/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ShowPlanInfo(goCtx context.Context, req *types.QueryShowPlanInfoRequest) (*types.QueryShowPlanInfoResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	var planToPrint types.Plan
	err := k.plansFs.FindEntry(ctx, req.GetPlanIndex(), uint64(ctx.BlockHeight()), &planToPrint)
	if err != nil {
		return nil, status.Error(codes.NotFound, "plan not found")
	}

	return &types.QueryShowPlanInfoResponse{PlanInfo: &planToPrint}, nil
}
