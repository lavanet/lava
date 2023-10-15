package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/conflict/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ProviderConflicts(c context.Context, req *types.QueryProviderConflictsRequest) (*types.QueryProviderConflictsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var (
		reported []string
		notVoted []string
	)

	ctx := sdk.UnwrapSDKContext(c)
	conflicts := k.GetAllConflictVote(ctx)

	for _, conflict := range conflicts {
		if conflict.FirstProvider.Account == req.Provider ||
			conflict.SecondProvider.Account == req.Provider {
			reported = append(reported, conflict.Index)
		}

		for _, vote := range conflict.Votes {
			if vote.Address == req.Provider && vote.Result == types.NoVote {
				notVoted = append(notVoted, conflict.Index)
			}
		}
	}

	return &types.QueryProviderConflictsResponse{Reported: reported, NotVoted: notVoted}, nil
}
