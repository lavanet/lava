package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/subscription/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ListProjects(goCtx context.Context, req *types.QueryListProjectsRequest) (*types.QueryListProjectsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	_, found := k.GetSubscription(ctx, req.Subscription)
	if !found {
		return nil, utils.LavaFormatWarning("subscription not found", sdkerrors.ErrKeyNotFound,
			utils.Attribute{Key: "subscription", Value: req.Subscription},
		)
	}

	projects := k.projectsKeeper.GetAllProjectsForSubscription(ctx, req.Subscription)

	return &types.QueryListProjectsResponse{Projects: projects}, nil
}
