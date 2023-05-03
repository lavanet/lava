package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	v2 "github.com/lavanet/lava/x/projects/migrations/v2"
	"github.com/lavanet/lava/x/projects/types"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate2to3 implements store migration from v2 to v3:
// Trigger version upgrade of the projectsFS, develooperKeysFS fixation stores
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	if err := m.keeper.projectsFS.MigrateVersion(ctx); err != nil {
		return fmt.Errorf("%w: projects fixation-store", err)
	}
	if err := m.keeper.developerKeysFS.MigrateVersion(ctx); err != nil {
		return fmt.Errorf("%w: developerKeys fixation-store", err)
	}

	projectIndices := m.keeper.projectsFS.GetAllEntryIndices(ctx)
	for _, projectIndex := range projectIndices {
		blocks := m.keeper.projectsFS.GetAllEntryVersions(ctx, projectIndex, true)
		for _, block := range blocks {
			var project_v2 v2.ProjectV2
			m.keeper.projectsFS.ReadEntry(ctx, projectIndex, block, &project_v2)

			// convert project keys from type v2.ProjectKeyv2 to types.ProjectKey
			projectKeys_v3 := []types.ProjectKey{}
			for _, projectKey_v2 := range project_v2.ProjectKeys {
				projectKey_v3 := types.ProjectKey{
					Key: projectKey_v2.Key,
				}

				for _, projectKeyType_v2 := range projectKey_v2.Types {
					if projectKeyType_v2 == v2.ProjectKey_ADMIN {
						projectKey_v3.Types = append(projectKey_v3.Types, types.ProjectKey_ADMIN)
					} else if projectKeyType_v2 == v2.ProjectKey_DEVELOPER {
						projectKey_v3.Types = append(projectKey_v3.Types, types.ProjectKey_DEVELOPER)
					}
				}
			}

			// convert chainPolicies from type v2.ChainPolicyv2 to types.Policy
			var chainPolicies_v3 []types.ChainPolicy
			for _, chainPolicy_v2 := range project_v2.Policy.ChainPolicies {
				chainPolicies_v3 = append(chainPolicies_v3, types.ChainPolicy{
					ChainId: chainPolicy_v2.ChainId,
					Apis:    chainPolicy_v2.Apis,
				})
			}

			// convert policy from type v2.Policyv2 to types.Policy
			policy_v3 := types.Policy{
				ChainPolicies:      chainPolicies_v3,
				GeolocationProfile: project_v2.Policy.GeolocationProfile,
				TotalCuLimit:       project_v2.Policy.TotalCuLimit,
				EpochCuLimit:       project_v2.Policy.EpochCuLimit,
				MaxProvidersToPair: project_v2.Policy.MaxProvidersToPair,
			}

			// convert project from type v2.Projectv2 to types.Project
			projectStruct_v3 := types.Project{
				Index:              project_v2.Index,
				Subscription:       project_v2.Subscription,
				Description:        project_v2.Description,
				Enabled:            project_v2.Enabled,
				ProjectKeys:        projectKeys_v3,
				AdminPolicy:        &policy_v3,
				SubscriptionPolicy: &policy_v3,
				UsedCu:             project_v2.UsedCu,
			}

			m.keeper.projectsFS.ModifyEntry(ctx, projectIndex, block, &projectStruct_v3)
		}
	}

	developerDataIndices := m.keeper.developerKeysFS.GetAllEntryIndices(ctx)
	for _, developerDataIndex := range developerDataIndices {
		blocks := m.keeper.developerKeysFS.GetAllEntryVersions(ctx, developerDataIndex, true)
		for _, block := range blocks {
			var developerDataStruct_v2 v2.ProtoDeveloperDataV2
			m.keeper.developerKeysFS.ReadEntry(ctx, developerDataIndex, block, &developerDataStruct_v2)

			developerData_v3 := types.ProtoDeveloperData{
				ProjectID: developerDataStruct_v2.ProjectID,
			}

			m.keeper.developerKeysFS.ModifyEntry(ctx, developerDataIndex, block, &developerData_v3)
		}
	}

	return nil
}
