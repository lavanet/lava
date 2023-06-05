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
		return project, utils.LavaFormatWarning("could not get project for block", fmt.Errorf("project not found"),
			utils.Attribute{Key: "project", Value: projectID},
			utils.Attribute{Key: "blockHeight", Value: blockHeight},
		)
	}

	return project, nil
}

func (k Keeper) GetProjectDeveloperData(ctx sdk.Context, developerKey string, blockHeight uint64) (types.ProtoDeveloperData, error) {
	var projectDeveloperData types.ProtoDeveloperData
	if found := k.developerKeysFS.FindEntry(ctx, developerKey, blockHeight, &projectDeveloperData); !found {
		return types.ProtoDeveloperData{}, fmt.Errorf("GetProjectIDForDeveloper_invalid_key, the requesting key is not registered to a project, developer: %s", developerKey)
	}
	return projectDeveloperData, nil
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
	var project types.Project
	if found := k.projectsFS.FindEntry(ctx, projectID, uint64(ctx.BlockHeight()), &project); !found {
		return utils.LavaFormatWarning("could not add keys to project", fmt.Errorf("project not found"),
			utils.Attribute{Key: "project", Value: projectID},
		)
	}

	// check if the admin key is valid
	if !project.IsAdminKey(adminKey) {
		return utils.LavaFormatWarning("could not add keys to project", fmt.Errorf("the requesting key is not admin key"),
			utils.Attribute{Key: "project", Value: projectID},
		)
	}

	for _, projectKey := range projectKeys {
		err := k.registerKey(ctx, projectKey, &project, uint64(ctx.BlockHeight()))
		if err != nil {
			return utils.LavaFormatError("failed to register key to project", err,
				utils.Attribute{Key: "project", Value: projectID},
				utils.Attribute{Key: "key", Value: projectKey.Key},
				utils.Attribute{Key: "keyTypes", Value: projectKey.Kinds},
			)
		}
	}

	return k.projectsFS.AppendEntry(ctx, projectID, uint64(ctx.BlockHeight()), &project)
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

		nextEpoch, err := k.epochStorageKeeper.GetNextEpoch(ctx, uint64(ctx.BlockHeight()))
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
