package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/subscription/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Current(goCtx context.Context, req *types.QueryCurrentRequest) (*types.QueryCurrentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	res := types.QueryCurrentResponse{}

	sub, found := k.GetSubscription(ctx, req.Consumer)
	if found {
		res.Sub = &sub
	}

	return &res, nil
}
