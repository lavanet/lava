package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/projects/types"
)

func (k msgServer) DelKeys(goCtx context.Context, msg *types.MsgDelKeys) (*types.MsgDelKeysResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	for _, projectKey := range msg.GetProjectKeys() {
		if !projectKey.IsTypeValid() {
			return nil, utils.LavaFormatWarning(
				"invalid project key type (must be ADMIN(=1) or DEVELOPER(=2)",
				fmt.Errorf("invalid project key type"),
				utils.Attribute{Key: "key", Value: projectKey.Key},
				utils.Attribute{Key: "keyType", Value: projectKey.Kinds},
			)
		}
	}

	err := k.DelKeysFromProject(ctx, msg.Project, msg.Creator, msg.ProjectKeys)
	if err != nil {
		return nil, err
	}

	return &types.MsgDelKeysResponse{}, nil
}
