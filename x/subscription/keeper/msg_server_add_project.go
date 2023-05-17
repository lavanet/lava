package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	projectstypes "github.com/lavanet/lava/x/projects/types"
	"github.com/lavanet/lava/x/subscription/types"
)

func (k msgServer) AddProject(goCtx context.Context, msg *types.MsgAddProject) (*types.MsgAddProjectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	for _, projectKey := range msg.GetProjectData().ProjectKeys {
		_, err := sdk.AccAddressFromBech32(projectKey.GetKey())
		if err != nil {
			return nil, utils.LavaFormatWarning("cannot add project with invalid project key to subscription", err,
				utils.Attribute{Key: "key", Value: projectKey.Key},
			)
		}

		for _, projectKeyType := range projectKey.Types {
			if projectKeyType != projectstypes.ProjectKey_ADMIN && projectKeyType != projectstypes.ProjectKey_DEVELOPER {
				return nil, utils.LavaFormatWarning("cannot add project with invalid project key type to subscription", err,
					utils.Attribute{Key: "type", Value: projectKeyType},
				)
			}
		}

		if !projectstypes.ValidateProjectNameAndDescription(msg.GetProjectData().Name, msg.GetProjectData().Description) {
			return nil, utils.LavaFormatWarning("cannot add project with invalid name/description to subscription", err,
				utils.Attribute{Key: "name", Value: msg.ProjectData.Name},
				utils.Attribute{Key: "description", Value: msg.ProjectData.Description},
			)
		}

		if msg.GetProjectData().Policy.MaxProvidersToPair <= 1 {
			return nil, utils.LavaFormatWarning("cannot add project with invalid providersToPair to subscription (must be >1)", err,
				utils.Attribute{Key: "name", Value: msg.ProjectData.Name},
				utils.Attribute{Key: "description", Value: msg.ProjectData.Description},
			)
		}
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
