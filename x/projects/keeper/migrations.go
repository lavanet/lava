package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	v3 "github.com/lavanet/lava/x/projects/migrations/v3"
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
	return nil
}

// Migrate3to4 implements protobuf migration from v3 to v4:
// Now the project protobuf has admin and subscription policies
func (m Migrator) Migrate3to4(ctx sdk.Context) error {
	projectIndices := m.keeper.projectsFS.GetAllEntryIndices(ctx)
	for _, projectIndex := range projectIndices {
		blocks := m.keeper.projectsFS.GetAllEntryVersions(ctx, projectIndex, true)
		for _, block := range blocks {
			var oldProjectStruct v3.Project
			if found := m.keeper.projectsFS.FindEntry(ctx, projectIndex, block, &oldProjectStruct); !found {
				return fmt.Errorf("could not find project with index %s", projectIndex)
			}

			// convert project keys from type v3.ProjectKey to types.ProjectKey
			newProjectKeys := []types.ProjectKey{}
			for _, oldProjectKey := range oldProjectStruct.ProjectKeys {
				newProjectKey := types.ProjectKey{
					Key:   oldProjectKey.Key,
					Vrfpk: oldProjectKey.Vrfpk,
				}

				for _, oldprojectKeyType := range oldProjectKey.Types {
					if oldprojectKeyType == v3.ProjectKey_ADMIN {
						newProjectKey.Types = append(newProjectKey.Types, types.ProjectKey_ADMIN)
					} else if oldprojectKeyType == v3.ProjectKey_DEVELOPER {
						newProjectKey.Types = append(newProjectKey.Types, types.ProjectKey_DEVELOPER)
					}
				}
			}

			// convert chainPolicies from type v3.ChainPolicy to types.Policy
			var newChainPolicies []types.ChainPolicy
			for _, oldChainPolicy := range oldProjectStruct.Policy.ChainPolicies {
				newChainPolicies = append(newChainPolicies, types.ChainPolicy{
					ChainId: oldChainPolicy.ChainId,
					Apis:    oldChainPolicy.Apis,
				})
			}

			// convert policy from type v3.Policy to types.Policy
			newPolicy := types.Policy{
				ChainPolicies:      newChainPolicies,
				GeolocationProfile: oldProjectStruct.Policy.GeolocationProfile,
				TotalCuLimit:       oldProjectStruct.Policy.TotalCuLimit,
				EpochCuLimit:       oldProjectStruct.Policy.EpochCuLimit,
				MaxProvidersToPair: oldProjectStruct.Policy.MaxProvidersToPair,
			}

			// convert project from type v3.Project to types.Project
			newProjectStruct := types.Project{
				Index:              oldProjectStruct.Index,
				Subscription:       oldProjectStruct.Subscription,
				Description:        oldProjectStruct.Description,
				Enabled:            oldProjectStruct.Enabled,
				ProjectKeys:        newProjectKeys,
				AdminPolicy:        &newPolicy,
				SubscriptionPolicy: &newPolicy,
				UsedCu:             oldProjectStruct.UsedCu,
			}

			err := m.keeper.projectsFS.ModifyEntry(ctx, projectIndex, block, &newProjectStruct)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
