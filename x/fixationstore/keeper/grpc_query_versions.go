package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/fixationstore/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k *Keeper) Versions(goCtx context.Context, req *types.QueryVersionsRequest) (*types.QueryVersionsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	var entries []types.Entry
	for _, store := range k.fixationsStores {
		if store.GetStoreKey().Name() == req.StoreKey && store.GetStorePrefix() == req.Prefix {
			blocks := store.GetAllEntryVersions(ctx, req.Key)
			for _, block := range blocks {
				entry, err := store.FindRawEntry(ctx, req.Key, block)
				if err != nil {
					return nil, err
				}
				entry.Index = req.Key // desanitize index so it will be printable
				entries = append(entries, entry)
			}
			break
		}
	}

	return &types.QueryVersionsResponse{Entries: entries}, nil
}
