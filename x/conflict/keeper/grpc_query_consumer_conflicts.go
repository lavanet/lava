package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/conflict/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ConsumerConflicts(c context.Context, req *types.QueryConsumerConflictsRequest) (*types.QueryConsumerConflictsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var conflictIndices []string

	ctx := sdk.UnwrapSDKContext(c)
	conflicts := k.GetAllConflictVote(ctx)
	for _, conflict := range conflicts {
		if conflict.ClientAddress == req.Consumer {
			conflictIndices = append(conflictIndices, conflict.Index)
		}
	}

	return &types.QueryConsumerConflictsResponse{Conflicts: conflictIndices}, nil
}
