package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/subscription/types"
)

func (k msgServer) AddProject(goCtx context.Context, msg *types.MsgAddProject) (*types.MsgAddProjectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	err := k.Keeper.AddProjectToSubscription(ctx, msg.GetCreator(), msg.GetConsumer(), msg.GetProjectName(), msg.GetEnable(), msg.GetVrfpk())
	if err == nil {
		logger := k.Keeper.Logger(ctx)
		details := map[string]string{
			"subscriptionOwner": msg.GetCreator(),
			"projectName":       msg.GetProjectName(),
			"projectAdmin":      msg.GetConsumer(),
		}
		utils.LogLavaEvent(ctx, logger, types.AddProjectEventName, details, "consumer added project to subscription")
	}

	return &types.MsgAddProjectResponse{}, err
}
