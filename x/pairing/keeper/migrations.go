package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/v3/x/epochstorage/types"
	v2 "github.com/lavanet/lava/v3/x/pairing/migrations/v2"
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

func (m Migrator) MigrateVersion4To5(ctx sdk.Context) error {
	entries := m.keeper.epochStorageKeeper.GetAllStakeEntriesCurrent(ctx)
	providerMap := map[string][]epochstoragetypes.StakeEntry{}

	for _, e := range entries {
		providerMap[e.Address] = append(providerMap[e.Address], e)
	}

	for _, entries := range providerMap {
		metadata := epochstoragetypes.ProviderMetadata{}
		for i, e := range entries {
			if i == 0 {
				metadata.Provider = e.Address
				metadata.Vault = e.Vault
			}
			metadata.Chains = append(metadata.Chains, e.Chain)

			if e.Vault != metadata.Vault {
				panic("ahahahaha")
			}
		}
		delegations, err := m.keeper.dualstakingKeeper.GetProviderDelegators(ctx, metadata.Provider)
		if err != nil {
			panic("ahahahaha")
		}
		for _, d := range delegations {
			if d.Delegator != metadata.Vault {
				if metadata.TotalDelegations.Denom == "" {
					metadata.TotalDelegations = d.Amount
				} else {
					metadata.TotalDelegations = metadata.TotalDelegations.Add(d.Amount)
				}
			}
		}

		m.keeper.epochStorageKeeper.SetMetadata(ctx, metadata)
	}
	return nil
}
