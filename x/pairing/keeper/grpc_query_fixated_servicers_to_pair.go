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

func (k Keeper) FixatedServicersToPairAll(c context.Context, req *types.QueryAllFixatedServicersToPairRequest) (*types.QueryAllFixatedServicersToPairResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var fixatedServicersToPairs []types.FixatedServicersToPair
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	fixatedServicersToPairStore := prefix.NewStore(store, types.KeyPrefix(types.FixatedServicersToPairKeyPrefix))

	pageRes, err := query.Paginate(fixatedServicersToPairStore, req.Pagination, func(key []byte, value []byte) error {
		var fixatedServicersToPair types.FixatedServicersToPair
		if err := k.cdc.Unmarshal(value, &fixatedServicersToPair); err != nil {
			return err
		}

		fixatedServicersToPairs = append(fixatedServicersToPairs, fixatedServicersToPair)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllFixatedServicersToPairResponse{FixatedServicersToPair: fixatedServicersToPairs, Pagination: pageRes}, nil
}

func (k Keeper) FixatedServicersToPair(c context.Context, req *types.QueryGetFixatedServicersToPairRequest) (*types.QueryGetFixatedServicersToPairResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetFixatedServicersToPair(
		ctx,
		req.Index,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetFixatedServicersToPairResponse{FixatedServicersToPair: val}, nil
}
