package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/lavanet/lava/v2/x/spec/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) doSpecAll(c context.Context, req *types.QueryAllSpecRequest, raw bool) (*types.QueryAllSpecResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var specs []types.Spec
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	specStore := prefix.NewStore(store, types.KeyPrefix(types.SpecKeyPrefix))

	pageRes, err := query.Paginate(specStore, req.Pagination, func(key, value []byte) error {
		var spec types.Spec

		if err := k.cdc.Unmarshal(value, &spec); err != nil {
			return err
		}

		if !raw {
			var err error
			spec, err = k.ExpandSpec(ctx, spec)
			if err != nil { // should not happen! (all specs on chain must be valid)
				return status.Error(codes.Internal, err.Error())
			}
		}

		specs = append(specs, spec)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllSpecResponse{Spec: specs, Pagination: pageRes}, nil
}

func (k Keeper) SpecAll(c context.Context, req *types.QueryAllSpecRequest) (*types.QueryAllSpecResponse, error) {
	return k.doSpecAll(c, req, false)
}

func (k Keeper) SpecAllRaw(c context.Context, req *types.QueryAllSpecRequest) (*types.QueryAllSpecResponse, error) {
	return k.doSpecAll(c, req, true)
}

func (k Keeper) doSpec(c context.Context, req *types.QueryGetSpecRequest, raw bool) (*types.QueryGetSpecResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	spec, found := k.GetSpec(
		ctx,
		req.ChainID,
	)
	if !found {
		return nil, status.Error(codes.InvalidArgument, "not found")
	}

	if !raw {
		var err error
		spec, err = k.ExpandSpec(ctx, spec)
		if err != nil { // should not happen! (all specs on chain must be valid)
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &types.QueryGetSpecResponse{Spec: spec}, nil
}

func (k Keeper) Spec(c context.Context, req *types.QueryGetSpecRequest) (*types.QueryGetSpecResponse, error) {
	return k.doSpec(c, req, false)
}

func (k Keeper) SpecRaw(c context.Context, req *types.QueryGetSpecRequest) (*types.QueryGetSpecResponse, error) {
	return k.doSpec(c, req, true)
}
