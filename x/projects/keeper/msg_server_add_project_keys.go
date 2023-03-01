package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/projects/types"
)

func (k msgServer) AddProjectKeys(goCtx context.Context, msg *types.MsgAddProjectKeys) (*types.MsgAddProjectKeysResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	projectID := msg.Project
	adminKey := msg.Creator
	projectKeys := msg.ProjectKeys

	var project types.Project
	err := k.projectsFS.FindEntry(ctx, projectID, uint64(ctx.BlockHeight()), &project)
	if err != nil {
		return nil, utils.LavaError(ctx, ctx.Logger(), "AddProjectKeys_project_not_found", map[string]string{"project": projectID}, "project id not found")
	}

	// check if the admin key is valid
	if !project.IsKeyType(adminKey, types.ProjectKey_ADMIN) && project.Subscription != adminKey {
		return nil, utils.LavaError(ctx, ctx.Logger(), "AddProjectKeys_not_admin", map[string]string{"project": projectID}, "the requesting key is not admin key")
	}

	// check that those keys are unique for developers
	for _, projectKey := range projectKeys {
		// check if this key is registered as a developer key
		var projectIDstring types.ProtoString
		err = k.developerKeysFS.FindEntry(ctx, projectKey.Key, uint64(ctx.BlockHeight()), &projectIDstring)

		// not registered
		if err != nil {
			projectIDstring.String_ = project.Index
			err = k.developerKeysFS.AppendEntry(ctx, projectKey.Key, uint64(ctx.BlockHeight()), &projectIDstring)
			if err != nil {
				return nil, utils.LavaError(ctx, ctx.Logger(), "AddProjectKeys_used_key", map[string]string{"project": projectID, "developerKey": projectKey.Key, "err": err.Error()}, "failed to register key")
			}
		}

		project.AppendKey(projectKey)
	}

	err = k.projectsFS.AppendEntry(ctx, projectID, uint64(ctx.BlockHeight()), &project)
	if err != nil {
		return nil, err
	}

	return &types.MsgAddProjectKeysResponse{}, err
}
