package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/fixationstore/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k *Keeper) AllIndices(goCtx context.Context, req *types.QueryAllIndicesRequest) (*types.QueryAllIndicesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	var indices []string
	for _, store := range k.fixationsStores {
		if store.GetStoreKey().Name() == req.StoreKey && store.GetStorePrefix() == req.Prefix {
			indices = store.GetAllEntryIndices(ctx)
			break
		}
	}

	return &types.QueryAllIndicesResponse{Indices: indices}, nil
}
