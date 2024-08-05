package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/plans/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) List(goCtx context.Context, req *types.QueryListRequest) (*types.QueryListResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	var allPlansInfo []types.ListInfoStruct
	ctx := sdk.UnwrapSDKContext(goCtx)

	// get all plan' unique indices
	planIndices := k.GetAllPlanIndices(ctx)

	// go over all the plan' unique indices
	for _, planIndex := range planIndices {
		// get the latest version plans
		latestVersionPlan, found := k.FindPlan(ctx, planIndex, uint64(ctx.BlockHeight()))
		if !found {
			return nil, utils.LavaFormatError("could not get the latest version of the plan", fmt.Errorf("plan not found"),
				utils.Attribute{Key: "planIndex", Value: planIndex},
			)
		}

		// set the planInfoStruct
		planInfoStruct := types.ListInfoStruct{
			Index:       latestVersionPlan.GetIndex(),
			Description: latestVersionPlan.GetDescription(),
			Price:       latestVersionPlan.GetPrice(),
		}

		// append the planInfoStruct to the allPlansInfo list
		allPlansInfo = append(allPlansInfo, planInfoStruct)
	}

	return &types.QueryListResponse{PlansInfo: allPlansInfo}, nil
}
