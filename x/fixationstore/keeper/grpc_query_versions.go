package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/fixationstore/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k *Keeper) Versions(goCtx context.Context, req *types.QueryVersionsRequest) (*types.QueryVersionsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	var blocks []uint64
	for _, store := range k.fixationsStores {
		if store.GetStoreKey().Name() == req.StoreKey && store.GetStorePrefix() == req.Prefix {
			blocks = store.GetAllEntryVersions(ctx, req.Key)
			break
		}
	}

	return &types.QueryVersionsResponse{Blocks: blocks}, nil
}
