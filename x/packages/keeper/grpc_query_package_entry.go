package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/lavanet/lava/x/packages/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) PackageEntryAll(c context.Context, req *types.QueryAllPackageEntryRequest) (*types.QueryAllPackageEntryResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var packageEntrys []types.PackageEntry
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	packageEntryStore := prefix.NewStore(store, types.KeyPrefix(types.PackageEntryKeyPrefix))

	pageRes, err := query.Paginate(packageEntryStore, req.Pagination, func(key []byte, value []byte) error {
		var packageEntry types.PackageEntry
		if err := k.cdc.Unmarshal(value, &packageEntry); err != nil {
			return err
		}

		packageEntrys = append(packageEntrys, packageEntry)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllPackageEntryResponse{PackageEntry: packageEntrys, Pagination: pageRes}, nil
}

func (k Keeper) PackageEntry(c context.Context, req *types.QueryGetPackageEntryRequest) (*types.QueryGetPackageEntryResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetPackageEntry(
		ctx,
		req.PackageIndex,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetPackageEntryResponse{PackageEntry: val}, nil
}
