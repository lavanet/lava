package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/user/types"
)

func (k msgServer) StakeUser(goCtx context.Context, msg *types.MsgStakeUser) (*types.MsgStakeUserResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgStakeUserResponse{}, nil
}
