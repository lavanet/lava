package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/conflict/types"
)

func (k msgServer) ConflictVoteCommit(goCtx context.Context, msg *types.MsgConflictVoteCommit) (*types.MsgConflictVoteCommitResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Keeper.Logger(ctx)

	conflictVote, found := k.GetConflictVote(ctx, msg.VoteID)
	if !found {
		return nil, utils.LavaFormatWarning("invalid vote id", sdkerrors.ErrKeyNotFound,
			utils.Attribute{Key: "provider", Value: msg.Creator},
			utils.Attribute{Key: "voteID", Value: msg.VoteID},
		)
	}
	if conflictVote.VoteState != types.StateCommit {
		return nil, utils.LavaFormatWarning("vote is not in commit state", sdkerrors.ErrInvalidRequest,
			utils.Attribute{Key: "provider", Value: msg.Creator},
			utils.Attribute{Key: "voteID", Value: msg.VoteID},
		)
	}
	index, ok := FindVote(&conflictVote.Votes, msg.Creator)
	if !ok {
		return nil, utils.LavaFormatWarning("provider is not in the voters list", sdkerrors.ErrKeyNotFound,
			utils.Attribute{Key: "provider", Value: msg.Creator},
			utils.Attribute{Key: "voteID", Value: msg.VoteID},
		)
	}
	if conflictVote.Votes[index].Result != types.NoVote {
		return nil, utils.LavaFormatWarning("provider already committed", sdkerrors.ErrInvalidRequest,
			utils.Attribute{Key: "provider", Value: msg.Creator},
			utils.Attribute{Key: "voteID", Value: msg.VoteID},
		)
	}

	conflictVote.Votes[index].Hash = msg.Hash
	conflictVote.Votes[index].Result = types.Commit
	k.SetConflictVote(ctx, conflictVote)

	utils.LogLavaEvent(ctx, logger, types.ConflictVoteGotCommitEventName, map[string]string{"voteID": msg.VoteID, "provider": msg.Creator}, "conflict commit received")
	return &types.MsgConflictVoteCommitResponse{}, nil
}
