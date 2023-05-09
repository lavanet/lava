package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/projects/types"
)

func (k msgServer) AddKeys(goCtx context.Context, msg *types.MsgAddKeys) (*types.MsgAddKeysResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.AddKeysToProject(ctx, msg.Project, msg.Creator, msg.ProjectKeys)
	if err != nil {
		return nil, err
	}
	return &types.MsgAddKeysResponse{}, nil
}
