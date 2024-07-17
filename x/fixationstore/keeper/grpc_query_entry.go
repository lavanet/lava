package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/fixationstore/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k *Keeper) Entry(goCtx context.Context, req *types.QueryEntryRequest) (res *types.QueryEntryResponse, err error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	var entry types.Entry
	for _, store := range k.fixationsStores {
		if store.GetStoreKey().Name() == req.StoreKey && store.GetStorePrefix() == req.Prefix {
			entry, err = store.FindRawEntry(ctx, req.Key, req.Block)
			if err != nil {
				return nil, err
			}
			entry.Index = req.Key // desanitize index so it will be printable
			break
		}
	}

	if req.HideData {
		entry.Data = nil
	}

	var stringData string
	if req.StringData {
		stringData = string(entry.Data)
	}

	return &types.QueryEntryResponse{Entry: &entry, StringData: stringData}, nil
}
