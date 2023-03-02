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

func (k Keeper) ValidateDeveloperRequest(ctx sdk.Context, developerKey string, chainID string, apiName string, blockHeight uint64) (valid bool, policy types.Policy, err error) {
	project, err := k.GetProjectForDeveloper(ctx, developerKey, blockHeight)
	if err != nil {
		return false, types.Policy{}, err
	}

	if project.UsedCu >= project.Policy.TotalCuLimit {
		return false, project.Policy, nil
	}

	for _, chain := range project.Policy.ChainPolicies {
		if chain.ChainId == chainID {
			for _, api := range chain.Apis {
				if api == apiName {
					return true, project.Policy, nil
				}
			}
		}
	}

	return false, project.Policy, nil
}
