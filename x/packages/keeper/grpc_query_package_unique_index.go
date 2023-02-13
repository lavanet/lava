package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/lavanet/lava/x/packages/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) PackageUniqueIndexAll(c context.Context, req *types.QueryAllPackageUniqueIndexRequest) (*types.QueryAllPackageUniqueIndexResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var packageUniqueIndexs []types.PackageUniqueIndex
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	packageUniqueIndexStore := prefix.NewStore(store, types.KeyPrefix(types.PackageUniqueIndexKey))

	pageRes, err := query.Paginate(packageUniqueIndexStore, req.Pagination, func(key []byte, value []byte) error {
		var packageUniqueIndex types.PackageUniqueIndex
		if err := k.cdc.Unmarshal(value, &packageUniqueIndex); err != nil {
			return err
		}

		packageUniqueIndexs = append(packageUniqueIndexs, packageUniqueIndex)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllPackageUniqueIndexResponse{PackageUniqueIndex: packageUniqueIndexs, Pagination: pageRes}, nil
}

func (k Keeper) PackageUniqueIndex(c context.Context, req *types.QueryGetPackageUniqueIndexRequest) (*types.QueryGetPackageUniqueIndexResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	packageUniqueIndex, found := k.GetPackageUniqueIndex(ctx, req.Id)
	if !found {
		return nil, sdkerrors.ErrKeyNotFound
	}

	return &types.QueryGetPackageUniqueIndexResponse{PackageUniqueIndex: packageUniqueIndex}, nil
}
