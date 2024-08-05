package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/rewards/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ShowIprpcData(goCtx context.Context, req *types.QueryShowIprpcDataRequest) (*types.QueryShowIprpcDataResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	cost := k.GetMinIprpcCost(ctx)
	subs := k.GetAllIprpcSubscription(ctx)

	return &types.QueryShowIprpcDataResponse{MinCost: cost, IprpcSubscriptions: subs}, nil
}
