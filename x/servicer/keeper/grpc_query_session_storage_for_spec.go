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

func (k Keeper) SessionStorageForSpecAll(c context.Context, req *types.QueryAllSessionStorageForSpecRequest) (*types.QueryAllSessionStorageForSpecResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var sessionStorageForSpecs []types.SessionStorageForSpec
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	sessionStorageForSpecStore := prefix.NewStore(store, types.KeyPrefix(types.SessionStorageForSpecKeyPrefix))

	pageRes, err := query.Paginate(sessionStorageForSpecStore, req.Pagination, func(key []byte, value []byte) error {
		var sessionStorageForSpec types.SessionStorageForSpec
		if err := k.cdc.Unmarshal(value, &sessionStorageForSpec); err != nil {
			return err
		}

		sessionStorageForSpecs = append(sessionStorageForSpecs, sessionStorageForSpec)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllSessionStorageForSpecResponse{SessionStorageForSpec: sessionStorageForSpecs, Pagination: pageRes}, nil
}

func (k Keeper) SessionStorageForSpec(c context.Context, req *types.QueryGetSessionStorageForSpecRequest) (*types.QueryGetSessionStorageForSpecResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetSessionStorageForSpec(
		ctx,
		req.Index,
	)
	if !found {
		return nil, status.Error(codes.InvalidArgument, "not found")
	}

	return &types.QueryGetSessionStorageForSpecResponse{SessionStorageForSpec: val}, nil
}
