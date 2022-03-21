package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/lavanet/lava/x/user/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) UnstakingUsersAllSpecsAll(c context.Context, req *types.QueryAllUnstakingUsersAllSpecsRequest) (*types.QueryAllUnstakingUsersAllSpecsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var unstakingUsersAllSpecss []types.UnstakingUsersAllSpecs
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	unstakingUsersAllSpecsStore := prefix.NewStore(store, types.KeyPrefix(types.UnstakingUsersAllSpecsKey))

	pageRes, err := query.Paginate(unstakingUsersAllSpecsStore, req.Pagination, func(key []byte, value []byte) error {
		var unstakingUsersAllSpecs types.UnstakingUsersAllSpecs
		if err := k.cdc.Unmarshal(value, &unstakingUsersAllSpecs); err != nil {
			return err
		}

		unstakingUsersAllSpecss = append(unstakingUsersAllSpecss, unstakingUsersAllSpecs)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllUnstakingUsersAllSpecsResponse{UnstakingUsersAllSpecs: unstakingUsersAllSpecss, Pagination: pageRes}, nil
}

func (k Keeper) UnstakingUsersAllSpecs(c context.Context, req *types.QueryGetUnstakingUsersAllSpecsRequest) (*types.QueryGetUnstakingUsersAllSpecsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	unstakingUsersAllSpecs, found := k.GetUnstakingUsersAllSpecs(ctx, req.Id)
	if !found {
		return nil, sdkerrors.ErrKeyNotFound
	}

	return &types.QueryGetUnstakingUsersAllSpecsResponse{UnstakingUsersAllSpecs: unstakingUsersAllSpecs}, nil
}
