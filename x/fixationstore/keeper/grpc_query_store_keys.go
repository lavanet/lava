package keeper

import (
	"context"

	"github.com/lavanet/lava/v2/x/fixationstore/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k *Keeper) StoreKeys(goCtx context.Context, req *types.QueryStoreKeysRequest) (*types.QueryStoreKeysResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var keys []types.StoreKeyAndPrefix
	for _, store := range k.fixationsStores {
		keys = append(keys, types.StoreKeyAndPrefix{StoreKey: store.GetStoreKey().Name(), Prefix: store.GetStorePrefix()})
	}

	return &types.QueryStoreKeysResponse{Keys: keys}, nil
}
