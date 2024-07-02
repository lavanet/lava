package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	v2 "github.com/lavanet/lava/x/pairing/migrations/v2"
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

// MigrateVersion3To4 initializes the new QosSum field in all ProviderConsumerEpochCu objects so it won't be nil
func (m Migrator) MigrateVersion3To4(ctx sdk.Context) error {
	pcecs := m.keeper.GetAllProviderConsumerEpochCuStore(ctx)
	for i := range pcecs {
		m.keeper.RemoveProviderConsumerEpochCu(ctx, pcecs[i].Epoch, pcecs[i].Provider, pcecs[i].Project, pcecs[i].ChainId)
		pcec := pcecs[i].ProviderConsumerEpochCu
		pcec.QosSum = math.LegacyZeroDec()
		pcec.QosAmount = 0
		m.keeper.SetProviderConsumerEpochCu(ctx, pcecs[i].Epoch, pcecs[i].Provider, pcecs[i].Project, pcecs[i].ChainId, pcec)
	}

	return nil
}
