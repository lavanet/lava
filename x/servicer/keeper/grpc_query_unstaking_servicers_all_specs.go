package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/lavanet/lava/x/servicer/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) UnstakingServicersAllSpecsAll(c context.Context, req *types.QueryAllUnstakingServicersAllSpecsRequest) (*types.QueryAllUnstakingServicersAllSpecsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var unstakingServicersAllSpecss []types.UnstakingServicersAllSpecs
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	unstakingServicersAllSpecsStore := prefix.NewStore(store, types.KeyPrefix(types.UnstakingServicersAllSpecsKey))

	pageRes, err := query.Paginate(unstakingServicersAllSpecsStore, req.Pagination, func(key []byte, value []byte) error {
		var unstakingServicersAllSpecs types.UnstakingServicersAllSpecs
		if err := k.cdc.Unmarshal(value, &unstakingServicersAllSpecs); err != nil {
			return err
		}

		unstakingServicersAllSpecss = append(unstakingServicersAllSpecss, unstakingServicersAllSpecs)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllUnstakingServicersAllSpecsResponse{UnstakingServicersAllSpecs: unstakingServicersAllSpecss, Pagination: pageRes}, nil
}

func (k Keeper) UnstakingServicersAllSpecs(c context.Context, req *types.QueryGetUnstakingServicersAllSpecsRequest) (*types.QueryGetUnstakingServicersAllSpecsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	unstakingServicersAllSpecs, found := k.GetUnstakingServicersAllSpecs(ctx, req.Id)
	if !found {
		return nil, sdkerrors.ErrKeyNotFound
	}

	return &types.QueryGetUnstakingServicersAllSpecsResponse{UnstakingServicersAllSpecs: unstakingServicersAllSpecs}, nil
}
