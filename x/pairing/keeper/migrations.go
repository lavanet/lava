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

// MigrateVersion3To4 fix delegation total in the stake entries
func (m Migrator) MigrateVersion3To4(ctx sdk.Context) error {
	entries := m.keeper.epochStorageKeeper.GetAllStakeEntriesCurrent(ctx)

	epoch := m.keeper.epochStorageKeeper.GetCurrentNextEpoch(ctx)
	for _, e := range entries {
		delegations, err := m.keeper.dualstakingKeeper.GetProviderDelegators(ctx, e.Address, epoch)
		if err != nil {
			return err
		}

		delegateTotal := sdk.ZeroInt()
		for _, d := range delegations {
			if e.Address == d.Delegator || e.Vault == d.Delegator || d.ChainID != e.Chain {
				continue
			}
			delegateTotal = delegateTotal.Add(d.Amount.Amount)
		}
		e.DelegateTotal.Amount = delegateTotal
		m.keeper.epochStorageKeeper.SetStakeEntryCurrent(ctx, e)
	}

	return nil
}
