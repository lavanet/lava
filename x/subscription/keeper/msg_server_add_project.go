package keeper

import (
	"context"
	"fmt"

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

		if !projectKey.IsTypeValid() {
			return nil, utils.LavaFormatWarning(
				"invalid project key type (must be ADMIN(=1) or DEVELOPER(=2)",
				fmt.Errorf("invalid project key type"),
				utils.Attribute{Key: "key", Value: projectKey.Key},
				utils.Attribute{Key: "keyType", Value: projectKey.Kinds},
			)
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
