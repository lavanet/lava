package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/subscription/types"
)

func (k msgServer) AddProject(goCtx context.Context, msg *types.MsgAddProject) (*types.MsgAddProjectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

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
