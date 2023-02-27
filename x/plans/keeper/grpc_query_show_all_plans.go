package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/plans/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ShowAllPlans(goCtx context.Context, req *types.QueryShowAllPlansRequest) (*types.QueryShowAllPlansResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	var allPlansInfo []*types.ShowAllPlansInfoStruct
	ctx := sdk.UnwrapSDKContext(goCtx)

	// get all plan' unique indices
	planIndices := k.plansFs.GetAllEntryIndices(ctx)

	// go over all the plan' unique indices
	for _, planIndex := range planIndices {
		planInfoStruct := types.ShowAllPlansInfoStruct{}

		// get the latest version plans
		var latestVersionPlan types.Plan
		err := k.plansFs.FindEntry(ctx, planIndex, uint64(ctx.BlockHeight()), &latestVersionPlan)
		if err != nil {
			return nil, utils.LavaError(ctx, ctx.Logger(), "get_plan_latest_version", map[string]string{"err": err.Error(), "planIndex": planIndex}, "could not get the latest version of the plan")
		}

		// set the planInfoStruct
		planInfoStruct.Index = latestVersionPlan.GetIndex()
		planInfoStruct.Name = latestVersionPlan.GetName()
		planInfoStruct.Price = latestVersionPlan.GetPrice()

		// append the planInfoStruct to the allPlansInfo list
		allPlansInfo = append(allPlansInfo, &planInfoStruct)
	}
	_ = ctx

	return &types.QueryShowAllPlansResponse{PlansInfo: allPlansInfo}, nil
}
