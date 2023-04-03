package keeper

import (
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/projects/types"
)

// add a default project to a subscription, add the subscription key as
func (k Keeper) CreateAdminProject(ctx sdk.Context, subscriptionAddress string, totalCU uint64, cuPerEpoch uint64, providers uint64, vrfpk string) error {
	return k.CreateProject(ctx, subscriptionAddress, types.ADMIN_PROJECT_NAME, subscriptionAddress, true, totalCU, cuPerEpoch, providers, math.MaxUint64, vrfpk)
}

// add a new project to the subscription
func (k Keeper) CreateProject(ctx sdk.Context, subscriptionAddress string, projectName string, adminAddress string, enable bool, totalCU uint64, cuPerEpoch uint64, providers uint64, geolocation uint64, vrfpk string) error {
	project := types.NewProject(subscriptionAddress, projectName)
	var emptyProject types.Project

	blockHeight := uint64(ctx.BlockHeight())
	_, found := k.projectsFS.FindEntry(ctx, project.Index, blockHeight, &emptyProject)
	// the project with the same name already exists if no error has returned
	if found {
		return utils.LavaError(ctx, ctx.Logger(), "CreateEmptyProject_already_exist", map[string]string{"subscription": subscriptionAddress}, "project already exist for the current subscription with the same name")
	}

	project.Policy.EpochCuLimit = cuPerEpoch
	project.Policy.TotalCuLimit = totalCU
	project.Policy.MaxProvidersToPair = providers
	project.Policy.GeolocationProfile = geolocation

	project.AppendKey(types.ProjectKey{Key: adminAddress, Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_ADMIN}, Vrfpk: vrfpk})

	err := k.RegisterDeveloperKey(ctx, adminAddress, project.Index, blockHeight, vrfpk)
	if err != nil {
		return err
	}

	project.Enabled = enable

	k.AppendSubscriptionProject(ctx, subscriptionAddress, project.Index)
	return k.projectsFS.AppendEntry(ctx, project.Index, blockHeight, &project)
}

func (k Keeper) RegisterDeveloperKey(ctx sdk.Context, developerKey string, projectIndex string, blockHeight uint64, vrfpk string) error {
	var developerData types.ProtoDeveloperData
	_, found := k.developerKeysFS.FindEntry(ctx, developerKey, blockHeight, &developerData)
	// a developer key with this address is not registered, add it to the developer keys list
	if !found {
		developerData.ProjectID = projectIndex
		developerData.Vrfpk = vrfpk
		err := k.developerKeysFS.AppendEntry(ctx, developerKey, blockHeight, &developerData)
		if err != nil {
			return err
		}
	}

	return nil
}

// snapshot project, create a snapshot of a project and reset the cu
func (k Keeper) snapshotProject(ctx sdk.Context, projectID string) error {
	var project types.Project
	err, found := k.projectsFS.FindEntry(ctx, projectID, uint64(ctx.BlockHeight()), &project)
	if err != nil || !found {
		return utils.LavaError(ctx, ctx.Logger(), "SnapshotProject_project_not_found", map[string]string{"projectID": projectID}, "snapshot of project failed, project does not exist")
	}

	project.UsedCu = 0

	return k.projectsFS.AppendEntry(ctx, project.Index, uint64(ctx.BlockHeight()), &project)
}

func (k Keeper) SnapshotSubscriptionProjects(ctx sdk.Context, subscriptionAddr string) {
	projects := k.GetSubscriptionProjects(ctx, subscriptionAddr)
	for _, projectID := range projects {
		err := k.snapshotProject(ctx, projectID)
		if err != nil {
			panic(err)
		}
	}
}

func (k Keeper) DeleteProject(ctx sdk.Context, projectID string) error {
	var project types.Project
	err, found := k.projectsFS.FindEntry(ctx, projectID, uint64(ctx.BlockHeight()), &project)
	if err != nil || !found {
		return utils.LavaError(ctx, ctx.Logger(), "DeleteProject_project_not_found", map[string]string{"projectID": projectID}, "project to delete was not found")
	}

	project.Enabled = false
	// TODO: delete all developer keys from the fixation
	// also really delete, and remove from subscription-projects
	// k.RemoveSubscriptionProject(ctx, subscriptionAddress, projectID)

	return k.projectsFS.AppendEntry(ctx, project.Index, uint64(ctx.BlockHeight()), &project)
}
