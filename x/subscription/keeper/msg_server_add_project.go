package keeper

import (
	"context"
	"fmt"
	"strconv"

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
			details := map[string]string{
				"key": projectKey.Key,
				"err": err.Error(),
			}
			return nil, utils.LavaError(ctx, k.Logger(ctx), "AddProject_invalid_project_key", details, "invalid project key")
		}

		if !projectKey.IsTypeValid() {
			return nil, fmt.Errorf("invalid project key: %d (must be ADMIN(=1) or DEVELOPER(=2)", projectKey.Kinds)
		}

		if !projectstypes.ValidateProjectNameAndDescription(msg.GetProjectData().Name, msg.GetProjectData().Description) {
			details := map[string]string{
				"name":        msg.GetProjectData().Name,
				"description": msg.GetProjectData().Description,
			}
			return nil, utils.LavaError(ctx, k.Logger(ctx), "AddProject_invalid_project_name_or_description", details, "invalid project name or description (might be too long or include disallowed characters)")
		}

		if msg.GetProjectData().Policy.MaxProvidersToPair <= 1 {
			details := map[string]string{
				"maxProvidersToPair": strconv.FormatUint(msg.GetProjectData().Policy.MaxProvidersToPair, 10),
			}
			return nil, utils.LavaError(ctx, k.Logger(ctx), "AddProject_invalid_project_providers_to_pair", details, "invalid project providersToPair (must be larger than one)")
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
