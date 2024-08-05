package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	projectstypes "github.com/lavanet/lava/v2/x/projects/types"
	"github.com/lavanet/lava/v2/x/subscription/types"
)

func (k msgServer) AddProject(goCtx context.Context, msg *types.MsgAddProject) (*types.MsgAddProjectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return nil, utils.LavaFormatError("Invalid creator address", err,
			utils.LogAttr("creator", msg.Creator),
		)
	}

	for _, projectKey := range msg.GetProjectData().ProjectKeys {
		_, err := sdk.AccAddressFromBech32(projectKey.GetKey())
		if err != nil {
			return nil, utils.LavaFormatWarning("cannot add project with invalid project key to subscription", err,
				utils.Attribute{Key: "key", Value: projectKey.Key},
			)
		}

		if !projectKey.IsTypeValid() {
			return nil, utils.LavaFormatWarning(
				"invalid project key type (must be ADMIN(=1) or DEVELOPER(=2) or ADMIN+DEVELOPER(=3)",
				fmt.Errorf("invalid project key type"),
				utils.Attribute{Key: "key", Value: projectKey.Key},
				utils.Attribute{Key: "keyType", Value: projectKey.Kinds},
			)
		}
	}

	if !projectstypes.ValidateProjectName(msg.GetProjectData().Name) {
		return nil, utils.LavaFormatWarning("cannot add project with invalid name to subscription", fmt.Errorf("invalid project name"),
			utils.Attribute{Key: "name", Value: msg.ProjectData.Name},
		)
	}

	if msg.ProjectData.Policy != nil && msg.ProjectData.Policy.MaxProvidersToPair <= 1 {
		return nil, utils.LavaFormatWarning("cannot add project with invalid providersToPair to subscription (must be >1)", fmt.Errorf("invalid policy"),
			utils.Attribute{Key: "name", Value: msg.ProjectData.Name},
		)
	}

	err := k.Keeper.AddProjectToSubscription(ctx, msg.GetCreator(), msg.GetProjectData())
	if err == nil {
		logger := k.Keeper.Logger(ctx)
		details := map[string]string{
			"subscription": msg.GetCreator(),
			"projectName":  msg.GetProjectData().Name,
		}
		utils.LogLavaEvent(ctx, logger, types.AddProjectEventName, details, "project added to subscription")
	}

	return &types.MsgAddProjectResponse{}, err
}
