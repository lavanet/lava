package keeper

import (
	"bytes"
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/conflict/types"
)

func (k msgServer) ConflictVoteReveal(goCtx context.Context, msg *types.MsgConflictVoteReveal) (*types.MsgConflictVoteRevealResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Keeper.Logger(ctx)

	conflictVote, found := k.GetConflictVote(ctx, msg.VoteID)
	if !found {
		return nil, utils.LavaError(ctx, logger, "response_conflict_detection_reveal", map[string]string{"provider": msg.Creator, "voteID": msg.VoteID}, "Simulation: invalid vote id")
	}
	if conflictVote.VoteState != types.StateReveal {
		return nil, utils.LavaError(ctx, logger, "response_conflict_detection_reveal", map[string]string{"provider": msg.Creator, "voteID": msg.VoteID}, "Simulation: vote is not in reveal state")
	}
	index, ok := FindVote(&conflictVote.Votes, msg.Creator)
	if !ok {
		return nil, utils.LavaError(ctx, logger, "response_conflict_detection_reveal", map[string]string{"provider": msg.Creator, "voteID": msg.VoteID}, "Simulation: provider is not in the voters list")
	}
	if conflictVote.Votes[index].Hash == nil {
		return nil, utils.LavaError(ctx, logger, "response_conflict_detection_reveal", map[string]string{"provider": msg.Creator, "voteID": msg.VoteID}, "Simulation: provider did not commit")
	}
	if conflictVote.Votes[index].Result != types.Commit {
		return nil, utils.LavaError(ctx, logger, "response_conflict_detection_reveal", map[string]string{"provider": msg.Creator, "voteID": msg.VoteID}, "Simulation: provider already revealed")
	}

	commitHash := types.CommitVoteData(msg.Nonce, msg.Hash)
	if !bytes.Equal(commitHash, conflictVote.Votes[index].Hash) {
		return nil, utils.LavaError(ctx, logger, "response_conflict_detection_reveal", map[string]string{"provider": msg.Creator, "voteID": msg.VoteID}, "Simulation: provider reveal does not match the commit")
	}

	if bytes.Equal(msg.Hash, conflictVote.FirstProvider.Response) {
		conflictVote.Votes[index].Result = types.Provider0
	} else if bytes.Equal(msg.Hash, conflictVote.SecondProvider.Response) {
		conflictVote.Votes[index].Result = types.Provider1
	} else {
		conflictVote.Votes[index].Result = types.NoneOfTheProviders
	}

	k.SetConflictVote(ctx, conflictVote)
	utils.LogLavaEvent(ctx, logger, types.ConflictVoteGotRevealEventName, map[string]string{"voteID": msg.VoteID, "provider": msg.Creator}, "Simulation: conflict reveal received")
	return &types.MsgConflictVoteRevealResponse{}, nil
}
