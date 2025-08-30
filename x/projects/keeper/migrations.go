package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
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

// Migrate6to7 implements store migration from v6 to v7:
// -- trigger fixation migration (v4->v5), initialize IsLatest field
func (m Migrator) Migrate6to7(ctx sdk.Context) error {
	return m.migrateFixationsVersion(ctx)
}
