package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/lavanet/lava/x/packagemanager/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) PackageVersionsStorageAll(c context.Context, req *types.QueryAllPackageVersionsStorageRequest) (*types.QueryAllPackageVersionsStorageResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var packageVersionsStorages []types.PackageVersionsStorage
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	packageVersionsStorageStore := prefix.NewStore(store, types.KeyPrefix(types.PackageVersionsStorageKeyPrefix))

	pageRes, err := query.Paginate(packageVersionsStorageStore, req.Pagination, func(key []byte, value []byte) error {
		var packageVersionsStorage types.PackageVersionsStorage
		if err := k.cdc.Unmarshal(value, &packageVersionsStorage); err != nil {
			return err
		}

		packageVersionsStorages = append(packageVersionsStorages, packageVersionsStorage)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllPackageVersionsStorageResponse{PackageVersionsStorage: packageVersionsStorages, Pagination: pageRes}, nil
}

func (k Keeper) PackageVersionsStorage(c context.Context, req *types.QueryGetPackageVersionsStorageRequest) (*types.QueryGetPackageVersionsStorageResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetPackageVersionsStorage(
		ctx,
		req.PackageIndex,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetPackageVersionsStorageResponse{PackageVersionsStorage: val}, nil
}
