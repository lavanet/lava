package keeper

import (
	"fmt"
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

func (k Keeper) GetProjectDeveloperData(ctx sdk.Context, developerKey string, blockHeight uint64) (types.ProtoDeveloperData, error) {
	var projectDeveloperData types.ProtoDeveloperData
	err, found := k.developerKeysFS.FindEntry(ctx, developerKey, blockHeight, &projectDeveloperData)
	if err != nil || !found {
		return types.ProtoDeveloperData{}, fmt.Errorf("GetProjectIDForDeveloper_invalid_key, the requesting key is not registered to a project, developer: %s", developerKey)
	}
	return projectDeveloperData, nil
}

func (k Keeper) GetProjectForDeveloper(ctx sdk.Context, developerKey string, blockHeight uint64) (proj types.Project, vrfpk string, errRet error) {
	var project types.Project
	projectDeveloperData, err := k.GetProjectDeveloperData(ctx, developerKey, blockHeight)
	if err != nil {
		return project, "", err
	}

	err, found := k.projectsFS.FindEntry(ctx, projectDeveloperData.ProjectID, blockHeight, &project)
	if err != nil {
		return project, "", err
	}

	if !found {
		return project, "", utils.LavaError(ctx, ctx.Logger(), "GetProjectForDeveloper_project_not_found", map[string]string{"developer": developerKey, "project": projectDeveloperData.ProjectID}, "the developers project was not found")
	}

	return project, projectDeveloperData.Vrfpk, nil
}

func (k Keeper) AddKeysToProject(ctx sdk.Context, projectID string, adminKey string, projectKeys []types.ProjectKey) error {
	var project types.Project
	err, found := k.projectsFS.FindEntry(ctx, projectID, uint64(ctx.BlockHeight()), &project)
	if err != nil || !found {
		return utils.LavaError(ctx, ctx.Logger(), "AddProjectKeys_project_not_found", map[string]string{"project": projectID}, "project id not found")
	}

	// check if the admin key is valid
	if !project.IsAdminKey(adminKey) {
		return utils.LavaError(ctx, ctx.Logger(), "AddProjectKeys_not_admin", map[string]string{"project": projectID}, "the requesting key is not admin key")
	}

	for _, projectKey := range projectKeys {
		k.RegisterKey(ctx, projectKey, &project, uint64(ctx.BlockHeight()))
	}

	return k.projectsFS.AppendEntry(ctx, projectID, uint64(ctx.BlockHeight()), &project)
}

func (k Keeper) GetProjectDevelopersPolicy(ctx sdk.Context, developerKey string, blockHeight uint64) (adminPolicy types.Policy, subscriptionPolicy types.Policy, err error) {
	project, _, err := k.GetProjectForDeveloper(ctx, developerKey, blockHeight)
	if err != nil {
		return types.Policy{}, types.Policy{}, err
	}

	return project.AdminPolicy, project.SubscriptionPolicy, nil
}

func (k Keeper) ChargeComputeUnitsToProject(ctx sdk.Context, project types.Project, cu uint64) (err error) {
	project.UsedCu += cu
	return k.projectsFS.ModifyEntry(ctx, project.Index, uint64(ctx.BlockHeight()), &project)
}

func (k Keeper) SetPolicy(ctx sdk.Context, projectIDs []string, policy types.Policy, key string, isAdminPolicy bool) error {
	for _, projectID := range projectIDs {
		project, err := k.GetProjectForBlock(ctx, projectID, uint64(ctx.BlockHeight()))
		if err != nil {
			return utils.LavaError(ctx, ctx.Logger(), "SetPolicy_project_not_found", map[string]string{"project": projectID}, "project id not found")
		}
		// for admin policy - check if the key is an address of a project admin
		if isAdminPolicy {
			if !project.IsAdminKey(key) {
				return utils.LavaError(ctx, ctx.Logger(), "SetPolicy_not_admin", map[string]string{"project": projectID, "key": key}, "cannot set admin policy because the requesting key is not admin key")
			} else {
				project.AdminPolicy = policy
			}
		}

		// for subscription policy - check if the key is an address of the project's subscription consumer
		if !isAdminPolicy {
			if key != project.GetSubscription() {
				return utils.LavaError(ctx, ctx.Logger(), "SetPolicy_not_subscription_consumer", map[string]string{"project": projectID, "key": key}, "cannot set subscription policy because the requesting key is not subscription consumer key")
			} else {
				project.SubscriptionPolicy = policy
			}
		}

		nextEpoch, err := k.epochStorageKeeper.GetNextEpoch(ctx, uint64(ctx.BlockHeight()))
		if err != nil {
			return utils.LavaError(ctx, k.Logger(ctx), "SetPolicy_cant_get_next_epoch", map[string]string{"block": strconv.FormatUint(uint64(ctx.BlockHeight()), 10)}, "can't get next epoch")
		}
		err = k.projectsFS.AppendEntry(ctx, projectID, nextEpoch, &project)
		if err != nil {
			return err
		}
	}

	return nil
}
