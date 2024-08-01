package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	v2 "github.com/lavanet/lava/v2/x/pairing/migrations/v2"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// MigrateVersion2To3 removes all the old payment objects (to get a fresh start for the new ones)
func (m Migrator) MigrateVersion2To3(ctx sdk.Context) error {
	v2.RemoveAllUniquePaymentStorageClientProvider(ctx, m.keeper.storeKey)
	v2.RemoveAllProviderPaymentStorage(ctx, m.keeper.storeKey)
	v2.RemoveAllEpochPayments(ctx, m.keeper.storeKey)
	return nil
}
