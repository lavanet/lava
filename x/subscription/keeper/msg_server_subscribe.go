package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/subscription/types"
)

func (k msgServer) Subscribe(goCtx context.Context, msg *types.MsgSubscribe) (*types.MsgSubscribeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgSubscribeResponse{}, nil
}
