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

func (k Keeper) SpecStakeStorageAll(c context.Context, req *types.QueryAllSpecStakeStorageRequest) (*types.QueryAllSpecStakeStorageResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var specStakeStorages []types.SpecStakeStorage
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	specStakeStorageStore := prefix.NewStore(store, types.KeyPrefix(types.SpecStakeStorageKeyPrefix))

	pageRes, err := query.Paginate(specStakeStorageStore, req.Pagination, func(key []byte, value []byte) error {
		var specStakeStorage types.SpecStakeStorage
		if err := k.cdc.Unmarshal(value, &specStakeStorage); err != nil {
			return err
		}

		specStakeStorages = append(specStakeStorages, specStakeStorage)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllSpecStakeStorageResponse{SpecStakeStorage: specStakeStorages, Pagination: pageRes}, nil
}

func (k Keeper) SpecStakeStorage(c context.Context, req *types.QueryGetSpecStakeStorageRequest) (*types.QueryGetSpecStakeStorageResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetSpecStakeStorage(
		ctx,
		req.Index,
	)
	if !found {
		return nil, status.Error(codes.InvalidArgument, "not found")
	}

	return &types.QueryGetSpecStakeStorageResponse{SpecStakeStorage: val}, nil
}
