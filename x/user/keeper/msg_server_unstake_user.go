package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/user/types"
)

func (k msgServer) UnstakeUser(goCtx context.Context, msg *types.MsgUnstakeUser) (*types.MsgUnstakeUserResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgUnstakeUserResponse{}, nil
}
