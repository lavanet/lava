package keeper

import (
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	commontypes "github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/utils"
	planstypes "github.com/lavanet/lava/x/plans/types"
	"github.com/lavanet/lava/x/projects/types"
)

// getProjectForBlock returns the version of a given project at a given block
func (k Keeper) getProjectForBlock(ctx sdk.Context, projectID string, block uint64) (types.Project, uint64, error) {
	var project types.Project
	block, found := k.projectsFS.FindEntry2(ctx, projectID, block, &project)
	if !found {
		return project, 0, sdkerrors.ErrNotFound
	}
	return project, block, nil
}

// GetProjectForBlock returns the version of a given project at a given block
func (k Keeper) GetProjectForBlock(ctx sdk.Context, projectID string, block uint64) (types.Project, error) {
	project, _, err := k.getProjectForBlock(ctx, projectID, block)
	if err != nil {
		return project, utils.LavaFormatWarning("failed to get project for block", err,
			utils.Attribute{Key: "project", Value: projectID},
			utils.Attribute{Key: "blockHeight", Value: block},
		)
	}
	return project, nil
}

func (k Keeper) GetProjectDeveloperData(ctx sdk.Context, developerKey string, blockHeight uint64) (types.ProtoDeveloperData, error) {
	var devkeyData types.ProtoDeveloperData
	if found := k.developerKeysFS.FindEntry(ctx, developerKey, blockHeight, &devkeyData); !found {
		return types.ProtoDeveloperData{}, fmt.Errorf("GetProjectIDForDeveloper_invalid_key: %s", developerKey)
	}
	return devkeyData, nil
}

func (k Keeper) GetProjectForDeveloper(ctx sdk.Context, developerKey string, blockHeight uint64) (proj types.Project, errRet error) {
	var project types.Project
	projectDeveloperData, err := k.GetProjectDeveloperData(ctx, developerKey, blockHeight)
	if err != nil {
		return project, err
	}

	if found := k.projectsFS.FindEntry(ctx, projectDeveloperData.ProjectID, blockHeight, &project); !found {
		return project, utils.LavaFormatWarning("the developer's project was not found", fmt.Errorf("project not found"),
			utils.Attribute{Key: "developer", Value: developerKey},
			utils.Attribute{Key: "project", Value: projectDeveloperData.ProjectID},
		)
	}

	return project, nil
}

func (k Keeper) AddKeysToProject(ctx sdk.Context, projectID string, adminKey string, projectKeys []types.ProjectKey) error {
	ctxBlock := uint64(ctx.BlockHeight())

	project, _, err := k.getProjectForBlock(ctx, projectID, ctxBlock)
	if err != nil {
		return utils.LavaFormatWarning("failed to add keys", err,
			utils.Attribute{Key: "project", Value: projectID},
			utils.Attribute{Key: "block", Value: ctxBlock},
		)
	}

	if !project.IsAdminKey(adminKey) {
		return utils.LavaFormatWarning("failed to add keys",
			fmt.Errorf("requesting key must be admin key"),
			utils.Attribute{Key: "project", Value: projectID},
			utils.Attribute{Key: "block", Value: ctxBlock},
		)
	}

	for _, projectKey := range projectKeys {
		err := k.registerKey(ctx, projectKey, &project, ctxBlock)
		if err != nil {
			return utils.LavaFormatError("failed to register key to project", err,
				utils.Attribute{Key: "project", Value: projectID},
				utils.Attribute{Key: "block", Value: ctxBlock},
			)
		}
	}

	err = k.projectsFS.AppendEntry(ctx, projectID, ctxBlock, &project)
	if err != nil {
		return utils.LavaFormatWarning("failed to add keys", err,
			utils.Attribute{Key: "project", Value: projectID},
			utils.Attribute{Key: "block", Value: ctxBlock},
		)
	}

	// If keys were deleted earlier in this epoch, then that deletion would be
	// deferred and marked in a future version of project (and the developer keys
	// mapping); Then we would add the new keys to the future version too.
	// (Note that this may bring back all the keys previously deleted, and then
	// the future version could be discarded altogether; However, there is really
	// no harm in having a version be replaced with the same version - so avoid
	// the extra complexity in detecting and handling this situation).

	nextEpoch, err := k.epochstorageKeeper.GetNextEpoch(ctx, ctxBlock)
	if err != nil {
		return utils.LavaFormatError("critical: AddKeysToProject failed to get NextEpoch", err,
			utils.Attribute{Key: "project", Value: projectID},
			utils.Attribute{Key: "block", Value: ctxBlock},
		)
	}

	projectNextEpoch, futureBlock, err := k.getProjectForBlock(ctx, projectID, nextEpoch)
	if err != nil {
		return utils.LavaFormatWarning("failed to add keys (", err,
			utils.Attribute{Key: "project", Value: projectID},
			utils.Attribute{Key: "block", Value: ctxBlock},
		)
	}

	// futureBlock is the version found nearest to (not-larger than) next epoch; If it
	// is smaller than nextEpoch (specifically, equal to ctxBlock) then we know there
	// isn't a future entry there. But if it is euqal to nextEpoch then we know there
	// exists a future version at next epoch different than the current version (must
	// have been created due to an delete key earlier in this epoch) - so add the key
	// to that future version too.

	if futureBlock == nextEpoch {
		for _, projectKey := range projectKeys {
			err := k.registerKey(ctx, projectKey, &projectNextEpoch, nextEpoch)
			if err != nil {
				return utils.LavaFormatError("failed to register key to project (future)", err,
					utils.Attribute{Key: "project", Value: projectID},
					utils.Attribute{Key: "block", Value: ctxBlock},
				)
			}
		}

		err = k.projectsFS.AppendEntry(ctx, projectID, nextEpoch, &projectNextEpoch)
		if err != nil {
			return utils.LavaFormatWarning("failed to add keys (future)", err,
				utils.Attribute{Key: "project", Value: projectID},
				utils.Attribute{Key: "block", Value: ctxBlock},
			)
		}
	}

	for _, projectKey := range projectKeys {
		details := map[string]string{
			"project": project.Index,
			"key":     projectKey.Key,
			"keytype": strconv.FormatInt(int64(projectKey.Kinds), 10),
			"block":   strconv.FormatInt(int64(ctxBlock), 10),
		}
		utils.LogLavaEvent(ctx, k.Logger(ctx), types.AddProjectKeyEventName,
			details, "key added to project")
	}

	return nil
}

func (k Keeper) DelKeysFromProject(ctx sdk.Context, projectID string, adminKey string, projectKeys []types.ProjectKey) error {
	ctxBlock := uint64(ctx.BlockHeight())

	nextEpoch, err := k.epochstorageKeeper.GetNextEpoch(ctx, ctxBlock)
	if err != nil {
		return utils.LavaFormatError("critical: DelKeysFromProject failed to get NextEpoch", err,
			utils.Attribute{Key: "project", Value: projectID},
			utils.Attribute{Key: "block", Value: ctxBlock},
		)
	}

	project, _, err := k.getProjectForBlock(ctx, projectID, ctxBlock)
	if err != nil {
		return utils.LavaFormatWarning("failed to delete keys", err,
			utils.Attribute{Key: "project", Value: projectID},
		)
	}

	projectNextEpoch, _, err := k.getProjectForBlock(ctx, projectID, nextEpoch)
	if err != nil {
		return utils.LavaFormatWarning("failed to delete keys",
			fmt.Errorf("project being deleted: %w", err),
			utils.Attribute{Key: "project", Value: projectID},
		)
	}

	// validate admin key with current state
	if !project.IsAdminKey(adminKey) {
		return utils.LavaFormatWarning("failed to delete keys",
			fmt.Errorf("requesting key must be admin key"),
			utils.Attribute{Key: "project", Value: projectID},
		)
	}

	// admin keys are deleted immediately

	for _, projectKey := range projectKeys {
		if projectKey.IsType(types.ProjectKey_ADMIN) {
			projectKey = projectKey.SetType(types.ProjectKey_ADMIN)
		} else {
			continue
		}

		err := k.unregisterKey(ctx, projectKey, &project, ctxBlock)
		if err != nil {
			return utils.LavaFormatWarning("failed to unregister key from project", err,
				utils.Attribute{Key: "project", Value: projectID},
				utils.Attribute{Key: "key", Value: projectKey.Key},
				utils.Attribute{Key: "keyTypes", Value: projectKey.Kinds},
			)
		}
	}

	err = k.projectsFS.AppendEntry(ctx, projectID, ctxBlock, &project)
	if err != nil {
		return utils.LavaFormatError("critical: failed to delete keys",
			fmt.Errorf("append entry (admin): %w", err),
			utils.Attribute{Key: "project", Value: projectID},
			utils.Attribute{Key: "block", Value: ctxBlock},
		)
	}

	// developer keys are deleted upon next epoch
	// (admin keys need also be removed from the future version, to keep them in sync)

	project = projectNextEpoch

	for _, projectKey := range projectKeys {
		err := k.unregisterKey(ctx, projectKey, &project, nextEpoch)
		if err != nil {
			return utils.LavaFormatWarning("failed to unregister key from project", err,
				utils.Attribute{Key: "project", Value: projectID},
				utils.Attribute{Key: "key", Value: projectKey.Key},
				utils.Attribute{Key: "keyTypes", Value: projectKey.Kinds},
			)
		}
	}

	err = k.projectsFS.AppendEntry(ctx, projectID, nextEpoch, &project)
	if err != nil {
		return utils.LavaFormatError("critical: failed to delete keys",
			fmt.Errorf("append entry (future): %w", err),
			utils.Attribute{Key: "project", Value: projectID},
			utils.Attribute{Key: "block", Value: ctxBlock},
		)
	}

	for _, projectKey := range projectKeys {
		details := map[string]string{
			"project": project.GetIndex(),
			"key":     projectKey.Key,
			"keytype": strconv.FormatInt(int64(projectKey.Kinds), 10),
			"block":   strconv.FormatInt(int64(ctxBlock), 10),
		}
		utils.LogLavaEvent(ctx, k.Logger(ctx), types.DelProjectKeyEventName,
			details, "key deleted from project")
	}

	return nil
}

func (k Keeper) ChargeComputeUnitsToProject(ctx sdk.Context, project types.Project, blockHeight uint64, cu uint64) (err error) {
	blocks := k.projectsFS.GetEntryVersionsRange(ctx, project.Index, blockHeight, uint64(commontypes.STALE_ENTRY_TIME))

	for _, block := range blocks {
		var proj types.Project
		k.projectsFS.ReadEntry(ctx, project.Index, block, &proj)
		if proj.Snapshot != project.Snapshot {
			break
		}
		proj.UsedCu += cu
		k.projectsFS.ModifyEntry(ctx, project.Index, block, &proj)
	}

	return nil
}

func (k Keeper) SetProjectPolicy(ctx sdk.Context, projectIDs []string, policy *planstypes.Policy, key string, setPolicyEnum types.SetPolicyEnum) error {
	ctxBlock := uint64(ctx.BlockHeight())

	nextEpoch, err := k.epochstorageKeeper.GetNextEpoch(ctx, ctxBlock)
	if err != nil {
		return utils.LavaFormatError("critical: SetProjectPolicy failed to get NextEpoch", err,
			utils.Attribute{Key: "block", Value: ctxBlock},
		)
	}

	for _, projectID := range projectIDs {
		project, _, err := k.getProjectForBlock(ctx, projectID, ctxBlock)
		if err != nil {
			return utils.LavaFormatWarning("failed to set project policy", err,
				utils.Attribute{Key: "project", Value: projectID},
			)
		}

		// it is possible that the policy was already modified in this epoch, in which
		// case changes are stored in a future (= end of epoch) policy version. Hence
		// we must start with that version when applying our change (if not, then e.g.
		// set admin policy that follows set subscription policy will lose the latter).

		projectNextEpoch, _, err := k.getProjectForBlock(ctx, projectID, nextEpoch)
		if err != nil {
			return utils.LavaFormatWarning("failed to set project policy",
				fmt.Errorf("project being deleted: %w", err),
				utils.Attribute{Key: "project", Value: projectID},
			)
		}

		switch setPolicyEnum {
		case types.SET_ADMIN_POLICY:
			// for admin policy - check if the key is an address of a project admin.
			// Note, the subscription key is also considered an admin key
			if !project.IsAdminKey(key) {
				return utils.LavaFormatWarning("failed to set project policy",
					fmt.Errorf("requesting key must be admin key"),
					utils.Attribute{Key: "project", Value: projectID},
					utils.Attribute{Key: "key", Value: key},
				)
			} else {
				projectNextEpoch.AdminPolicy = policy
			}
		case types.SET_SUBSCRIPTION_POLICY:
			// for subscription policy - check if the key is an address of the
			// project's subscription consumer
			if key != project.GetSubscription() {
				return utils.LavaFormatWarning("failed to set subscription policy",
					fmt.Errorf("requesting key must be subscription key"),
					utils.Attribute{Key: "project", Value: projectID},
					utils.Attribute{Key: "key", Value: key},
				)
			} else {
				projectNextEpoch.SubscriptionPolicy = policy
			}
		}

		err = k.projectsFS.AppendEntry(ctx, projectID, nextEpoch, &projectNextEpoch)
		if err != nil {
			return utils.LavaFormatError("critical: failed to set policy",
				fmt.Errorf("append entry: %w", err),
				utils.Attribute{Key: "project", Value: projectID},
				utils.Attribute{Key: "block", Value: ctxBlock},
			)
		}
	}

	return nil
}

func (k Keeper) GetAllProjectsForSubscription(ctx sdk.Context, subscription string) []string {
	return k.projectsFS.GetAllEntryIndicesWithPrefix(ctx, subscription)
}
