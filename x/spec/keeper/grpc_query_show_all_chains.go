package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/spec/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ShowAllChains(goCtx context.Context, req *types.QueryShowAllChainsRequest) (*types.QueryShowAllChainsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	var names []string

	// get all spec
	allSpec := k.GetAllSpec(ctx)

	// iterate over specs and extract the chain's name
	for _, spec := range allSpec {
		names = append(names, spec.GetName())
	}

	return &types.QueryShowAllChainsResponse{ChainNames: names}, nil
}
