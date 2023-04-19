package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/projects/types"
)

func (k msgServer) SetProjectPolicy(goCtx context.Context, msg *types.MsgSetProjectPolicy) (*types.MsgSetProjectPolicyResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	projectID := msg.Project
	adminKey := msg.Creator
	var project types.Project

	if found := k.projectsFS.FindEntry(ctx, projectID, uint64(ctx.BlockHeight()), &project); !found {
		return nil, utils.LavaError(ctx, ctx.Logger(), "SetProjectPolicy_project_not_found", map[string]string{"project": projectID}, "project id not found")
	}

	// check if the admin key is valid
	if !project.IsAdminKey(adminKey) {
		return nil, utils.LavaError(ctx, ctx.Logger(), "SetProjectPolicy_not_admin", map[string]string{"project": projectID}, "the requesting key is not admin key")
	}

	project.Policy = *msg.Policy

	if project.UsedCu > project.Policy.TotalCuLimit {
		project.Policy.TotalCuLimit = project.UsedCu
	}

	// TODO this needs to be applied in the next epoch
	err := k.projectsFS.AppendEntry(ctx, projectID, uint64(ctx.BlockHeight()), &project)
	if err != nil {
		return nil, err
	}

	return &types.MsgSetProjectPolicyResponse{}, nil
}
