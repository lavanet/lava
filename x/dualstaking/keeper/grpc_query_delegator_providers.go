package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v4/x/dualstaking/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) DelegatorProviders(goCtx context.Context, req *types.QueryDelegatorProvidersRequest) (res *types.QueryDelegatorProvidersResponse, err error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	providers, err := k.GetDelegatorProviders(ctx, req.Delegator)
	if err != nil {
		return nil, err
	}

	var delegations []types.Delegation
	for _, provider := range providers {
		delegation, found := k.GetDelegation(ctx, provider, req.Delegator)
		if found {
			delegations = append(delegations, delegation)
		}
	}

	return &types.QueryDelegatorProvidersResponse{Delegations: delegations}, nil
}
