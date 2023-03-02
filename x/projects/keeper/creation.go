package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/projects/types"
)

// add a default project to a subscription, add the subscription key as
func (k Keeper) CreateDefaultProject(ctx sdk.Context, subscriptionAddress string) error {
	return k.CreateProject(ctx, subscriptionAddress, types.DEFAULT_PROJECT_NAME, subscriptionAddress, true)
}

// add a new project to the subscription
func (k Keeper) CreateProject(ctx sdk.Context, subscriptionAddress string, projectName string, adminAddress string, enable bool) error {
	project := types.CreateEmptyProject(subscriptionAddress, projectName)
	var emptyProject types.Project

	_, found := k.projectsFS.FindEntry(ctx, project.Index, uint64(ctx.BlockHeight()), &emptyProject)
	// the project with the same name already exists if no error has returned
	if found {
		return utils.LavaError(ctx, ctx.Logger(), "CreateEmptyProject_already_exist", map[string]string{"subscription": subscriptionAddress}, "project already exist for the current subscription with the same name")
	}

	if subscriptionAddress != adminAddress {
		project.AppendKey(types.ProjectKey{Key: adminAddress, Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_ADMIN}})
	}

	var projectID types.ProtoString
	_, found = k.developerKeysFS.FindEntry(ctx, adminAddress, uint64(ctx.BlockHeight()), &projectID)
	// a developer key with this address is not registered, add it to the developer keys list
	if !found {
		projectID.String_ = project.Index
		err := k.developerKeysFS.AppendEntry(ctx, adminAddress, uint64(ctx.BlockHeight()), &projectID)
		if err != nil {
			return err
		}
	}

	project.Enabled = enable
	return k.projectsFS.AppendEntry(ctx, project.Index, uint64(ctx.BlockHeight()), &project)
}

// snapshot project, create a snapshot of a project and reset the cu
func (k Keeper) SnapshotProject(ctx sdk.Context, projectID string) error {
	var project types.Project
	err, found := k.projectsFS.FindEntry(ctx, projectID, uint64(ctx.BlockHeight()), &project)
	if err != nil || !found {
		return utils.LavaError(ctx, ctx.Logger(), "SnapshotProject_project_not_found", map[string]string{"projectID": projectID}, "snapshot of project failed, project does not exist")
	}

	project.UsedCu = 0

	return k.projectsFS.AppendEntry(ctx, project.Index, uint64(ctx.BlockHeight()), &project)
}

func (k Keeper) DeleteProject(ctx sdk.Context, projectID string) error {
	var project types.Project
	err, found := k.projectsFS.FindEntry(ctx, projectID, uint64(ctx.BlockHeight()), &project)
	if err != nil || !found {
		return utils.LavaError(ctx, ctx.Logger(), "DeleteProject_project_not_found", map[string]string{"projectID": projectID}, "project to delete was not found")
	}

	project.Enabled = false
	// TODO: delete all developer keys from the fixation

	return k.projectsFS.AppendEntry(ctx, project.Index, uint64(ctx.BlockHeight()), &project)
}
