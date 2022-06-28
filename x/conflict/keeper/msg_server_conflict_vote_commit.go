package keeper

import (
	"context"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/conflict/types"
)

func (k msgServer) ConflictVoteCommit(goCtx context.Context, msg *types.MsgConflictVoteCommit) (*types.MsgConflictVoteCommitResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Keeper.Logger(ctx)

	_ = ctx
	conflictVote, found := k.GetConflictVote(ctx, strconv.FormatUint(msg.VoteID, 10))
	if !found {
		return nil, utils.LavaError(ctx, logger, "response_conflict_detection_commit", map[string]string{"provider": msg.Creator, "voteID": strconv.FormatUint(msg.VoteID, 10)}, "invalid vote id")
	}
	if !conflictVote.VoteIsCommit {
		return nil, utils.LavaError(ctx, logger, "response_conflict_detection_commit", map[string]string{"provider": msg.Creator, "voteID": strconv.FormatUint(msg.VoteID, 10)}, "vote is not in commit state")
	}
	if _, ok := conflictVote.VotersHash[msg.Creator]; !ok {
		return nil, utils.LavaError(ctx, logger, "response_conflict_detection_commit", map[string]string{"provider": msg.Creator, "voteID": strconv.FormatUint(msg.VoteID, 10)}, "provider is not in the voters list")
	}
	if conflictVote.VotersHash[msg.Creator].Hash != nil {
		return nil, utils.LavaError(ctx, logger, "response_conflict_detection_commit", map[string]string{"provider": msg.Creator, "voteID": strconv.FormatUint(msg.VoteID, 10)}, "provider already commited")
	}

	conflictVote.VotersHash[msg.Creator] = types.Vote{Hash: msg.Hash, Result: NoVote}
	k.SetConflictVote(ctx, conflictVote)

	return &types.MsgConflictVoteCommitResponse{}, nil
}
