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

func (k Keeper) UserPaymentStorageAll(c context.Context, req *types.QueryAllUserPaymentStorageRequest) (*types.QueryAllUserPaymentStorageResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var userPaymentStorages []types.UserPaymentStorage
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	userPaymentStorageStore := prefix.NewStore(store, types.KeyPrefix(types.UserPaymentStorageKeyPrefix))

	pageRes, err := query.Paginate(userPaymentStorageStore, req.Pagination, func(key []byte, value []byte) error {
		var userPaymentStorage types.UserPaymentStorage
		if err := k.cdc.Unmarshal(value, &userPaymentStorage); err != nil {
			return err
		}

		userPaymentStorages = append(userPaymentStorages, userPaymentStorage)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllUserPaymentStorageResponse{UserPaymentStorage: userPaymentStorages, Pagination: pageRes}, nil
}

func (k Keeper) UserPaymentStorage(c context.Context, req *types.QueryGetUserPaymentStorageRequest) (*types.QueryGetUserPaymentStorageResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetUserPaymentStorage(
		ctx,
		req.Index,
	)
	if !found {
		return nil, status.Error(codes.InvalidArgument, "not found")
	}

	return &types.QueryGetUserPaymentStorageResponse{UserPaymentStorage: val}, nil
}
