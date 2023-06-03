package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	commontypes "github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/projects/types"
)

func (k Keeper) GetProjectForBlock(ctx sdk.Context, projectID string, blockHeight uint64) (types.Project, error) {
	var project types.Project

	if found := k.projectsFS.FindEntry(ctx, projectID, blockHeight, &project); !found {
		return project, utils.LavaFormatWarning("failed to get project for block",
			fmt.Errorf("project or block not found"),
			utils.Attribute{Key: "project", Value: projectID},
			utils.Attribute{Key: "blockHeight", Value: blockHeight},
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

	nextEpoch, err := k.epochstorageKeeper.GetNextEpoch(ctx, ctxBlock)
	if err != nil {
		return utils.LavaFormatError("AddKeysToProject: failed to get NextEpoch", err,
			utils.Attribute{Key: "index", Value: projectID},
		)
	}

	// validate the admin key against the current project state; but make the
	// changes on the future (= end of epoch) version if already exists. This
	// ensures that we do not lose changes if there are e.g. multiple additions
	// in the same epoch.

	// validate admin key with current state
	var project types.Project
	if found := k.projectsFS.FindEntry(ctx, projectID, ctxBlock, &project); !found {
		return utils.LavaFormatWarning("failed to add keys",
			fmt.Errorf("project not found"),
			utils.Attribute{Key: "project", Value: projectID},
		)
	}

	if !project.IsAdminKey(adminKey) {
		return utils.LavaFormatWarning("failed to add keys",
			fmt.Errorf("requesting key must be admin key"),
			utils.Attribute{Key: "project", Value: projectID},
		)
	}

	// make changes on the future state
	project = types.Project{}
	if found := k.projectsFS.FindEntry(ctx, projectID, nextEpoch, &project); !found {
		return utils.LavaFormatError("failed to add keys",
			fmt.Errorf("project not found (for next epoch)"),
			utils.Attribute{Key: "project", Value: projectID},
		)
	}

	for _, projectKey := range projectKeys {
		err := k.registerKey(ctx, projectKey, &project, nextEpoch)
		if err != nil {
			return utils.LavaFormatError("failed to register key to project", err,
				utils.Attribute{Key: "project", Value: projectID},
			)
		}
	}

	return k.projectsFS.AppendEntry(ctx, projectID, nextEpoch, &project)
}

func (k Keeper) DelKeysFromProject(ctx sdk.Context, projectID string, adminKey string, projectKeys []types.ProjectKey) error {
	ctxBlock := uint64(ctx.BlockHeight())

	nextEpoch, err := k.epochstorageKeeper.GetNextEpoch(ctx, ctxBlock)
	if err != nil {
		return utils.LavaFormatError("DelKeysFromProject: failed to get NextEpoch", err,
			utils.Attribute{Key: "index", Value: projectID},
		)
	}

	// validate the admin key against the current project state; but make the
	// changes on the future (= end of epoch) version if already exists. This
	// ensures that we do not lose changes if there are e.g. multiple additions
	// in the same epoch.

	// validate admin key with current state
	var project types.Project
	if found := k.projectsFS.FindEntry(ctx, projectID, ctxBlock, &project); !found {
		return utils.LavaFormatWarning("failed to delete keys",
			fmt.Errorf("project not found"),
			utils.Attribute{Key: "project", Value: projectID},
		)
	}

	if !project.IsAdminKey(adminKey) {
		return utils.LavaFormatWarning("failed to delete keys",
			fmt.Errorf("requesting key must be admin key"),
			utils.Attribute{Key: "project", Value: projectID},
		)
	}

	project = types.Project{}
	if found := k.projectsFS.FindEntry(ctx, projectID, nextEpoch, &project); !found {
		return utils.LavaFormatWarning("failed to delete keys",
			fmt.Errorf("project not found (for next epoch)"),
			utils.Attribute{Key: "project", Value: projectID},
		)
	}

	// make changes on the future state
	for _, projectKey := range projectKeys {
		// check if the deleted key is subscription owner
		if projectKey.Key == project.Subscription {
			return utils.LavaFormatWarning("failed to delete keys",
				fmt.Errorf("subscription key may not be deleted"),
				utils.Attribute{Key: "project", Value: projectID},
			)
		}

		err := k.unregisterKey(ctx, projectKey, &project, nextEpoch)
		if err != nil {
			return utils.LavaFormatWarning("failed to unregister key to project", err,
				utils.Attribute{Key: "project", Value: projectID},
			)
		}
	}

	return k.projectsFS.AppendEntry(ctx, projectID, nextEpoch, &project)
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

func (k Keeper) SetProjectPolicy(ctx sdk.Context, projectIDs []string, policy *types.Policy, key string, setPolicyEnum types.SetPolicyEnum) error {
	for _, projectID := range projectIDs {
		project, err := k.GetProjectForBlock(ctx, projectID, uint64(ctx.BlockHeight()))
		if err != nil {
			return utils.LavaFormatWarning("could not set project policy", fmt.Errorf("project not found"),
				utils.Attribute{Key: "project", Value: projectID},
			)
		}
		// for admin policy - check if the key is an address of a project admin.
		// Note, the subscription key is also considered an admin key
		if setPolicyEnum == types.SET_ADMIN_POLICY {
			if !project.IsAdminKey(key) {
				return utils.LavaFormatWarning("could not set project policy", fmt.Errorf("the requesting key is not admin key"),
					utils.Attribute{Key: "project", Value: projectID},
					utils.Attribute{Key: "key", Value: key},
				)
			} else {
				project.AdminPolicy = policy
			}
		} else if setPolicyEnum == types.SET_SUBSCRIPTION_POLICY {
			// for subscription policy - check if the key is an address of the project's subscription consumer
			if key != project.GetSubscription() {
				return utils.LavaFormatWarning("could not set subscription policy", fmt.Errorf("the requesting key is not subscription key"),
					utils.Attribute{Key: "project", Value: projectID},
					utils.Attribute{Key: "key", Value: key},
				)
			} else {
				project.SubscriptionPolicy = policy
			}
		}

		nextEpoch, err := k.epochstorageKeeper.GetNextEpoch(ctx, uint64(ctx.BlockHeight()))
		if err != nil {
			panic("could not set policy. can't get next epoch")
		}
		err = k.projectsFS.AppendEntry(ctx, projectID, nextEpoch, &project)
		if err != nil {
			return err
		}
	}

	return nil
}

func (k Keeper) GetAllProjectsForSubscription(ctx sdk.Context, subscription string) []string {
	return k.projectsFS.GetAllEntryIndicesWithPrefix(ctx, subscription)
}
