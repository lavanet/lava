package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/dualstaking/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ProviderDelegators(goCtx context.Context, req *types.QueryProviderDelegatorsRequest) (*types.QueryProviderDelegatorsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	delegations, err := k.GetProviderDelegators(ctx, req.Provider)
	if err != nil {
		return nil, err
	}

	return &types.QueryProviderDelegatorsResponse{Delegations: delegations}, nil
}
