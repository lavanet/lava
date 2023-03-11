package keeper

import (
	"errors"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/projects/types"
)

func (k Keeper) GetProjectForBlock(ctx sdk.Context, projectID string, blockHeight uint64) (types.Project, error) {
	var project types.Project

	err, found := k.projectsFS.FindEntry(ctx, projectID, blockHeight, &project)
	if err != nil || !found {
		return project, utils.LavaError(ctx, ctx.Logger(), "GetProjectForBlock_not_found", map[string]string{"project": projectID, "blockHeight": strconv.FormatUint(blockHeight, 10)}, "project not found")
	}

	return project, nil
}

func (k Keeper) GetProjectIDForDeveloper(ctx sdk.Context, developerKey string, blockHeight uint64) (string, error) {
	var projectIDstring types.ProtoString
	err, found := k.developerKeysFS.FindEntry(ctx, developerKey, blockHeight, &projectIDstring)
	if err != nil || !found {
		return "", utils.LavaError(ctx, ctx.Logger(), "GetProjectIDForDeveloper_invalid_key", map[string]string{"developer": developerKey}, "the requesting key is not registered to a project")
	}

	return projectIDstring.String_, nil
}

func (k Keeper) GetProjectForDeveloper(ctx sdk.Context, developerKey string, blockHeight uint64) (types.Project, error) {
	var project types.Project
	projectID, err := k.GetProjectIDForDeveloper(ctx, developerKey, blockHeight)
	if err != nil {
		return project, err
	}

	err, found := k.projectsFS.FindEntry(ctx, projectID, blockHeight, &project)
	if err != nil {
		return project, err
	}

	if !found {
		return project, utils.LavaError(ctx, ctx.Logger(), "GetProjectForDeveloper_project_not_found", map[string]string{"developer": developerKey, "project": projectID}, "the developers project was not found")
	}

	return project, nil
}

func (k Keeper) AddKeysToProject(ctx sdk.Context, projectID string, adminKey string, projectKeys []types.ProjectKey) error {
	var project types.Project
	err, found := k.projectsFS.FindEntry(ctx, projectID, uint64(ctx.BlockHeight()), &project)
	if err != nil || !found {
		return utils.LavaError(ctx, ctx.Logger(), "AddProjectKeys_project_not_found", map[string]string{"project": projectID}, "project id not found")
	}

	// check if the admin key is valid
	if !project.HasKeyType(adminKey, types.ProjectKey_ADMIN) && project.Subscription != adminKey {
		return utils.LavaError(ctx, ctx.Logger(), "AddProjectKeys_not_admin", map[string]string{"project": projectID}, "the requesting key is not admin key")
	}

	// check that those keys are unique for developers
	for _, projectKey := range projectKeys {
		err = k.RegisterDeveloperKey(ctx, projectKey.Key, project.Index, uint64(ctx.BlockHeight()))
		if err != nil {
			return err
		}

		project.AppendKey(projectKey)
	}

	return k.projectsFS.AppendEntry(ctx, projectID, uint64(ctx.BlockHeight()), &project)
}

func (k Keeper) GetProjectDevelopersPolicy(ctx sdk.Context, developerKey string, blockHeight uint64) (policy types.Policy, err error) {
	project, err := k.GetProjectForDeveloper(ctx, developerKey, blockHeight)
	if err != nil {
		return types.Policy{}, err
	}

	// if project.UsedCu >= project.Policy.TotalCuLimit {
	// 	return false, project.Policy, nil
	// }

	return project.Policy, nil
}

func (k Keeper) AddComputeUnitsToProject(ctx sdk.Context, developerKey string, blockHeight uint64) (err error) {
	// TODO
	return errors.New("AddComputeUnitsToProject not implemented")
}
