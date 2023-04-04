package keeper

import (
	"golang.org/x/exp/slices"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate2to3 implements store migration from v2 to v3:
// For each subscription address generate the list of its projects
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	keeper := m.keeper

	for _, projectID := range keeper.projectsFS.GetAllEntryIndices(ctx) {
		// for each projectID: extract the subscription address and check its
		// list of projects; add the project to the list if not already there.
		project, err := keeper.GetProjectForBlock(ctx, projectID, uint64(ctx.BlockHeight()))
		if err != nil {
			return err
		}

		projects := keeper.GetSubscriptionProjects(ctx, project.Subscription)

		if !slices.Contains(projects, projectID) {
			projects = append(projects, projectID)
			keeper.SetSubscriptionProjects(ctx, project.Subscription, projects)
		}
	}

	return nil
}
