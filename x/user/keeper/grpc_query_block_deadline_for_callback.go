package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/user/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) BlockDeadlineForCallback(c context.Context, req *types.QueryGetBlockDeadlineForCallbackRequest) (*types.QueryGetBlockDeadlineForCallbackResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetBlockDeadlineForCallback(ctx)
	if !found {
		return nil, status.Error(codes.InvalidArgument, "not found")
	}

	return &types.QueryGetBlockDeadlineForCallbackResponse{BlockDeadlineForCallback: val}, nil
}
