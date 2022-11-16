package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/lavanet/lava/x/spec/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) SpecAll(c context.Context, req *types.QueryAllSpecRequest) (*types.QueryAllSpecResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var Specs []types.Spec
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	SpecStore := prefix.NewStore(store, types.KeyPrefix(types.SpecKeyPrefix))

	pageRes, err := query.Paginate(SpecStore, req.Pagination, func(key []byte, value []byte) error {
		var Spec types.Spec
		if err := k.cdc.Unmarshal(value, &Spec); err != nil {
			return err
		}

		Specs = append(Specs, Spec)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllSpecResponse{Spec: Specs, Pagination: pageRes}, nil
}

func (k Keeper) Spec(c context.Context, req *types.QueryGetSpecRequest) (*types.QueryGetSpecResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetSpec(
		ctx,
		req.ChainID,
	)
	if !found {
		return nil, status.Error(codes.InvalidArgument, "not found")
	}

	return &types.QueryGetSpecResponse{Spec: val}, nil
}
