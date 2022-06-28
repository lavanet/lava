package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/conflict/types"
)

func (k msgServer) ConflictVoteCommit(goCtx context.Context, msg *types.MsgConflictVoteCommit) (*types.MsgConflictVoteCommitResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgConflictVoteCommitResponse{}, nil
}
