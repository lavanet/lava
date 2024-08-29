package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/dualstaking/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ProviderDelegators(goCtx context.Context, req *types.QueryProviderDelegatorsRequest) (res *types.QueryProviderDelegatorsResponse, err error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	epoch := uint64(ctx.BlockHeight())
	if req.WithPending {
		epoch = k.epochstorageKeeper.GetCurrentNextEpoch(ctx)
	}

	delegations, err := k.GetProviderDelegators(ctx, req.Provider, epoch)
	if err != nil {
		return nil, err
	}

	return &types.QueryProviderDelegatorsResponse{Delegations: delegations}, nil
}
