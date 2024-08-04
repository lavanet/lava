package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	v2 "github.com/lavanet/lava/v2/x/projects/migrations/v2"
	v3 "github.com/lavanet/lava/v2/x/projects/migrations/v3"
	v4 "github.com/lavanet/lava/v2/x/projects/migrations/v4"
	v5 "github.com/lavanet/lava/v2/x/projects/migrations/v5"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

func (m Migrator) migrateFixationsVersion(ctx sdk.Context) error {
	// This migration used to call a deprecated fixationstore function called MigrateVersionAndPrefix

	return nil
}

// Migrate2to3 implements store migration from v2 to v3:
//   - Trigger version upgrade of the projectsFS, develooperKeysFS fixation stores
//   - Update keys contents
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	if err := m.migrateFixationsVersion(ctx); err != nil {
		return err
	}

	projectIndices := m.keeper.projectsFS.AllEntryIndicesFilter(ctx, "", nil)
	for _, projectIndex := range projectIndices {
		blocks := m.keeper.projectsFS.GetAllEntryVersions(ctx, projectIndex)
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

	developerDataIndices := m.keeper.developerKeysFS.AllEntryIndicesFilter(ctx, "", nil)
	for _, developerDataIndex := range developerDataIndices {
		blocks := m.keeper.developerKeysFS.GetAllEntryVersions(ctx, developerDataIndex)
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
//   - Trigger version upgrade of the projectsFS, develooperKeysFS fixation-stores
func (m Migrator) Migrate3to4(ctx sdk.Context) error {
	if err := m.migrateFixationsVersion(ctx); err != nil {
		return err
	}
	return nil
}

// Migrate4to5 implements store migration from v4 to v5:
//   - Trigger version upgrade of the projectsFS, developerKeysFS fixation stores
//   - Update keys types (from list of types to bitmap)
func (m Migrator) Migrate4to5(ctx sdk.Context) error {
	if err := m.migrateFixationsVersion(ctx); err != nil {
		return err
	}

	projectIndices := m.keeper.projectsFS.GetAllEntryIndices(ctx)
	for _, projectIndex := range projectIndices {
		utils.LavaFormatDebug("migrate:",
			utils.Attribute{Key: "project", Value: projectIndex})

		blocks := m.keeper.projectsFS.GetAllEntryVersions(ctx, projectIndex)
		for _, block := range blocks {
			utils.LavaFormatDebug("  project:",
				utils.Attribute{Key: "block", Value: block})

			var project_v4 v4.Project
			m.keeper.projectsFS.ReadEntry(ctx, projectIndex, block, &project_v4)

			// convert project keys from type v4.ProjectKey to v5.ProjectKey
			var projectKeys_v5 []v5.ProjectKey
			for _, projectKey_v4 := range project_v4.ProjectKeys {
				utils.LavaFormatDebug("    block:",
					utils.Attribute{Key: "key", Value: projectKey_v4})

				projectKey_v5 := v5.NewProjectKey(projectKey_v4.Key, 0x0)

				for _, projectKeyType_v4 := range projectKey_v4.Types {
					if projectKeyType_v4 == v4.ProjectKey_ADMIN {
						projectKey_v5 = projectKey_v5.AddType(v5.ProjectKey_ADMIN)
					} else if projectKeyType_v4 == v4.ProjectKey_DEVELOPER {
						projectKey_v5 = projectKey_v5.AddType(v5.ProjectKey_DEVELOPER)
					}
				}

				projectKeys_v5 = append(projectKeys_v5, projectKey_v5)
			}

			// convert policy from type v4.Policy to v5.Policy
			// convert chainPolicies from type v4.ChainPolicy to v5.ChainPolicy
			var adminPolicy_v5 *v5.Policy
			if project_v4.AdminPolicy != nil {
				var adminChainPolicies_v5 []v5.ChainPolicy
				for _, chainPolicy_v4 := range project_v4.AdminPolicy.ChainPolicies {
					adminChainPolicies_v5 = append(adminChainPolicies_v5, v5.ChainPolicy{
						ChainId: chainPolicy_v4.ChainId,
						Apis:    chainPolicy_v4.Apis,
					})
				}

				adminPolicy_v5_temp := v5.Policy{
					ChainPolicies:      adminChainPolicies_v5,
					GeolocationProfile: project_v4.AdminPolicy.GeolocationProfile,
					TotalCuLimit:       project_v4.AdminPolicy.TotalCuLimit,
					EpochCuLimit:       project_v4.AdminPolicy.EpochCuLimit,
					MaxProvidersToPair: project_v4.AdminPolicy.MaxProvidersToPair,
				}

				adminPolicy_v5 = &adminPolicy_v5_temp
			}

			var subscriptionPolicy_v5 *v5.Policy
			if project_v4.SubscriptionPolicy != nil {
				var subscriptionChainPolicies_v5 []v5.ChainPolicy
				for _, chainPolicy_v4 := range project_v4.SubscriptionPolicy.ChainPolicies {
					subscriptionChainPolicies_v5 = append(subscriptionChainPolicies_v5, v5.ChainPolicy{
						ChainId: chainPolicy_v4.ChainId,
						Apis:    chainPolicy_v4.Apis,
					})
				}

				subscriptionPolicy_v5_temp := v5.Policy{
					ChainPolicies:      subscriptionChainPolicies_v5,
					GeolocationProfile: project_v4.SubscriptionPolicy.GeolocationProfile,
					TotalCuLimit:       project_v4.SubscriptionPolicy.TotalCuLimit,
					EpochCuLimit:       project_v4.SubscriptionPolicy.EpochCuLimit,
					MaxProvidersToPair: project_v4.SubscriptionPolicy.MaxProvidersToPair,
				}

				subscriptionPolicy_v5 = &subscriptionPolicy_v5_temp
			}

			// convert project from type v4.Project to v5.Project
			project_v5 := v5.Project{
				Index:              project_v4.Index,
				Subscription:       project_v4.Subscription,
				Description:        project_v4.Description,
				Enabled:            project_v4.Enabled,
				ProjectKeys:        projectKeys_v5,
				AdminPolicy:        adminPolicy_v5,
				SubscriptionPolicy: subscriptionPolicy_v5,
				UsedCu:             project_v4.UsedCu,
				Snapshot:           project_v4.Snapshot,
			}

			utils.LavaFormatDebug("  project:",
				utils.Attribute{Key: "entry_v4", Value: project_v4})
			utils.LavaFormatDebug("  project:",
				utils.Attribute{Key: "entry_v5", Value: project_v5})

			m.keeper.projectsFS.ModifyEntry(ctx, projectIndex, block, &project_v5)
		}
	}

	return nil
}

// Migrate5to6 implements store migration from v5 to v6:
// -- trigger fixation migration, deleteat and live variables
func (m Migrator) Migrate5to6(ctx sdk.Context) error {
	return m.migrateFixationsVersion(ctx)
}

// Migrate6to7 implements store migration from v6 to v7:
// -- trigger fixation migration (v4->v5), initialize IsLatest field
func (m Migrator) Migrate6to7(ctx sdk.Context) error {
	return m.migrateFixationsVersion(ctx)
}
