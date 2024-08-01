package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	v5 "github.com/lavanet/lava/v2/x/conflict/migrations/v5"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

func (m Migrator) MigrateToV5(ctx sdk.Context) error {
	return v5.DeleteOpenConflicts(ctx, m.keeper.storeKey, m.keeper.cdc)
}
