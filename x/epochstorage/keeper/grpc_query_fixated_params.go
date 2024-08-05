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

func (k Keeper) FixatedParamsAll(c context.Context, req *types.QueryAllFixatedParamsRequest) (*types.QueryAllFixatedParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var fixatedParamss []types.FixatedParams
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	fixatedParamsStore := prefix.NewStore(store, types.KeyPrefix(types.FixatedParamsKeyPrefix))

	pageRes, err := query.Paginate(fixatedParamsStore, req.Pagination, func(key, value []byte) error {
		var fixatedParams types.FixatedParams
		if err := k.cdc.Unmarshal(value, &fixatedParams); err != nil {
			return err
		}

		fixatedParamss = append(fixatedParamss, fixatedParams)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllFixatedParamsResponse{FixatedParams: fixatedParamss, Pagination: pageRes}, nil
}

func (k Keeper) FixatedParams(c context.Context, req *types.QueryGetFixatedParamsRequest) (*types.QueryGetFixatedParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetFixatedParams(
		ctx,
		req.Index,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetFixatedParamsResponse{FixatedParams: val}, nil
}
