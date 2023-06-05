package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/projects/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Info(goCtx context.Context, req *types.QueryInfoRequest) (*types.QueryInfoResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	project, err := k.GetProjectForBlock(ctx, req.Project, uint64(ctx.BlockHeight()))
	if err != nil {
		return nil, err
	}

	return &types.QueryInfoResponse{Project: &project}, nil
}
