package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/projects/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ShowDevelopersProject(goCtx context.Context, req *types.QueryShowDevelopersProjectRequest) (*types.QueryShowDevelopersProjectResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	project, err := k.GetProjectForDeveloper(ctx, req.Developer, uint64(ctx.BlockHeight()))
	if err != nil {
		return nil, err
	}

	return &types.QueryShowDevelopersProjectResponse{Project: &project}, nil
}
