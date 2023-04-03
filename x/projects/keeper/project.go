package keeper

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/projects/types"
)

func (k Keeper) GetSubscriptionProjects(ctx sdk.Context, subscriptionAddr string) []string {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubscriptionProjectsPrefix))

	b := store.Get(types.SubscriptionKey(subscriptionAddr))
	if b == nil {
		return []string{}
	}

	var val types.Subscription
	k.cdc.MustUnmarshal(b, &val)

	return val.Projects
}

func (k Keeper) SetSubscriptionProjects(ctx sdk.Context, subscriptionAddr string, projects []string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SubscriptionProjectsPrefix))

	val := types.Subscription{
		Projects: projects,
	}

	b := k.cdc.MustMarshal(&val)
	store.Set(types.SubscriptionKey(subscriptionAddr), b)
}

func (k Keeper) AppendSubscriptionProject(ctx sdk.Context, subscriptionAddr string, projectID string) {
	projects := k.GetSubscriptionProjects(ctx, subscriptionAddr)
	projects = append(projects, projectID)
	k.SetSubscriptionProjects(ctx, subscriptionAddr, projects)
}

func (k Keeper) RemoveSubscriptionProject(ctx sdk.Context, subscriptionAddr string, projectID string) {
	projects := k.GetSubscriptionProjects(ctx, subscriptionAddr)

	length := len(projects)
	for i := 0; i < length; i++ {
		if projects[i] == projectID {
			if i >= length {
				projects = []string{}
			} else {
				projects[i] = projects[length-1]
				projects = projects[:length-1]
			}
			break
		}
	}

	k.SetSubscriptionProjects(ctx, subscriptionAddr, projects)
}

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

	// check that those keys are unique for developers
	for _, projectKey := range projectKeys {
		err = k.RegisterDeveloperKey(ctx, projectKey.Key, project.Index, uint64(ctx.BlockHeight()), projectKey.Vrfpk)
		if err != nil {
			return err
		}

		project.AppendKey(projectKey)
	}

	return k.projectsFS.AppendEntry(ctx, projectID, uint64(ctx.BlockHeight()), &project)
}

func (k Keeper) GetProjectDevelopersPolicy(ctx sdk.Context, developerKey string, blockHeight uint64) (policy types.Policy, err error) {
	project, _, err := k.GetProjectForDeveloper(ctx, developerKey, blockHeight)
	if err != nil {
		return types.Policy{}, err
	}

	return project.Policy, nil
}

func (k Keeper) ChargeProject(ctx sdk.Context, developerKey string, blockHeight uint64, cu uint64) (err error) {
	project, _, err := k.GetProjectForDeveloper(ctx, developerKey, blockHeight)
	if err != nil {
		return err
	}

	err = project.VerifyCuUsage()
	if err != nil {
		return err
	}

	project.UsedCu += cu

	return k.projectsFS.ModifyEntry(ctx, project.Index, blockHeight, &project)
}
