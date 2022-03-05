package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/lavanet/lava/x/spec/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) SpecAll(c context.Context, req *types.QueryAllSpecRequest) (*types.QueryAllSpecResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var specs []types.Spec
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	specStore := prefix.NewStore(store, types.KeyPrefix(types.SpecKey))

	pageRes, err := query.Paginate(specStore, req.Pagination, func(key []byte, value []byte) error {
		var spec types.Spec
		if err := k.cdc.Unmarshal(value, &spec); err != nil {
			return err
		}

		specs = append(specs, spec)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllSpecResponse{Spec: specs, Pagination: pageRes}, nil
}

func (k Keeper) Spec(c context.Context, req *types.QueryGetSpecRequest) (*types.QueryGetSpecResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	spec, found := k.GetSpec(ctx, req.Id)
	if !found {
		return nil, sdkerrors.ErrKeyNotFound
	}

	return &types.QueryGetSpecResponse{Spec: spec}, nil
}
