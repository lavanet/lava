package keeper

import (
	"math"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	plantypes "github.com/lavanet/lava/x/plans/types"
	"github.com/lavanet/lava/x/projects/types"
)

// add a default project to a subscription, add the subscription key as
func (k Keeper) CreateAdminProject(ctx sdk.Context, subscriptionAddress string, plan plantypes.Plan, vrfpk string) error {
	return k.CreateProject(ctx, subscriptionAddress, types.ADMIN_PROJECT_NAME, subscriptionAddress, true, types.ADMIN_PROJECT_DESCRIPTION, plan, math.MaxUint64, vrfpk)
}

// add a new project to the subscription
func (k Keeper) CreateProject(ctx sdk.Context, subscriptionAddress string, projectName string, adminAddress string, enabled bool, projectDescription string, plan plantypes.Plan, geolocation uint64, vrfpk string) error {
	project := types.CreateProject(subscriptionAddress, projectName)
	var emptyProject types.Project

	blockHeight := uint64(ctx.BlockHeight())
	_, found := k.projectsFS.FindEntry(ctx, project.Index, blockHeight, &emptyProject)
	// the project with the same name already exists if no error has returned
	if found {
		return utils.LavaError(ctx, ctx.Logger(), "CreateEmptyProject_already_exist", map[string]string{"subscription": subscriptionAddress}, "project already exist for the current subscription with the same name")
	}

	project.Policy.EpochCuLimit = plan.GetComputeUnitsPerEpoch()
	project.Policy.TotalCuLimit = plan.GetComputeUnits()
	project.Policy.MaxProvidersToPair = plan.GetMaxProvidersToPair()
	project.Policy.GeolocationProfile = geolocation

	project.Enabled = enabled
	project.Description = projectDescription
	err := k.RegisterKey(ctx, types.ProjectKey{Key: adminAddress, Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_ADMIN, types.ProjectKey_DEVELOPER}, Vrfpk: vrfpk}, &project, blockHeight)
	if err != nil {
		return err
	}
	return k.projectsFS.AppendEntry(ctx, project.Index, blockHeight, &project)
}

func (k Keeper) RegisterKey(ctx sdk.Context, key types.ProjectKey, project *types.Project, blockHeight uint64) error {
	if project == nil {
		return utils.LavaError(ctx, k.Logger(ctx), "RegisterKey_project_is_nil", nil, "project is nil")
	}

	developerKeyFound := false
	inputKeyIsAdmin := false
	inputKeyIsDeveloper := false
	var developerData types.ProtoDeveloperData

	for _, keyType := range key.GetTypes() {
		if keyType == types.ProjectKey_ADMIN {
			inputKeyIsAdmin = true
		}

		if keyType == types.ProjectKey_DEVELOPER {
			inputKeyIsDeveloper = true
			_, found := k.developerKeysFS.FindEntry(ctx, key.GetKey(), blockHeight, &developerData)
			if found {
				developerKeyFound = true
			}
		}
	}

	// handle admin key type
	if inputKeyIsAdmin {
		k.AddAdminKey(ctx, project, key.GetKey(), key.GetVrfpk())
	}

	// handle case of admin-only key type (no need to register developer key)
	if !inputKeyIsDeveloper {
		return nil
	}

	// handle developer key type (and admin-developer key type)
	if developerKeyFound && inputKeyIsDeveloper {
		details := map[string]string{"key": key.GetKey(), "keyTypes": string(key.GetTypes())}
		return utils.LavaError(ctx, k.Logger(ctx), "RegisterKey_key_exists", details, "key already exists")
	}

	err := k.AddDeveloperKey(ctx, key.GetKey(), project, blockHeight, key.GetVrfpk(), &developerData)
	if err != nil {
		details := map[string]string{
			"developerKey": key.GetKey(),
			"projectIndex": project.GetIndex(),
			"blockHeight":  strconv.FormatUint(blockHeight, 10),
		}
		return utils.LavaError(ctx, k.Logger(ctx), "RegisterKey_add_dev_key_failed", details, "adding developer key failed")
	}

	return nil
}

func (k Keeper) AddAdminKey(ctx sdk.Context, project *types.Project, adminKey string, vrfpk string) {
	project.AppendKey(types.ProjectKey{Key: adminKey, Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_ADMIN}, Vrfpk: vrfpk})
}

func (k Keeper) AddDeveloperKey(ctx sdk.Context, developerKey string, project *types.Project, blockHeight uint64, vrfpk string, developerData *types.ProtoDeveloperData) error {
	if developerData == nil {
		return utils.LavaError(ctx, k.Logger(ctx), "AddDeveloperKey_developer_data_nil", nil, "developer data is nil")
	}

	developerData.ProjectID = project.GetIndex()
	developerData.Vrfpk = vrfpk
	err := k.developerKeysFS.AppendEntry(ctx, developerKey, blockHeight, developerData)
	if err != nil {
		return err
	}

	project.AppendKey(types.ProjectKey{Key: developerKey, Types: []types.ProjectKey_KEY_TYPE{types.ProjectKey_DEVELOPER}, Vrfpk: vrfpk})

	return nil
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
