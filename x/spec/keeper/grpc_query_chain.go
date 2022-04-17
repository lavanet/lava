package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/spec/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) Chain(goCtx context.Context, req *types.QueryChainRequest) (*types.QueryChainResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	_, found, specID := k.IsSpecFoundAndActive(ctx, req.ChainID)
	if !found {
		return nil, fmt.Errorf("did not find chainID: %s", req.ChainID)
	}
	spec, found := k.GetSpec(ctx, specID)
	if !found {
		return nil, fmt.Errorf("did not find spec for specID: %d chainID: %s", specID, req.ChainID)
	}

	return &types.QueryChainResponse{Spec: spec}, nil
}
