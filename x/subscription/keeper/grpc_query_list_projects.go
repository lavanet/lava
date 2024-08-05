package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/subscription/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ListProjects(goCtx context.Context, req *types.QueryListProjectsRequest) (*types.QueryListProjectsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	block := uint64(ctx.BlockHeight())

	var sub types.Subscription
	if found := k.subsFS.FindEntry(ctx, req.Subscription, block, &sub); !found {
		return nil, utils.LavaFormatWarning("subscription not found", legacyerrors.ErrKeyNotFound,
			utils.Attribute{Key: "subscription", Value: req.Subscription},
		)
	}

	projects := k.projectsKeeper.GetAllProjectsForSubscription(ctx, req.Subscription)

	return &types.QueryListProjectsResponse{Projects: projects}, nil
}
