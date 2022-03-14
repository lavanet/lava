package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
)

func (k msgServer) UnstakeServicer(goCtx context.Context, msg *types.MsgUnstakeServicer) (*types.MsgUnstakeServicerResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgUnstakeServicerResponse{}, nil
}
