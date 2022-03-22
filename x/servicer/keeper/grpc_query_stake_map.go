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

func (k Keeper) StakeMapAll(c context.Context, req *types.QueryAllStakeMapRequest) (*types.QueryAllStakeMapResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var stakeMaps []types.StakeMap
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	stakeMapStore := prefix.NewStore(store, types.KeyPrefix(types.StakeMapKeyPrefix))

	pageRes, err := query.Paginate(stakeMapStore, req.Pagination, func(key []byte, value []byte) error {
		var stakeMap types.StakeMap
		if err := k.cdc.Unmarshal(value, &stakeMap); err != nil {
			return err
		}

		stakeMaps = append(stakeMaps, stakeMap)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllStakeMapResponse{StakeMap: stakeMaps, Pagination: pageRes}, nil
}

func (k Keeper) StakeMap(c context.Context, req *types.QueryGetStakeMapRequest) (*types.QueryGetStakeMapResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetStakeMap(
		ctx,
		req.Index,
	)
	if !found {
		return nil, status.Error(codes.InvalidArgument, "not found")
	}

	return &types.QueryGetStakeMapResponse{StakeMap: val}, nil
}
