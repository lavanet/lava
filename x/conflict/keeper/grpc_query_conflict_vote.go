package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/lavanet/lava/v2/x/conflict/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ConflictVoteAll(c context.Context, req *types.QueryAllConflictVoteRequest) (*types.QueryAllConflictVoteResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var conflictVotes []types.ConflictVote
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	conflictVoteStore := prefix.NewStore(store, types.KeyPrefix(types.ConflictVoteKeyPrefix))

	pageRes, err := query.Paginate(conflictVoteStore, req.Pagination, func(key, value []byte) error {
		var conflictVote types.ConflictVote
		if err := k.cdc.Unmarshal(value, &conflictVote); err != nil {
			return err
		}

		conflictVotes = append(conflictVotes, conflictVote)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllConflictVoteResponse{ConflictVote: conflictVotes, Pagination: pageRes}, nil
}

func (k Keeper) ConflictVote(c context.Context, req *types.QueryGetConflictVoteRequest) (*types.QueryGetConflictVoteResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetConflictVote(
		ctx,
		req.Index,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetConflictVoteResponse{ConflictVote: val}, nil
}
