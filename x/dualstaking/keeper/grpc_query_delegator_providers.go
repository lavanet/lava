package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/dualstaking/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) DelegatorProviders(goCtx context.Context, req *types.QueryDelegatorProvidersRequest) (*types.QueryDelegatorProvidersResponse, error) {
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
		providerDelegations, err := k.GetProviderDelegators(ctx, provider)
		if err != nil {
			utils.LavaFormatError("could not get provider's delegators", err,
				utils.Attribute{Key: "proivder", Value: provider},
			)
			continue
		}
		delegations = append(delegations, providerDelegations...)
	}

	return &types.QueryDelegatorProvidersResponse{Delegations: delegations}, nil
}
