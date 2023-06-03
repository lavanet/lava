package keeper

import (
	"fmt"

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
	ctxBlock := uint64(ctx.BlockHeight())
	if found := k.projectsFS.FindEntry(ctx, project.Index, ctxBlock, &emptyProject); found {
		// the project with the same name already exists if no error has returned
		return utils.LavaFormatWarning(
			"project already exist for the current subscription with the same name",
			fmt.Errorf("could not create project"),
			utils.Attribute{Key: "subscription", Value: subAddr},
		)
	}

	project.AdminPolicy = projectData.GetPolicy()

	// projects can be created only by the subscription owner:
	// clone the admin policy to use as the subscription policy too.
	project.SubscriptionPolicy = project.AdminPolicy

	for _, projectKey := range projectData.GetProjectKeys() {
		err = k.registerKey(ctx, projectKey, &project, ctxBlock)
		if err != nil {
			return err
		}
	}

	return k.projectsFS.AppendEntry(ctx, project.Index, ctxBlock, &project)
}

func (k Keeper) registerKey(ctx sdk.Context, key types.ProjectKey, project *types.Project, blockHeight uint64) error {
	if !key.IsTypeValid() {
		return sdkerrors.ErrInvalidType
	}

	if key.IsType(types.ProjectKey_ADMIN) {
		project.AppendKey(types.ProjectAdminKey(key.Key))
	}

	if key.IsType(types.ProjectKey_DEVELOPER) {
		ctxBlock := uint64(ctx.BlockHeight())

		// check that the developer key is valid in the current project state
		var devkeyData types.ProtoDeveloperData
		found := k.developerKeysFS.FindEntry(ctx, key.Key, ctxBlock, &devkeyData)

		// the developer key may already belong to a different project
		if found && devkeyData.ProjectID != project.GetIndex() {
			return utils.LavaFormatWarning("failed to register key",
				fmt.Errorf("key already exists"),
				utils.Attribute{Key: "key", Value: key.Key},
				utils.Attribute{Key: "keyTypes", Value: key.Kinds},
			)
		}

		// the project may have future (e.g. end of epoch) changes pending; so
		// check that the developer key is still valid in that future state
		// (for example, it could be removed and added elsewhere by then).
		found = k.developerKeysFS.FindEntry(ctx, key.Key, blockHeight, &devkeyData)

		// the developer key may already belong to a different project
		devkeyData = types.ProtoDeveloperData{}
		if found && devkeyData.ProjectID != project.GetIndex() {
			return utils.LavaFormatWarning("failed to register key",
				fmt.Errorf("key already exists in next epoch"),
				utils.Attribute{Key: "key", Value: key.Key},
				utils.Attribute{Key: "keyTypes", Value: key.Kinds},
			)
		}

		if !found {
			devkeyData := types.ProtoDeveloperData{
				ProjectID: project.GetIndex(),
			}

			err := k.developerKeysFS.AppendEntry(ctx, key.Key, blockHeight, &devkeyData)
			if err != nil {
				return utils.LavaFormatError("failed to register key", err,
					utils.Attribute{Key: "key", Value: key.Key},
					utils.Attribute{Key: "keyTypes", Value: key.Kinds},
				)
			}

			project.AppendKey(types.ProjectDeveloperKey(key.Key))
		}
	}

	return nil
}

func (k Keeper) unregisterKey(ctx sdk.Context, key types.ProjectKey, project *types.Project, blockHeight uint64) error {
	if !key.IsTypeValid() {
		return sdkerrors.ErrInvalidType
	}

	if key.IsType(types.ProjectKey_ADMIN) {
		found := project.DeleteKey(types.ProjectAdminKey(key.Key))
		if !found {
			return sdkerrors.ErrKeyNotFound
		}
	}

	if key.IsType(types.ProjectKey_DEVELOPER) {
		ctxBlock := uint64(ctx.BlockHeight())

		// check that the developer key is valid in the current project state
		var devkeyData types.ProtoDeveloperData
		found := k.developerKeysFS.FindEntry(ctx, key.Key, ctxBlock, &devkeyData)
		if !found {
			// if not found now, check if it is valid in the future (e.g. end of
			// epoch) state, as it may have been added in this epoch and pending
			// to become visible).
			found = k.developerKeysFS.FindEntry(ctx, key.Key, blockHeight, &devkeyData)
		}

		if !found {
			return sdkerrors.ErrNotFound
		}

		// the developer key belongs to a different project
		if devkeyData.ProjectID != project.GetIndex() {
			return utils.LavaFormatWarning("failed to unregister key", sdkerrors.ErrNotFound,
				utils.Attribute{Key: "key", Value: key.Key},
				utils.Attribute{Key: "keyTypes", Value: key.Kinds},
			)
		}

		err := k.developerKeysFS.DelEntry(ctx, key.Key, blockHeight)
		if err != nil {
			return err
		}
		found = project.DeleteKey(types.ProjectDeveloperKey(key.Key))
		if !found {
			panic("unregisterKey: developer key not found")
		}
	}

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
		return utils.LavaFormatWarning("snapshot of project failed, project does not exist", fmt.Errorf("project not found"),
			utils.Attribute{Key: "projectID", Value: projectID},
		)
	}

	project.UsedCu = 0
	project.Snapshot += 1

	return k.projectsFS.AppendEntry(ctx, project.Index, uint64(ctx.BlockHeight()), &project)
}

func (k Keeper) DeleteProject(ctx sdk.Context, projectID string) error {
	var project types.Project
	if found := k.projectsFS.FindEntry(ctx, projectID, uint64(ctx.BlockHeight()), &project); !found {
		return utils.LavaFormatWarning("project to delete was not found", fmt.Errorf("project not found"),
			utils.Attribute{Key: "projectID", Value: projectID},
		)
	}

	project.Enabled = false
	// TODO: delete all developer keys from the fixation

	return k.projectsFS.AppendEntry(ctx, project.Index, uint64(ctx.BlockHeight()), &project)
}
