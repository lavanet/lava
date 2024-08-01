package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/subscription/types"
)

func (k msgServer) DelProject(goCtx context.Context, msg *types.MsgDelProject) (*types.MsgDelProjectResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return nil, utils.LavaFormatError("Invalid creator address", err,
			utils.LogAttr("creator", msg.Creator),
		)
	}

	err := k.Keeper.DelProjectFromSubscription(ctx, msg.GetCreator(), msg.GetName())
	if err == nil {
		logger := k.Keeper.Logger(ctx)
		details := map[string]string{
			"subscription": msg.GetCreator(),
			"projectName":  msg.GetName(),
		}
		utils.LogLavaEvent(ctx, logger, types.DelProjectEventName, details, "project deleted from subscription")
	}

	return &types.MsgDelProjectResponse{}, err
}
