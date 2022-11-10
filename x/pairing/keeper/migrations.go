package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	v4 "github.com/lavanet/lava/x/pairing/migrations/v4"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

func (m Migrator) MigrateToV4(ctx sdk.Context) error {
	return v4.DeleteAllClientPaymentStorageEntries(ctx, m.keeper.storeKey, m.keeper.cdc)
}
