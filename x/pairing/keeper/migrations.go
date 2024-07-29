package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	v2 "github.com/lavanet/lava/x/pairing/migrations/v2"
	"github.com/lavanet/lava/x/pairing/types"
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

// MigrateVersion3To4 sets new parameters
func (m Migrator) MigrateVersion3To4(ctx sdk.Context) error {
	utils.LavaFormatInfo("migrate: pairing to set new parameters")

	m.keeper.SetParams(ctx, types.DefaultParams())
	return nil
}
