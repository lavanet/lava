package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/projects/types"
)

// add a default project to a subscription
func (k Keeper) CreateDefaultProject(ctx sdk.Context, subscriptionAddress string) error {
	project := types.DefualtProject(subscriptionAddress)
	var emptyProject types.Project

	err := k.projectsFS.FindEntry(ctx, project.Index, uint64(ctx.BlockHeight()), &emptyProject)
	if err == nil {
		return utils.LavaError(ctx, ctx.Logger(), "CreateDefaultProject_already_exist", map[string]string{"subscription": subscriptionAddress}, "default project already exist for the current subscription")
	}

	return k.projectsFS.AppendEntry(ctx, project.Index, uint64(ctx.BlockHeight()), &project)
}

// add a new project to the subscription
func (k Keeper) CreateEmptyProject(ctx sdk.Context, subscriptionAddress string, projectName string, adminAddress string) error {
	project := types.CreateEmptyProject(subscriptionAddress, projectName)
	var emptyProject types.Project

	err := k.projectsFS.FindEntry(ctx, project.Index, uint64(ctx.BlockHeight()), &emptyProject)
	// the project with the same name already exists if no error has returned
	if err == nil {
		return utils.LavaError(ctx, ctx.Logger(), "CreateEmptyProject_already_exist", map[string]string{"subscription": subscriptionAddress}, "project already exist for the current subscription with the same name")
	}

	project.ProjectKeys = append(project.ProjectKeys, types.ProjectKey{Key: adminAddress, Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_ADMIN}})

	var projectID types.ProtoString
	err = k.developerKeysFS.FindEntry(ctx, adminAddress, uint64(ctx.BlockHeight()), &projectID)
	// a developer key with this address is not registered, add it to the developer keys list
	if err == nil {
		projectID.String_ = project.Index
		err = k.developerKeysFS.AppendEntry(ctx, project.Index, uint64(ctx.BlockHeight()), &projectID)
		if err != nil {
			return err
		}
	}

	return k.projectsFS.AppendEntry(ctx, project.Index, uint64(ctx.BlockHeight()), &project)
}

// snapshot project, create a snapshot of a project and reset the cu
func (k Keeper) SnapshotProject(ctx sdk.Context, projectID string) error {
	var project types.Project
	err := k.projectsFS.FindEntry(ctx, projectID, uint64(ctx.BlockHeight()), &project)
	if err != nil {
		return utils.LavaError(ctx, ctx.Logger(), "SnapshotProject_project_not_found", map[string]string{"projectID": projectID}, "snapshot of project failed, oriject does not exist")
	}

	project.UsedCu = 0

	return k.projectsFS.AppendEntry(ctx, project.Index, uint64(ctx.BlockHeight()), &project)
}
