package keeper

import (
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/projects/types"
)

// add a default project to a subscription
func (k Keeper) GetProjectForBlock(ctx sdk.Context, projectID string, blockHeight uint64) (types.Project, error) {
	var project types.Project

	err := k.projectsFS.FindEntry(ctx, projectID, blockHeight, &project)
	if err != nil {
		return project, utils.LavaError(ctx, ctx.Logger(), "GetProjectForBlock_not_found", map[string]string{"project": projectID, "blockHeight": strconv.FormatUint(blockHeight, 10)}, "default project already exist for the current subscription")
	}

	return project, nil
}

func (k Keeper) AddProjectKeys(ctx sdk.Context, projectID string, adminKey string, projectKeys []types.ProjectKey) error {
	var project types.Project

	err := k.projectsFS.FindEntry(ctx, projectID, uint64(ctx.BlockHeight()), &project)
	if err != nil {
		return utils.LavaError(ctx, ctx.Logger(), "AddProjectKeys_project_not_found", map[string]string{"project": projectID}, "project id not found")
	}

	// check if the admin key is valid
	if !project.IsKeyType(adminKey, types.ProjectKey_ADMIN) || project.Subscription != adminKey {
		return utils.LavaError(ctx, ctx.Logger(), "AddProjectKeys_not_admin", map[string]string{"project": projectID}, "the requesting key is not admin key")
	}

	// check that those keys are unique for developers
	for _, projectKey := range projectKeys {
		if projectKey.IsKeyType(types.ProjectKey_DEVELOPER) {
			// if the key is a developer key add it to the map of developer keys
			var projectIDstring types.ProtoString
			err = k.developerKeysFS.FindEntry(ctx, projectKey.Key, uint64(ctx.BlockHeight()), &projectIDstring)
			if err == nil {
				projectIDstring.String_ = project.Index
				err = k.developerKeysFS.AppendEntry(ctx, project.Index, uint64(ctx.BlockHeight()), &projectIDstring)
				if err != nil {
					return utils.LavaError(ctx, ctx.Logger(), "AddProjectKeys_used_key", map[string]string{"project": projectID, "developerKey": projectKey.Key, "err": err.Error()}, "the requesting key is not admin key")
				}
			}
		}
	}

	return k.projectsFS.AppendEntry(ctx, projectID, uint64(ctx.BlockHeight()), &project)
}

func (k Keeper) SetProjectPolicy(ctx sdk.Context, projectID string, adminKey string, policy types.Policy) error {
	var project types.Project

	err := k.projectsFS.FindEntry(ctx, projectID, uint64(ctx.BlockHeight()), &project)
	if err != nil {
		return utils.LavaError(ctx, ctx.Logger(), "SetProjectPolicy_project_not_found", map[string]string{"project": projectID}, "project id not found")
	}

	// check if the admin key is valid
	if !project.IsKeyType(adminKey, types.ProjectKey_ADMIN) || project.Subscription != adminKey {
		return utils.LavaError(ctx, ctx.Logger(), "SetProjectPolicy_not_admin", map[string]string{"project": projectID}, "the requesting key is not admin key")
	}

	project.Policy = policy

	if project.UsedCu > project.Policy.TotalCuLimit {
		policy.TotalCuLimit = project.UsedCu
	}

	// TODO this needs to be applied in the next epoch
	return k.projectsFS.AppendEntry(ctx, projectID, uint64(ctx.BlockHeight()), &project)
}
