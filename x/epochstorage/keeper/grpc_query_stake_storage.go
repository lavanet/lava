package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/lavanet/lava/v2/x/epochstorage/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) StakeStorageAll(c context.Context, req *types.QueryAllStakeStorageRequest) (*types.QueryAllStakeStorageResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var stakeStorages []types.StakeStorage
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	stakeStorageStore := prefix.NewStore(store, types.KeyPrefix(types.StakeStorageKeyPrefix))

	pageRes, err := query.Paginate(stakeStorageStore, req.Pagination, func(key, value []byte) error {
		var stakeStorage types.StakeStorage
		if err := k.cdc.Unmarshal(value, &stakeStorage); err != nil {
			return err
		}

		stakeStorages = append(stakeStorages, stakeStorage)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllStakeStorageResponse{StakeStorage: stakeStorages, Pagination: pageRes}, nil
}

func (k Keeper) StakeStorage(c context.Context, req *types.QueryGetStakeStorageRequest) (*types.QueryGetStakeStorageResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetStakeStorage(
		ctx,
		req.Index,
	)
	if !found {
		return nil, status.Error(codes.InvalidArgument, "not found")
	}

	return &types.QueryGetStakeStorageResponse{StakeStorage: val}, nil
}
