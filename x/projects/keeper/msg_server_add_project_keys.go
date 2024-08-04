package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/projects/types"
)

func (k msgServer) AddKeys(goCtx context.Context, msg *types.MsgAddKeys) (*types.MsgAddKeysResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return nil, utils.LavaFormatWarning("invalid address", err,
			utils.LogAttr("creator", msg.Creator),
		)
	}

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

	err := k.AddKeysToProject(ctx, msg.Project, msg.Creator, msg.ProjectKeys)
	if err != nil {
		return nil, err
	}

	return &types.MsgAddKeysResponse{}, nil
}
