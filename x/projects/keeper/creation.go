package keeper

import (
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/v2/utils"
	plantypes "github.com/lavanet/lava/v2/x/plans/types"
	"github.com/lavanet/lava/v2/x/projects/types"
)

// Keys management logic:
//
// upon CreateProject(project):
//   -> registerKey(this epch)
//     -> AppendEntry(project, this epch)
//
// upon AddKeysToProject(project, key-by, keys-to-add)
//   -> FindEntry(project, now)
//   -> FindEntry(project-next, epoch-next)
//   -> validate key-by is admin-key in project-next
//   -> registerKey(keys-to-add, project, epoch)
//   -> AppendEntry(project, epoch)
//   if needed:
//   -> registerKey(keys-to-add, project-next, epoch-next)
//   -> AppendEntry(project, epoch-next)
//
// upon DelKeysFromProject(project, key-by, keys-to-del)
//   -> FindEntry(project-next, epoch-next)
//   -> validate key-by is admin-key in project-next
//   -> unregisterKey(dev-keys-to-del, project-next, epoch-next)
//   -> AppendEntry(project-next, epoch-next)
//
// upon DeleteProject(project)
//   -> find project (now)
//   -> unregisterKey(all-keys, project, nextEpoch) (see below)
//   -> DelEntry(project, nextEpoch)
//
// upon registerKey(project, epoch)
//   -> if admin: add to project
//   -> if devel:
//        find devel-key (epoch)
//        if belongs to another project: bail
//        else if not already ours: add to project, AppendEntry(dev-key, epoch)
//
// upon unregisterKey(project, epoch)
//   -> if admin: del from project
//   -> if devel:
//        find devel-key (epoch-next)
//        if belongs to another project bail
//        else: if ours: del from project-next, DelEntry(dev-key, epoch-next)

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
	projects := k.GetAllProjectsForSubscription(ctx, subAddr)
	if len(projects) >= int(plan.ProjectsLimit) && plan.ProjectsLimit != 0 {
		return utils.LavaFormatWarning("CreateProject failed", fmt.Errorf("subscription already has max number of projects"),
			utils.LogAttr("plan", plan.Index),
			utils.LogAttr("plan_projects_limit", plan.ProjectsLimit),
			utils.LogAttr("sub_number_of_projects", len(projects)),
		)
	}

	ctxBlock := uint64(ctx.BlockHeight())

	if len(projectData.ProjectKeys) > types.MAX_KEYS_AMOUNT {
		return utils.LavaFormatWarning("create project failed", fmt.Errorf("max number of keys for project exceeded"),
			utils.LogAttr("project", projectData.Name),
			utils.LogAttr("block", ctxBlock),
			utils.LogAttr("project_keys_amount", len(projectData.ProjectKeys)),
			utils.LogAttr("max_keys_allowed", types.MAX_KEYS_AMOUNT),
		)
	}

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
	// we also check !isDeleted because when a new subscription is created
	// in the same epoch that a subscription from the same creator was expired,
	// we'll find the old admin project but since it's deleted we want to ignore
	// it and allow creating a new admin project
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
func (k Keeper) DeleteProject(ctx sdk.Context, creator, projectID string) error {
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
		err := k.unregisterKey(ctx, projectKey, &project, nextEpoch)
		if err != nil {
			return err
		}
	}

	return k.projectsFS.DelEntry(ctx, project.Index, nextEpoch)
}

// registerKey adds a key to a project. For developer keys it also updates the
// developer key registry (that maps them to projects). The block argument is
// expected to be current block height (takes effect immediately).
func (k Keeper) registerKey(ctx sdk.Context, key types.ProjectKey, project *types.Project, epoch uint64) error {
	if !key.IsTypeValid() {
		return legacyerrors.ErrInvalidType
	}

	if key.IsType(types.ProjectKey_ADMIN) {
		project.AppendKey(types.ProjectAdminKey(key.Key))
	}

	if key.IsType(types.ProjectKey_DEVELOPER) {
		var devkeyData types.ProtoDeveloperData
		found := k.developerKeysFS.FindEntry(ctx, key.Key, epoch, &devkeyData)

		// check that the developer key is valid, and that it does not already
		// belong to a different project.
		if found && devkeyData.ProjectID != project.Index {
			return utils.LavaFormatWarning("failed to register key",
				fmt.Errorf("key already exists"),
				utils.Attribute{Key: "key", Value: key.Key},
				utils.Attribute{Key: "keyTypes", Value: key.Kinds},
			)
		}

		// by now, the key was either not found, or found and belongs to us already.
		// if the former, then we surely need to add it.
		if !found {
			devkeyData := types.ProtoDeveloperData{
				ProjectID: project.GetIndex(),
			}

			err := k.developerKeysFS.AppendEntry(ctx, key.Key, epoch, &devkeyData)
			if err != nil {
				return utils.LavaFormatWarning("failed to register key", err,
					utils.Attribute{Key: "key", Value: key.Key},
					utils.Attribute{Key: "keyTypes", Value: key.Kinds},
				)
			}
		}

		project.AppendKey(types.ProjectDeveloperKey(key.Key))
	}

	return nil
}

// unregisterKey removes a key from a project. For developer keys it also updates
// the developer key registry (that maps them to projects). The epoch argument is
// expected to be the next epoch start (takes effect upon next epoch).
func (k Keeper) unregisterKey(ctx sdk.Context, key types.ProjectKey, project *types.Project, epoch uint64) error {
	if !key.IsTypeValid() {
		return legacyerrors.ErrInvalidType
	}

	if key.IsType(types.ProjectKey_ADMIN) {
		found := project.DeleteKey(types.ProjectAdminKey(key.Key))
		if !found {
			return legacyerrors.ErrKeyNotFound
		}
	}

	if key.IsType(types.ProjectKey_DEVELOPER) {
		// check that the developer key belongs to the project (and remove it)
		found := project.DeleteKey(types.ProjectDeveloperKey(key.Key))
		if !found {
			return legacyerrors.ErrKeyNotFound
		}

		// and now remove it from the developer keys mapping (to projects)
		var devkeyData types.ProtoDeveloperData
		found = k.developerKeysFS.FindEntry(ctx, key.Key, epoch, &devkeyData)

		if !found {
			return legacyerrors.ErrNotFound
		}

		// the developer key belongs to a different project
		if devkeyData.ProjectID != project.GetIndex() {
			return utils.LavaFormatWarning("failed to unregister key", legacyerrors.ErrNotFound,
				utils.Attribute{Key: "projectID", Value: project.Index},
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
func (k Keeper) SnapshotSubscriptionProjects(ctx sdk.Context, subscriptionAddr string, block uint64) {
	projects := k.projectsFS.GetAllEntryIndicesWithPrefix(ctx, subscriptionAddr)
	for _, projectID := range projects {
		k.snapshotProject(ctx, projectID, block)
	}
}

// snapshot project, create a snapshot of a project and reset the cu
func (k Keeper) snapshotProject(ctx sdk.Context, projectID string, block uint64) {
	var project types.Project
	entryBlock, _, _, found := k.projectsFS.FindEntryDetailed(ctx, projectID, block, &project)
	if !found {
		utils.LavaFormatError("critical: snapshot of project failed (find)", legacyerrors.ErrKeyNotFound,
			utils.Attribute{Key: "project", Value: projectID},
			utils.Attribute{Key: "block", Value: ctx.BlockHeight()},
		)

		details := map[string]string{
			"projectID": projectID,
			"block":     strconv.FormatInt(int64(block), 10),
		}
		utils.LogLavaEvent(ctx, k.Logger(ctx), types.ProjectResetFailEventName, details, "reset projects failed: unable to find project")
		return
	}

	project.UsedCu = 0
	project.Snapshot += 1

	err := k.projectsFS.AppendEntry(ctx, project.Index, block, &project)
	if err != nil {
		utils.LavaFormatError("critical: snapshot of project failed (append)", err,
			utils.Attribute{Key: "project", Value: projectID},
			utils.Attribute{Key: "block", Value: ctx.BlockHeight()},
		)

		details := map[string]string{
			"projectID": projectID,
			"block":     strconv.FormatInt(int64(block), 10),
		}
		utils.LogLavaEvent(ctx, k.Logger(ctx), types.ProjectResetFailEventName, details, "reset projects failed: unable to append project")
		return
	}

	utils.LavaFormatDebug("snapshotting project",
		utils.LogAttr("entry_block", entryBlock),
		utils.LogAttr("block", block),
		utils.LogAttr("project_id", projectID))
}
