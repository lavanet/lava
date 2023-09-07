package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/dualstaking/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) DelegatorProviders(goCtx context.Context, req *types.QueryDelegatorProvidersRequest) (res *types.QueryDelegatorProvidersResponse, err error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	epoch := uint64(ctx.BlockHeight())
	if req.WithPending {
		epoch, err = k.getNextEpoch(ctx)
		if err != nil {
			return nil, err
		}
	}

	providers, err := k.GetDelegatorProviders(ctx, req.Delegator, epoch)
	if err != nil {
		return nil, err
	}

	var delegations []types.Delegation
	for _, provider := range providers {
		providerDelegations, err := k.GetProviderDelegators(ctx, provider, epoch)
		if err != nil {
			utils.LavaFormatError("could not get provider's delegators", err,
				utils.Attribute{Key: "proivder", Value: provider},
			)
			continue
		}
		for _, delegation := range providerDelegations {
			if delegation.Delegator == req.Delegator {
				delegations = append(delegations, delegation)
			}
		}
	}

	return &types.QueryDelegatorProvidersResponse{Delegations: delegations}, nil
}
