package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/pairing/types"
)

func (k msgServer) UnstakeClient(goCtx context.Context, msg *types.MsgUnstakeClient) (*types.MsgUnstakeClientResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgUnstakeClientResponse{}, nil
}
