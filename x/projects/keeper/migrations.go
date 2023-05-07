package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
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
