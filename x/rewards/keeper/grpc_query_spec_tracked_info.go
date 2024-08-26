package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/rewards/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) SpecTrackedInfo(goCtx context.Context, req *types.QuerySpecTrackedInfoRequest) (*types.QuerySpecTrackedInfoResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	res := types.QuerySpecTrackedInfoResponse{}
	res.Info = k.getAllBasePayForChain(ctx, req.ChainId, req.Provider)

	return &res, nil
}
