package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/lavanet/lava/x/pairing/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ClientPaymentStorageAll(c context.Context, req *types.QueryAllClientPaymentStorageRequest) (*types.QueryAllClientPaymentStorageResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var clientPaymentStorages []types.ClientPaymentStorage
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	clientPaymentStorageStore := prefix.NewStore(store, types.KeyPrefix(types.ClientPaymentStorageKeyPrefix))

	pageRes, err := query.Paginate(clientPaymentStorageStore, req.Pagination, func(key []byte, value []byte) error {
		var clientPaymentStorage types.ClientPaymentStorage
		if err := k.cdc.Unmarshal(value, &clientPaymentStorage); err != nil {
			return err
		}

		clientPaymentStorages = append(clientPaymentStorages, clientPaymentStorage)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllClientPaymentStorageResponse{ClientPaymentStorage: clientPaymentStorages, Pagination: pageRes}, nil
}

func (k Keeper) ClientPaymentStorage(c context.Context, req *types.QueryGetClientPaymentStorageRequest) (*types.QueryGetClientPaymentStorageResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetClientPaymentStorage(
		ctx,
		req.Index,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetClientPaymentStorageResponse{ClientPaymentStorage: val}, nil
}
