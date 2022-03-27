package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/lavanet/lava/x/servicer/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) UniquePaymentStorageUserServicerAll(c context.Context, req *types.QueryAllUniquePaymentStorageUserServicerRequest) (*types.QueryAllUniquePaymentStorageUserServicerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var uniquePaymentStorageUserServicers []types.UniquePaymentStorageUserServicer
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	uniquePaymentStorageUserServicerStore := prefix.NewStore(store, types.KeyPrefix(types.UniquePaymentStorageUserServicerKeyPrefix))

	pageRes, err := query.Paginate(uniquePaymentStorageUserServicerStore, req.Pagination, func(key []byte, value []byte) error {
		var uniquePaymentStorageUserServicer types.UniquePaymentStorageUserServicer
		if err := k.cdc.Unmarshal(value, &uniquePaymentStorageUserServicer); err != nil {
			return err
		}

		uniquePaymentStorageUserServicers = append(uniquePaymentStorageUserServicers, uniquePaymentStorageUserServicer)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllUniquePaymentStorageUserServicerResponse{UniquePaymentStorageUserServicer: uniquePaymentStorageUserServicers, Pagination: pageRes}, nil
}

func (k Keeper) UniquePaymentStorageUserServicer(c context.Context, req *types.QueryGetUniquePaymentStorageUserServicerRequest) (*types.QueryGetUniquePaymentStorageUserServicerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetUniquePaymentStorageUserServicer(
		ctx,
		req.Index,
	)
	if !found {
		return nil, status.Error(codes.InvalidArgument, "not found")
	}

	return &types.QueryGetUniquePaymentStorageUserServicerResponse{UniquePaymentStorageUserServicer: val}, nil
}
