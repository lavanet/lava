package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/dualstaking/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) DelegatorProviders(goCtx context.Context, req *types.QueryDelegatorProvidersRequest) (*types.QueryDelegatorProvidersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: implement logic
	_ = ctx

	return &types.QueryDelegatorProvidersResponse{StakeEntries: nil}, nil
}
