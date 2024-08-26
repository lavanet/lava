package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/projects/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Developer(goCtx context.Context, req *types.QueryDeveloperRequest) (*types.QueryDeveloperResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	project, err := k.GetProjectForDeveloper(ctx, req.Developer, uint64(ctx.BlockHeight()))
	if err != nil {
		return nil, err
	}

	nextEpoch, err := k.epochstorageKeeper.GetNextEpoch(ctx, uint64(ctx.BlockHeight()))
	if err != nil {
		return nil, err
	}

	pendingProject, err := k.GetProjectForDeveloper(ctx, req.Developer, nextEpoch)
	if err != nil || project.Equal(pendingProject) {
		return &types.QueryDeveloperResponse{Project: &project}, nil
	} else {
		return &types.QueryDeveloperResponse{Project: &project, PendingProject: &pendingProject}, nil
	}
}
