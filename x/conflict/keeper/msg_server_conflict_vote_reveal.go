package keeper

import (
	"bytes"
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/utils/sigs"
	"github.com/lavanet/lava/v2/x/conflict/types"
)

func (k msgServer) ConflictVoteReveal(goCtx context.Context, msg *types.MsgConflictVoteReveal) (*types.MsgConflictVoteRevealResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Keeper.Logger(ctx)

	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return nil, utils.LavaFormatWarning("invalid client address", err,
			utils.Attribute{Key: "client", Value: msg.Creator},
		)
	}

	conflictVote, found := k.GetConflictVote(ctx, msg.VoteID)
	if !found {
		return nil, utils.LavaFormatWarning("Simulation: invalid vote id", legacyerrors.ErrKeyNotFound,
			utils.Attribute{Key: "provider", Value: msg.Creator},
			utils.Attribute{Key: "voteID", Value: msg.VoteID},
		)
	}
	if conflictVote.VoteState != types.StateReveal {
		return nil, utils.LavaFormatWarning("Simulation: vote is not in reveal state", legacyerrors.ErrInvalidRequest,
			utils.Attribute{Key: "provider", Value: msg.Creator},
			utils.Attribute{Key: "voteID", Value: msg.VoteID},
		)
	}
	index, ok := FindVote(&conflictVote.Votes, msg.Creator)
	if !ok {
		return nil, utils.LavaFormatWarning("Simulation: provider is not in the voters list", legacyerrors.ErrKeyNotFound,
			utils.Attribute{Key: "provider", Value: msg.Creator},
			utils.Attribute{Key: "voteID", Value: msg.VoteID},
		)
	}
	if conflictVote.Votes[index].Hash == nil {
		return nil, utils.LavaFormatWarning("Simulation: provider did not commit", legacyerrors.ErrInvalidRequest,
			utils.Attribute{Key: "provider", Value: msg.Creator},
			utils.Attribute{Key: "voteID", Value: msg.VoteID},
		)
	}
	if conflictVote.Votes[index].Result != types.Commit {
		return nil, utils.LavaFormatWarning("Simulation: provider already revealed", legacyerrors.ErrInvalidRequest,
			utils.Attribute{Key: "provider", Value: msg.Creator},
			utils.Attribute{Key: "voteID", Value: msg.VoteID},
		)
	}

	commitHash := types.CommitVoteData(msg.Nonce, msg.Hash, msg.Creator)
	if !bytes.Equal(commitHash, conflictVote.Votes[index].Hash) {
		return nil, utils.LavaFormatWarning("Simulation: provider reveal does not match the commit", legacyerrors.ErrInvalidRequest,
			utils.Attribute{Key: "provider", Value: msg.Creator},
			utils.Attribute{Key: "voteID", Value: msg.VoteID},
		)
	}
	hashAllDataHash := sigs.HashMsg(msg.Hash) // since the conflict detection sends hashmsg(sigs.AllDataHash()), for equality we need to hash
	if bytes.Equal(hashAllDataHash, conflictVote.FirstProvider.Response) {
		conflictVote.Votes[index].Result = types.Provider0
	} else if bytes.Equal(hashAllDataHash, conflictVote.SecondProvider.Response) {
		conflictVote.Votes[index].Result = types.Provider1
	} else {
		conflictVote.Votes[index].Result = types.NoneOfTheProviders
	}

	k.SetConflictVote(ctx, conflictVote)
	utils.LogLavaEvent(ctx, logger, types.ConflictVoteGotRevealEventName, map[string]string{"voteID": msg.VoteID, "provider": msg.Creator}, "Simulation: conflict reveal received")
	return &types.MsgConflictVoteRevealResponse{}, nil
}
