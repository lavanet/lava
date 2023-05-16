package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	v2 "github.com/lavanet/lava/x/projects/migrations/v2"
	v3 "github.com/lavanet/lava/x/projects/migrations/v3"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

func (m Migrator) migrateFixationsVersion(ctx sdk.Context) error {
	if err := m.keeper.projectsFS.MigrateVersion(ctx); err != nil {
		return fmt.Errorf("%w: projects fixation-store", err)
	}
	if err := m.keeper.developerKeysFS.MigrateVersion(ctx); err != nil {
		return fmt.Errorf("%w: developerKeys fixation-store", err)
	}
	return nil
}

// Migrate2to3 implements store migration from v2 to v3:
// - Trigger version upgrade of the projectsFS, developerKeysFS fixation stores
// - Update keys contents
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	if err := m.migrateFixationsVersion(ctx); err != nil {
		return err
	}

	projectIndices := m.keeper.projectsFS.GetAllEntryIndices(ctx)
	for _, projectIndex := range projectIndices {
		blocks := m.keeper.projectsFS.GetAllEntryVersions(ctx, projectIndex, true)
		for _, block := range blocks {
			var project_v2 v2.Project
			m.keeper.projectsFS.ReadEntry(ctx, projectIndex, block, &project_v2)

			// convert project keys from type v2.ProjectKey to types.ProjectKey
			var projectKeys_v3 []v3.ProjectKey
			for _, projectKey_v2 := range project_v2.ProjectKeys {
				projectKey_v3 := v3.ProjectKey{
					Key: projectKey_v2.Key,
				}

				for _, projectKeyType_v2 := range projectKey_v2.Types {
					if projectKeyType_v2 == v2.ProjectKey_ADMIN {
						projectKey_v3.Types = append(projectKey_v3.Types, v3.ProjectKey_ADMIN)
					} else if projectKeyType_v2 == v2.ProjectKey_DEVELOPER {
						projectKey_v3.Types = append(projectKey_v3.Types, v3.ProjectKey_DEVELOPER)
					}
				}
			}

			// convert chainPolicies from type v2.ChainPolicy to v3.ChainPolicy
			var chainPolicies_v3 []v3.ChainPolicy
			for _, chainPolicy_v2 := range project_v2.Policy.ChainPolicies {
				chainPolicies_v3 = append(chainPolicies_v3, v3.ChainPolicy{
					ChainId: chainPolicy_v2.ChainId,
					Apis:    chainPolicy_v2.Apis,
				})
			}

			// convert policy from type v2.Policy to v3.Policy
			policy_v3 := v3.Policy{
				ChainPolicies:      chainPolicies_v3,
				GeolocationProfile: project_v2.Policy.GeolocationProfile,
				TotalCuLimit:       project_v2.Policy.TotalCuLimit,
				EpochCuLimit:       project_v2.Policy.EpochCuLimit,
				MaxProvidersToPair: project_v2.Policy.MaxProvidersToPair,
			}

			// convert project from type v2.Project to v3.Project
			projectStruct_v3 := v3.Project{
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
			var developerDataStruct_v2 v2.ProtoDeveloperData
			m.keeper.developerKeysFS.ReadEntry(ctx, developerDataIndex, block, &developerDataStruct_v2)

			developerData_v3 := v3.ProtoDeveloperData{
				ProjectID: developerDataStruct_v2.ProjectID,
			}

			m.keeper.developerKeysFS.ModifyEntry(ctx, developerDataIndex, block, &developerData_v3)
		}
	}

	return nil
}

// Migrate3to4 implements store migration from v3 to v4:
// - Trigger version upgrade of the projectsFS, developerKeysFS fixation-stores
func (m Migrator) Migrate3to4(ctx sdk.Context) error {
	if err := m.migrateFixationsVersion(ctx); err != nil {
		return err
	}
	return nil
}
