package keeper

import (
	"bytes"
	"context"
	"encoding/binary"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/conflict/types"
	tendermintcrypto "github.com/tendermint/tendermint/crypto"
)

func (k msgServer) ConflictVoteReveal(goCtx context.Context, msg *types.MsgConflictVoteReveal) (*types.MsgConflictVoteRevealResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Keeper.Logger(ctx)

	_ = ctx
	conflictVote, found := k.GetConflictVote(ctx, strconv.FormatUint(msg.VoteID, 10))
	if !found {
		return nil, utils.LavaError(ctx, logger, "response_conflict_detection_reveal", map[string]string{"provider": msg.Creator, "voteID": strconv.FormatUint(msg.VoteID, 10)}, "invalid vote id")
	}
	if conflictVote.VoteIsCommit {
		return nil, utils.LavaError(ctx, logger, "response_conflict_detection_reveal", map[string]string{"provider": msg.Creator, "voteID": strconv.FormatUint(msg.VoteID, 10)}, "vote is not in reveal state")
	}
	if _, ok := conflictVote.VotersHash[msg.Creator]; !ok {
		return nil, utils.LavaError(ctx, logger, "response_conflict_detection_reveal", map[string]string{"provider": msg.Creator, "voteID": strconv.FormatUint(msg.VoteID, 10)}, "provider is not in the voters list")
	}
	if conflictVote.VotersHash[msg.Creator].Hash == nil {
		return nil, utils.LavaError(ctx, logger, "response_conflict_detection_reveal", map[string]string{"provider": msg.Creator, "voteID": strconv.FormatUint(msg.VoteID, 10)}, "provider did not commit")
	}

	if conflictVote.VotersHash[msg.Creator].Result != NoVote {
		return nil, utils.LavaError(ctx, logger, "response_conflict_detection_reveal", map[string]string{"provider": msg.Creator, "voteID": strconv.FormatUint(msg.VoteID, 10)}, "provider already revealed")
	}

	dataHash := make([]byte, 8)
	binary.LittleEndian.PutUint64(dataHash, uint64(msg.Nonce))
	dataHash = append(dataHash, msg.Hash...) //todo is this ok?
	hash := tendermintcrypto.Sha256(dataHash)

	if bytes.Equal(hash, conflictVote.VotersHash[msg.Creator].Hash) {
		return nil, utils.LavaError(ctx, logger, "response_conflict_detection_reveal", map[string]string{"provider": msg.Creator, "voteID": strconv.FormatUint(msg.VoteID, 10)}, "provider reveal does not match the commit")
	}

	vote := conflictVote.VotersHash[msg.Creator]

	if bytes.Equal(msg.Hash, conflictVote.FirstProvider.Response) {
		vote.Result = Provider0
	} else if bytes.Equal(msg.Hash, conflictVote.SecondProvider.Response) {
		vote.Result = Provider1
	} else {
		vote.Result = None
	}
	conflictVote.VotersHash[msg.Creator] = vote

	k.SetConflictVote(ctx, conflictVote)
	return &types.MsgConflictVoteRevealResponse{}, nil
}
