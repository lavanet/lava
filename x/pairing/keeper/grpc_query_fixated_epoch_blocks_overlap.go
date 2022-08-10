package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/lavanet/lava/x/pairing/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) FixatedEpochBlocksOverlapAll(c context.Context, req *types.QueryAllFixatedEpochBlocksOverlapRequest) (*types.QueryAllFixatedEpochBlocksOverlapResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var fixatedEpochBlocksOverlaps []types.FixatedEpochBlocksOverlap
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	fixatedEpochBlocksOverlapStore := prefix.NewStore(store, types.KeyPrefix(types.FixatedEpochBlocksOverlapKeyPrefix))

	pageRes, err := query.Paginate(fixatedEpochBlocksOverlapStore, req.Pagination, func(key []byte, value []byte) error {
		var fixatedEpochBlocksOverlap types.FixatedEpochBlocksOverlap
		if err := k.cdc.Unmarshal(value, &fixatedEpochBlocksOverlap); err != nil {
			return err
		}

		fixatedEpochBlocksOverlaps = append(fixatedEpochBlocksOverlaps, fixatedEpochBlocksOverlap)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllFixatedEpochBlocksOverlapResponse{FixatedEpochBlocksOverlap: fixatedEpochBlocksOverlaps, Pagination: pageRes}, nil
}

func (k Keeper) FixatedEpochBlocksOverlap(c context.Context, req *types.QueryGetFixatedEpochBlocksOverlapRequest) (*types.QueryGetFixatedEpochBlocksOverlapResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetFixatedEpochBlocksOverlap(
		ctx,
		req.Index,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetFixatedEpochBlocksOverlapResponse{FixatedEpochBlocksOverlap: val}, nil
}
