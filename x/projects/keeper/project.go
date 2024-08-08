package keeper

import (
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/v2/utils"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
	"github.com/lavanet/lava/v2/x/projects/types"
)

// getProjectForBlock returns the version of a given project at a given block
func (k Keeper) getProjectForBlock(ctx sdk.Context, projectID string, block uint64) (types.Project, uint64, error) {
	var project types.Project
	block, _, _, found := k.projectsFS.FindEntryDetailed(ctx, projectID, block, &project)
	if !found {
		return project, 0, legacyerrors.ErrNotFound
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

// AddKeysToProject adds keys to a project. Keys are added immediately, retroactively
// effective at the start of this epoch. The adminKey must be valid (and specifically,
// not already marked for deletion by next epoch).
func (k Keeper) AddKeysToProject(ctx sdk.Context, projectID, adminKey string, projectKeys []types.ProjectKey) error {
	ctxBlock := uint64(ctx.BlockHeight())

	epoch, _, err := k.epochstorageKeeper.GetEpochStartForBlock(ctx, ctxBlock)
	if err != nil {
		return utils.LavaFormatError("critical: AddKeysToProject failed to get EpochStart", err,
			utils.Attribute{Key: "project", Value: projectID},
			utils.Attribute{Key: "block", Value: ctxBlock},
		)
	}

	project, _, err := k.getProjectForBlock(ctx, projectID, ctxBlock)
	if err != nil {
		return utils.LavaFormatWarning("failed to add keys", err,
			utils.Attribute{Key: "project", Value: projectID},
			utils.Attribute{Key: "block", Value: ctxBlock},
		)
	}

	if len(project.ProjectKeys)+len(projectKeys) > types.MAX_KEYS_AMOUNT {
		return utils.LavaFormatWarning("failed to add keys", fmt.Errorf("max number of keys for project exceeded"),
			utils.LogAttr("project", projectID),
			utils.LogAttr("block", ctxBlock),
			utils.LogAttr("admin_key", adminKey),
			utils.LogAttr("current_project_keys_amount", len(project.ProjectKeys)),
			utils.LogAttr("keys_to_add_amount", len(projectKeys)),
			utils.LogAttr("max_keys_allowed", types.MAX_KEYS_AMOUNT),
		)
	}

	// note that realBlockNextEpoch is expected to always be at epoch boundaries (except
	// perhaps the first epoch of deploying this version, but that's ephemeral)

	nextEpoch, err := k.epochstorageKeeper.GetNextEpoch(ctx, ctxBlock)
	if err != nil {
		return utils.LavaFormatError("critical: AddKeysToProject failed to get NextEpoch", err,
			utils.Attribute{Key: "project", Value: projectID},
			utils.Attribute{Key: "block", Value: ctxBlock},
		)
	}

	projectNextEpoch, realBlockNextEpoch, err := k.getProjectForBlock(ctx, projectID, nextEpoch)
	if err != nil {
		return utils.LavaFormatWarning("failed to add keys (peek)", err,
			utils.Attribute{Key: "project", Value: projectID},
			utils.Attribute{Key: "block", Value: ctxBlock},
		)
	}

	// all checks for admin key are done respective of next epoch (because any
	// deletion earlier in this epoch thereof should be effecitive immediately
	// but would be marked there).

	if !projectNextEpoch.IsAdminKey(adminKey) {
		return utils.LavaFormatWarning("failed to add keys",
			fmt.Errorf("requesting key must be admin key"),
			utils.Attribute{Key: "project", Value: projectID},
			utils.Attribute{Key: "block", Value: ctxBlock},
		)
	}

	for _, projectKey := range projectKeys {
		err := k.registerKey(ctx, projectKey, &project, epoch)
		if err != nil {
			return utils.LavaFormatError("failed to register key to project", err,
				utils.Attribute{Key: "project", Value: projectID},
				utils.Attribute{Key: "block", Value: ctxBlock},
			)
		}
	}

	err = k.projectsFS.AppendEntry(ctx, projectID, epoch, &project)
	if err != nil {
		return utils.LavaFormatWarning("failed to add keys (append)", err,
			utils.Attribute{Key: "project", Value: projectID},
			utils.Attribute{Key: "block", Value: ctxBlock},
		)
	}

	// If some keys were deleted earlier in this epoch, then that deletion would
	// be deferred and marked in the next epoch of project (and the developer keys
	// mapping); Then we also add the new keys to the future version too.
	//
	// If an admin key is re-added in same epoch then it will be re-added in the
	// next epoch version. (Note: that version may now become the same as current
	// and discarded altogether; However, we let it be as there is really no harm
	// in having a version be replaced with the same version).
	//
	// If a developer key is re-added in same epoch it will fail, because the key is
	// still mapped to the project (as the previous delete will be setteld at the
	// beginning of next epoch).
	//
	// Use realBlockNextEpoch to test if there is a next epoch version to update. It
	// is the version found nearest to (not-larger than) next epoch, so if equal to
	// nextEpoch then we know there is a next epoch version there.

	if realBlockNextEpoch == nextEpoch {
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

// DelKeysFromProject delete keys from a project. Keys are marked for deleteion due by
// the beginning of next epoch. The adminKey must be valid (specifically, not already
// marked for deletion by next epoch).
func (k Keeper) DelKeysFromProject(ctx sdk.Context, projectID, adminKey string, projectKeys []types.ProjectKey) error {
	ctxBlock := uint64(ctx.BlockHeight())

	nextEpoch, err := k.epochstorageKeeper.GetNextEpoch(ctx, ctxBlock)
	if err != nil {
		return utils.LavaFormatError("critical: DelKeysFromProject failed to get NextEpoch", err,
			utils.Attribute{Key: "project", Value: projectID},
			utils.Attribute{Key: "block", Value: ctxBlock},
		)
	}

	projectNextEpoch, _, err := k.getProjectForBlock(ctx, projectID, nextEpoch)
	if err != nil {
		return utils.LavaFormatWarning("failed to delete keys (peek)",
			fmt.Errorf("project being deleted: %w", err),
			utils.Attribute{Key: "project", Value: projectID},
			utils.Attribute{Key: "block", Value: ctxBlock},
		)
	}

	// all checks for admin key are done respective of next epoch (because any
	// deletion earlier in this epoch thereof should be effecitive immediately
	// but would be marked there).

	if !projectNextEpoch.IsAdminKey(adminKey) {
		return utils.LavaFormatWarning("failed to delete keys",
			fmt.Errorf("requesting key must be admin key"),
			utils.Attribute{Key: "project", Value: projectID},
			utils.Attribute{Key: "block", Value: ctxBlock},
		)
	}

	for _, projectKey := range projectKeys {
		err := k.unregisterKey(ctx, projectKey, &projectNextEpoch, nextEpoch)
		if err != nil {
			return utils.LavaFormatWarning("failed to unregister key from project", err,
				utils.Attribute{Key: "project", Value: projectID},
				utils.Attribute{Key: "block", Value: ctxBlock},
			)
		}
	}

	err = k.projectsFS.AppendEntry(ctx, projectID, nextEpoch, &projectNextEpoch)
	if err != nil {
		return utils.LavaFormatError("failed to delete keys (append)", err,
			utils.Attribute{Key: "project", Value: projectID},
			utils.Attribute{Key: "block", Value: ctxBlock},
		)
	}

	for _, projectKey := range projectKeys {
		details := map[string]string{
			"project": projectNextEpoch.Index,
			"key":     projectKey.Key,
			"keytype": strconv.FormatInt(int64(projectKey.Kinds), 10),
			"block":   strconv.FormatInt(int64(ctxBlock), 10),
		}
		utils.LogLavaEvent(ctx, k.Logger(ctx), types.DelProjectKeyEventName,
			details, "key deleted from project")
	}

	return nil
}

// ChargeComputUnitsToProject charges use of CU to the project at a given block.
// Propgage the charge to subsequent versions (blocks) within the same snapshot.
func (k Keeper) ChargeComputeUnitsToProject(ctx sdk.Context, project types.Project, blockHeight, cu uint64) (err error) {
	blocks := k.projectsFS.GetEntryVersionsRange(ctx, project.Index, blockHeight, k.epochstorageKeeper.BlocksToSaveRaw(ctx))

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

// SetProjectPolicy applies new policies to project(s). The change will take effect in
// the beginning of the next epoch. The adminKey must be valid (and specifically, not
// already marked for deletion by next epoch).
func (k Keeper) SetProjectPolicy(ctx sdk.Context, projectIDs []string, policy *planstypes.Policy, key string, setPolicyEnum types.SetPolicyEnum) error {
	ctxBlock := uint64(ctx.BlockHeight())

	nextEpoch, err := k.epochstorageKeeper.GetNextEpoch(ctx, ctxBlock)
	if err != nil {
		return utils.LavaFormatError("critical: SetProjectPolicy failed to get NextEpoch", err,
			utils.Attribute{Key: "block", Value: ctxBlock},
		)
	}

	projectIDsStr := ""
	for _, projectID := range projectIDs {
		projectNextEpoch, _, err := k.getProjectForBlock(ctx, projectID, nextEpoch)
		if err != nil {
			return utils.LavaFormatWarning("failed to set project policy (peek)", err,
				utils.Attribute{Key: "project", Value: projectID},
				utils.Attribute{Key: "block", Value: ctxBlock},
			)
		}

		// all checks for admin key are done respective of next epoch (because any
		// deletion earlier in this epoch thereof should be effecitive immediately
		// but would be marked there).

		switch setPolicyEnum {
		case types.SET_ADMIN_POLICY:
			// for admin policy - check if the key is an address of a project admin
			// (note that the subscription key is also considered an admin key)
			if !projectNextEpoch.IsAdminKey(key) {
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
			if key != projectNextEpoch.GetSubscription() {
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

		projectIDsStr += projectID + ", "
	}

	details := map[string]string{
		"creator":     key,
		"project_ids": projectIDsStr,
		"policy":      policy.String(),
	}

	if setPolicyEnum == types.SET_ADMIN_POLICY {
		utils.LogLavaEvent(ctx, k.Logger(ctx), types.SetAdminPolicyEventName, details, "set admin policy successfully")
	} else if setPolicyEnum == types.SET_SUBSCRIPTION_POLICY {
		utils.LogLavaEvent(ctx, k.Logger(ctx), types.SetSubscriptionPolicyEventName, details, "set subscription policy successfully")
	}
	return nil
}

// GetAllProjectsForSubscription returns a list of all projectID for a subscription
func (k Keeper) GetAllProjectsForSubscription(ctx sdk.Context, subscription string) []string {
	return k.projectsFS.GetAllEntryIndicesWithPrefix(ctx, subscription)
}
