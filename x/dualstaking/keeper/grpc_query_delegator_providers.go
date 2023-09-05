package keeper

import (
	"context"
	"fmt"

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
	if req.WithPendingDelegators {
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
		indices := k.delegationFS.GetAllEntryIndicesWithPrefix(ctx, provider)
		for _, ind := range indices {
			delegator, _, chainID := types.DelegationKeyDecode(ind)
			if delegator != req.Delegator {
				continue
			}
			var delegation types.Delegation
			found := k.delegationFS.FindEntry(ctx, ind, epoch, &delegation)
			if !found {
				utils.LavaFormatError("critical: provider found in delegatorFS but not in delegationFS", fmt.Errorf("provider delegation not found"),
					utils.Attribute{Key: "delegator", Value: delegator},
					utils.Attribute{Key: "provider", Value: provider},
					utils.Attribute{Key: "chainID", Value: chainID},
				)
				continue
			}
			delegations = append(delegations, delegation)
		}
	}

	return &types.QueryDelegatorProvidersResponse{Delegations: delegations}, nil
}
