package keeper

import (
	"bytes"
	"context"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/conflict/types"
)

func (k msgServer) ConflictVoteReveal(goCtx context.Context, msg *types.MsgConflictVoteReveal) (*types.MsgConflictVoteRevealResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Keeper.Logger(ctx)

	conflictVote, found := k.GetConflictVote(ctx, strconv.FormatUint(msg.VoteID, 10))
	if !found {
		return nil, utils.LavaError(ctx, logger, "response_conflict_detection_reveal", map[string]string{"provider": msg.Creator, "voteID": strconv.FormatUint(msg.VoteID, 10)}, "invalid vote id")
	}
	if conflictVote.VoteState != types.StateReveal {
		return nil, utils.LavaError(ctx, logger, "response_conflict_detection_reveal", map[string]string{"provider": msg.Creator, "voteID": strconv.FormatUint(msg.VoteID, 10)}, "vote is not in reveal state")
	}
	if _, ok := conflictVote.VotersHash[msg.Creator]; !ok {
		return nil, utils.LavaError(ctx, logger, "response_conflict_detection_reveal", map[string]string{"provider": msg.Creator, "voteID": strconv.FormatUint(msg.VoteID, 10)}, "provider is not in the voters list")
	}
	if conflictVote.VotersHash[msg.Creator].Hash == nil {
		return nil, utils.LavaError(ctx, logger, "response_conflict_detection_reveal", map[string]string{"provider": msg.Creator, "voteID": strconv.FormatUint(msg.VoteID, 10)}, "provider did not commit")
	}
	if conflictVote.VotersHash[msg.Creator].Result != types.Commit {
		return nil, utils.LavaError(ctx, logger, "response_conflict_detection_reveal", map[string]string{"provider": msg.Creator, "voteID": strconv.FormatUint(msg.VoteID, 10)}, "provider already revealed")
	}

	commitHash := types.CommitVoteData(msg.Nonce, msg.Hash)
	if !bytes.Equal(commitHash, conflictVote.VotersHash[msg.Creator].Hash) {
		return nil, utils.LavaError(ctx, logger, "response_conflict_detection_reveal", map[string]string{"provider": msg.Creator, "voteID": strconv.FormatUint(msg.VoteID, 10)}, "provider reveal does not match the commit")
	}

	vote := conflictVote.VotersHash[msg.Creator]
	if bytes.Equal(msg.Hash, conflictVote.FirstProvider.Response) {
		vote.Result = types.Provider0
	} else if bytes.Equal(msg.Hash, conflictVote.SecondProvider.Response) {
		vote.Result = types.Provider1
	} else {
		vote.Result = types.None
	}
	conflictVote.VotersHash[msg.Creator] = vote

	k.SetConflictVote(ctx, conflictVote)
	utils.LogLavaEvent(ctx, logger, types.ConflictVoteGotCommitEventName, map[string]string{"voteID": strconv.FormatUint(msg.VoteID, 10), "provider": msg.Creator}, "conflict reveal recieved")
	return &types.MsgConflictVoteRevealResponse{}, nil
}
