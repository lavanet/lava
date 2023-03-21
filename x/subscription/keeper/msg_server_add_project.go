package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/subscription/types"
)

func (k msgServer) AddProject(goCtx context.Context, msg *types.MsgAddProject) (*types.MsgAddProjectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgAddProjectResponse{}, nil
}
