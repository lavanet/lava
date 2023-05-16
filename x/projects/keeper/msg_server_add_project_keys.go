package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/projects/types"
)

func (k msgServer) AddKeys(goCtx context.Context, msg *types.MsgAddKeys) (*types.MsgAddKeysResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	for _, projectKey := range msg.GetProjectKeys() {
		if !projectKey.IsTypeValid() {
			return nil, fmt.Errorf("invalid project key: %d (must be ADMIN(=1) or DEVELOPER(=2)", projectKey.Kinds)
		}
	}

	err := k.AddKeysToProject(ctx, msg.Project, msg.Creator, msg.ProjectKeys)
	if err != nil {
		return nil, err
	}
	return &types.MsgAddKeysResponse{}, nil
}
