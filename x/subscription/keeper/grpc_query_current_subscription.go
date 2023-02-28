package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/subscription/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) CurrentSubscription(goCtx context.Context, req *types.QueryCurrentSubscriptionRequest) (*types.QueryCurrentSubscriptionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	res := types.QueryCurrentSubscriptionResponse{}

	// TODO: should indeed allow anyone to query about anyone?
	sub, found := k.GetSubscription(ctx, req.Consumer)
	if found {
		res.Sub = sub
	}

	return &res, nil
}
