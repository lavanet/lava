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

func (k Keeper) ProviderDelegators(goCtx context.Context, req *types.QueryProviderDelegatorsRequest) (*types.QueryProviderDelegatorsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	_, err := sdk.AccAddressFromBech32(req.Provider)
	if err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	nextEpoch, err := k.getNextEpoch(ctx)
	if err != nil {
		return nil, err
	}

	var delegations []types.Delegation
	indices := k.delegationFS.GetAllEntryIndicesWithPrefix(ctx, req.Provider)
	for _, ind := range indices {
		var delegation types.Delegation
		found := k.delegationFS.FindEntry(ctx, ind, nextEpoch, &delegation)
		if !found {
			delegator, provider, chainID := types.DelegationKeyDecode(ind)
			utils.LavaFormatError("delegationFS entry index has no entry", fmt.Errorf("provider delegation not found"),
				utils.Attribute{Key: "delegator", Value: delegator},
				utils.Attribute{Key: "provider", Value: provider},
				utils.Attribute{Key: "chainID", Value: chainID},
			)
			continue
		}
		delegations = append(delegations, delegation)
	}

	return &types.QueryProviderDelegatorsResponse{Delegations: delegations}, nil
}
