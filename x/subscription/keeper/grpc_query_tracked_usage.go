package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/subscription/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) TrackedUsage(goCtx context.Context, req *types.QuerySubscriptionTrackedUsageRequest) (*types.QuerySubscriptionTrackedUsageResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	res := types.QuerySubscriptionTrackedUsageResponse{}

	sub, _ := k.GetSubscription(ctx, req.Subscription)

	res.Subscription = &sub
	res.Usage, res.TotalUsage = k.GetSubTrackedCuInfo(ctx, req.Subscription, uint64(ctx.BlockHeader().Height))

	return &res, nil
}
