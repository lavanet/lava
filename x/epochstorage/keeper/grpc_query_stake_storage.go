package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/epochstorage/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) StakeStorageAll(c context.Context, req *types.QueryAllStakeStorageRequest) (*types.QueryAllStakeStorageResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	stakeStorages := k.GetAllStakeEntriesForGenesis(ctx)

	return &types.QueryAllStakeStorageResponse{StakeStorage: stakeStorages}, nil
}

func (k Keeper) StakeStorage(c context.Context, req *types.QueryGetStakeStorageRequest) (*types.QueryGetStakeStorageResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	stakeEntries := k.GetAllStakeEntriesCurrentForChainId(ctx, req.Index)
	if len(stakeEntries) == 0 {
		return nil, status.Error(codes.InvalidArgument, "not found")
	}
	val := types.StakeStorage{
		Index:        req.Index,
		StakeEntries: stakeEntries,
	}

	return &types.QueryGetStakeStorageResponse{StakeStorage: val}, nil
}
