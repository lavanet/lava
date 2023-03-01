package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/projects/types"
)

func (k msgServer) AddProjectKeys(goCtx context.Context, msg *types.MsgAddProjectKeys) (*types.MsgAddProjectKeysResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgAddProjectKeysResponse{}, nil
}
