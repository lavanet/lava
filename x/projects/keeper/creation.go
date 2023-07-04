package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/utils"
	plantypes "github.com/lavanet/lava/x/plans/types"
	"github.com/lavanet/lava/x/projects/types"
)

// Keys management logic:
//
// upon CreateProject(project):
//   -> registerKey(now)
//     -> AppendEntry(project, now)
//
// upon AddKeysToProject(project, key-by, keys-to-add)
//   -> FindEntry(project, now)
//   -> validate key-by is admin-key in project
//   -> FindEntry(project, nextEpoch)
//   -> registerKey(keys-to-add, project, nextEpoch) (see below)
//   -> AppendEntry(project, nextEpoch)
//
// upon DelKeysFromProject(project, key-by, keys-to-del)
//   -> FindEntry(project, now)
//   -> validate key-by is admin-key in project
//   -> FindEntry(project, nextEpoch)
//   -> unregisterKey(dev-keys-to-del, project, nextEpoch) (see below)
//   -> AppendEntry(project, nextEpoch)
//
// upon DeleteProject(project)
//   -> find project (now)
//   -> unregisterKey(all-keys, project, nextEpoch) (see below)
//   -> DelEntry(project, nextEpoch)
//
// upon registerKey(project, when)
//   -> if admin: add to project
//   -> if devel:
//        find devel-key (now)
//        if not belong to project: bail
//        find devel-key (when)
//        if the key not belong to this project: bail
//        else if not found: add to project, AppendEntry(dev-key, when)
//
// upon unregisterKey(project, when)
//   -> if admin: del from project
//   -> if devel:
//        find devel-key (now)
//        if not belong to project: bail
//        find devel-key (when)
//        if not found: nothing to do
//        if not belong to project: bail
//        else: del from project, DelEntry(dev-key, when)

// CreateAdminProject creates the (default) admin project
func (k Keeper) CreateAdminProject(ctx sdk.Context, subAddr string, plan plantypes.Plan) error {
	projectData := types.ProjectData{
		Name:        types.ADMIN_PROJECT_NAME,
		ProjectKeys: []types.ProjectKey{types.ProjectDeveloperKey(subAddr)},
		Enabled:     true,
		Policy:      nil,
	}
	return k.CreateProject(ctx, subAddr, projectData, plan)
}

// CreateProject adds a new project to a subscription
// (takes effect retroactively at the beginning of this epoch)
func (k Keeper) CreateProject(ctx sdk.Context, subAddr string, projectData types.ProjectData, plan plantypes.Plan) error {
	ctxBlock := uint64(ctx.BlockHeight())

	// project creation takes effect retroactively at the beginning of the current epoch
	epoch, _, err := k.epochstorageKeeper.GetEpochStartForBlock(ctx, ctxBlock)
	if err != nil {
		return utils.LavaFormatError("critical: CreateProject failed to get current epoch", err,
			utils.Attribute{Key: "index", Value: projectData.Name},
			utils.Attribute{Key: "block", Value: ctxBlock},
		)
	}

	project, err := types.NewProject(subAddr, projectData.GetName(), projectData.GetEnabled())
	if err != nil {
		return utils.LavaFormatWarning("create project failed", err,
			utils.Attribute{Key: "subscription", Value: subAddr},
		)
	}

	// project name per subscription is unique: check for duplicates
	var emptyProject types.Project
	if found := k.projectsFS.FindEntry(ctx, project.Index, epoch, &emptyProject); found {
		return utils.LavaFormatWarning("create project failed",
			fmt.Errorf("project name already exist for current subscription"),
			utils.Attribute{Key: "subscription", Value: subAddr},
		)
	}

	project.AdminPolicy = projectData.GetPolicy()

	// projects can be created only by the subscription owner:
	// clone the admin policy to use as the subscription policy too.
	project.SubscriptionPolicy = project.AdminPolicy

	for _, projectKey := range projectData.GetProjectKeys() {
		err = k.registerKey(ctx, projectKey, &project, epoch)
		if err != nil {
			return err
		}
	}

	return k.projectsFS.AppendEntry(ctx, project.Index, epoch, &project)
}

// DeleteProject deletes a project from a subscription
// (takes effect at the beginning of next epoch)
func (k Keeper) DeleteProject(ctx sdk.Context, creator string, projectID string) error {
	ctxBlock := uint64(ctx.BlockHeight())

	// project deletion takes effect at the beginning of the next epoch
	nextEpoch, err := k.epochstorageKeeper.GetNextEpoch(ctx, ctxBlock)
	if err != nil {
		return utils.LavaFormatError("critical: DeleteProject failed to get next epoch", err,
			utils.Attribute{Key: "projectID", Value: projectID},
			utils.Attribute{Key: "block", Value: ctxBlock},
		)
	}

	var project types.Project
	found := k.projectsFS.FindEntry(ctx, projectID, nextEpoch, &project)
	if !found {
		return utils.LavaFormatWarning("delete project failed",
			fmt.Errorf("project not found"),
			utils.Attribute{Key: "projectID", Value: projectID},
			utils.Attribute{Key: "block", Value: ctxBlock},
		)
	}

	if creator != project.Subscription {
		return utils.LavaFormatWarning("delete project failed",
			fmt.Errorf("creator not subscription owner"),
			utils.Attribute{Key: "creator", Value: creator},
			utils.Attribute{Key: "projectID", Value: projectID},
		)
	}

	for _, projectKey := range project.GetProjectKeys() {
		err = k.unregisterKey(ctx, projectKey, &project, nextEpoch)
		if err != nil {
			return err
		}
	}

	return k.projectsFS.DelEntry(ctx, project.Index, nextEpoch)
}

// registerKey adds a key to a project. For developer keys it also updates the
// developer key registry (that maps them to projects). The block argument is
// expected to be current block height (takes effect immediately).
func (k Keeper) registerKey(ctx sdk.Context, key types.ProjectKey, project *types.Project, block uint64) error {
	if !key.IsTypeValid() {
		return sdkerrors.ErrInvalidType
	}

	if key.IsType(types.ProjectKey_ADMIN) {
		project.AppendKey(types.ProjectAdminKey(key.Key))
	}

	if key.IsType(types.ProjectKey_DEVELOPER) {
		var devkeyData types.ProtoDeveloperData

		// we may be called with a future block (e.g. if unRegisterKey removed keys
		// earlier in the same epoch; see AddKeysToProject). in that case, realBlock
		// will report in which version the developer key was found (if at all).
		realBlock, found := k.developerKeysFS.FindEntry2(ctx, key.Key, block, &devkeyData)

		// check that the developer key is valid, and that it does not already
		// belong to a different project.
		if found && devkeyData.ProjectID != project.GetIndex() {
			return utils.LavaFormatWarning("failed to register key",
				fmt.Errorf("key already exists"),
				utils.Attribute{Key: "key", Value: key.Key},
				utils.Attribute{Key: "keyTypes", Value: key.Kinds},
			)
		}

		// by now, the key was either not found, or found and belongs to us already.
		// if the former, then we surely need to add it. if the latter, we may still
		// need to add it if realBlock != block (to cover the case where the key was
		// first removed and a future version without it was created, and re-added).

		if !found || realBlock != block {
			devkeyData := types.ProtoDeveloperData{
				ProjectID: project.GetIndex(),
			}

			err := k.developerKeysFS.AppendEntry(ctx, key.Key, block, &devkeyData)
			if err != nil {
				return utils.LavaFormatWarning("failed to register key", err,
					utils.Attribute{Key: "key", Value: key.Key},
					utils.Attribute{Key: "keyTypes", Value: key.Kinds},
				)
			}

			project.AppendKey(types.ProjectDeveloperKey(key.Key))
		}
	}

	return nil
}

// unregsiterKey removes a key from a project. For developer keys it also updates
// the developer key registry (that maps them to projects). The epoch argument is
// expected to be the next epoch start (takes effect upon next epoch).
func (k Keeper) unregisterKey(ctx sdk.Context, key types.ProjectKey, project *types.Project, epoch uint64) error {
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
		// check that the developer key belongs to the project (and remove it)
		found := project.DeleteKey(types.ProjectDeveloperKey(key.Key))
		if !found {
			return sdkerrors.ErrKeyNotFound
		}

		// and now remove it from the developer keys mapping (to projects)
		var devkeyData types.ProtoDeveloperData
		found = k.developerKeysFS.FindEntry(ctx, key.Key, epoch, &devkeyData)

		// check again that the developer key is valid, and that it belongs to the
		// project the developer key belongs to a different project.
		if !found || devkeyData.ProjectID != project.GetIndex() {
			// ... but we already checked above that it belongs to this project,
			// so this should never happen!
			return utils.LavaFormatError("critical: developer key mapped wrongly",
				fmt.Errorf("developer key included in project but mapped to another"),
				utils.Attribute{Key: "key", Value: key.Key},
				utils.Attribute{Key: "keyTypes", Value: key.Kinds},
				utils.Attribute{Key: "projectID", Value: project.GetIndex()},
				utils.Attribute{Key: "otherID", Value: devkeyData.ProjectID},
			)
		}

		err := k.developerKeysFS.DelEntry(ctx, key.Key, epoch)
		if err != nil {
			return err
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
		return utils.LavaFormatWarning("snapshot of project failed, project does not exist",
			fmt.Errorf("project not found"),
			utils.Attribute{Key: "projectID", Value: projectID},
		)
	}

	project.UsedCu = 0
	project.Snapshot += 1

	return k.projectsFS.AppendEntry(ctx, project.Index, uint64(ctx.BlockHeight()), &project)
}
