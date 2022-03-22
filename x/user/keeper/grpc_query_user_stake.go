package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/lavanet/lava/x/user/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) UserStakeAll(c context.Context, req *types.QueryAllUserStakeRequest) (*types.QueryAllUserStakeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var userStakes []types.UserStake
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	userStakeStore := prefix.NewStore(store, types.KeyPrefix(types.UserStakeKeyPrefix))

	pageRes, err := query.Paginate(userStakeStore, req.Pagination, func(key []byte, value []byte) error {
		var userStake types.UserStake
		if err := k.cdc.Unmarshal(value, &userStake); err != nil {
			return err
		}

		userStakes = append(userStakes, userStake)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllUserStakeResponse{UserStake: userStakes, Pagination: pageRes}, nil
}

func (k Keeper) UserStake(c context.Context, req *types.QueryGetUserStakeRequest) (*types.QueryGetUserStakeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetUserStake(
		ctx,
		req.Index,
	)
	if !found {
		return nil, status.Error(codes.InvalidArgument, "not found")
	}

	return &types.QueryGetUserStakeResponse{UserStake: val}, nil
}
