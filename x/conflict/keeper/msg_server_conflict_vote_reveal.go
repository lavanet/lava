package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/conflict/types"
)

func (k msgServer) ConflictVoteReveal(goCtx context.Context, msg *types.MsgConflictVoteReveal) (*types.MsgConflictVoteRevealResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgConflictVoteRevealResponse{}, nil
}
