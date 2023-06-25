package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/subscription/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Current(goCtx context.Context, req *types.QueryCurrentRequest) (*types.QueryCurrentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	res := types.QueryCurrentResponse{}

	block := uint64(ctx.BlockHeight())

	var sub types.Subscription
	if found := k.subsFS.FindEntry(ctx, req.Consumer, block, &sub); found {
		res.Sub = &sub
	}

	return &res, nil
}
