package keeper

import (
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/utils"
	plantypes "github.com/lavanet/lava/x/plans/types"
	"github.com/lavanet/lava/x/projects/types"
)

// add a default project to a subscription, add the subscription key as
func (k Keeper) CreateAdminProject(ctx sdk.Context, subAddr string, plan plantypes.Plan) error {
	projectData := types.ProjectData{
		Name:        types.ADMIN_PROJECT_NAME,
		Description: types.ADMIN_PROJECT_DESCRIPTION,
		ProjectKeys: []types.ProjectKey{types.ProjectDeveloperKey(subAddr)},
		Enabled:     true,
		Policy:      nil,
	}
	return k.CreateProject(ctx, subAddr, projectData, plan)
}

// add a new project to the subscription
func (k Keeper) CreateProject(ctx sdk.Context, subAddr string, projectData types.ProjectData, plan plantypes.Plan) error {
	project, err := types.NewProject(subAddr, projectData.GetName(), projectData.GetDescription(), projectData.GetEnabled())
	if err != nil {
		return err
	}

	var emptyProject types.Project
	blockHeight := uint64(ctx.BlockHeight())
	if found := k.projectsFS.FindEntry(ctx, project.Index, blockHeight, &emptyProject); found {
		// the project with the same name already exists if no error has returned
		return utils.LavaError(ctx, ctx.Logger(), "CreateEmptyProject_already_exist", map[string]string{"subscription": subAddr}, "project already exist for the current subscription with the same name")
	}

	project.AdminPolicy = projectData.GetPolicy()

	// projects can be created only by the subscription owner:
	// clone the admin policy to use as the subscription policy too.
	project.SubscriptionPolicy = project.AdminPolicy

	for _, projectKey := range projectData.GetProjectKeys() {
		err = k.registerKey(ctx, projectKey, &project, blockHeight)
		if err != nil {
			return err
		}
	}

	return k.projectsFS.AppendEntry(ctx, project.Index, blockHeight, &project)
}

func (k Keeper) registerKey(ctx sdk.Context, key types.ProjectKey, project *types.Project, blockHeight uint64) error {
	if !key.IsTypeValid() {
		return sdkerrors.ErrInvalidType
	}

	if key.IsType(types.ProjectKey_ADMIN) {
		k.addAdminKey(project, key.GetKey())
	}

	if key.IsType(types.ProjectKey_DEVELOPER) {
		// try to find the developer key
		var devkeyData types.ProtoDeveloperData
		found := k.developerKeysFS.FindEntry(ctx, key.GetKey(), blockHeight, &devkeyData)

		// if we find the developer key and it belongs to a different project, return error
		if found && devkeyData.ProjectID != project.GetIndex() {
			details := map[string]string{
				"key":      key.GetKey(),
				"keyTypes": strconv.FormatUint(uint64(key.GetKinds()), 16),
			}
			return utils.LavaError(ctx, k.Logger(ctx), "RegisterKey_key_exists", details, "key already exists")
		}

		if !found {
			err := k.addDeveloperKey(ctx, key.GetKey(), project, blockHeight)
			if err != nil {
				details := map[string]string{
					"developerKey": key.GetKey(),
					"projectIndex": project.GetIndex(),
					"blockHeight":  strconv.FormatUint(blockHeight, 10),
				}
				return utils.LavaError(ctx, k.Logger(ctx), "RegisterKey_add_dev_key_failed", details, "adding developer key failed")
			}
		}
	}

	return nil
}

func (k Keeper) addAdminKey(project *types.Project, adminKey string) {
	project.AppendKey(types.ProjectAdminKey(adminKey))
}

func (k Keeper) addDeveloperKey(ctx sdk.Context, devkey string, project *types.Project, blockHeight uint64) error {
	devkeyData := types.ProtoDeveloperData{
		ProjectID: project.GetIndex(),
	}

	err := k.developerKeysFS.AppendEntry(ctx, devkey, blockHeight, &devkeyData)
	if err != nil {
		return err
	}

	project.AppendKey(types.ProjectDeveloperKey(devkey))

	return nil
}

// Snapshot all projects of a given subscription
func (k Keeper) SnapshotSubscriptionProjects(ctx sdk.Context, subscriptionAddr string) {
	projects := k.projectsFS.GetAllEntryIndicesWithPrefix(ctx, subscriptionAddr)
	for _, projectID := range projects {
		err := k.snapshotProject(ctx, projectID)
		if err != nil {
			panic(err)
		}
	}
}

// snapshot project, create a snapshot of a project and reset the cu
func (k Keeper) snapshotProject(ctx sdk.Context, projectID string) error {
	var project types.Project
	if found := k.projectsFS.FindEntry(ctx, projectID, uint64(ctx.BlockHeight()), &project); !found {
		return utils.LavaError(ctx, ctx.Logger(), "SnapshotProject_project_not_found", map[string]string{"projectID": projectID}, "snapshot of project failed, project does not exist")
	}

	project.UsedCu = 0
	project.Snapshot += 1

	return k.projectsFS.AppendEntry(ctx, project.Index, uint64(ctx.BlockHeight()), &project)
}

func (k Keeper) DeleteProject(ctx sdk.Context, projectID string) error {
	var project types.Project
	if found := k.projectsFS.FindEntry(ctx, projectID, uint64(ctx.BlockHeight()), &project); !found {
		return utils.LavaError(ctx, ctx.Logger(), "DeleteProject_project_not_found", map[string]string{"projectID": projectID}, "project to delete was not found")
	}

	project.Enabled = false
	// TODO: delete all developer keys from the fixation

	return k.projectsFS.AppendEntry(ctx, project.Index, uint64(ctx.BlockHeight()), &project)
}
