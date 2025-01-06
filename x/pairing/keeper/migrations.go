package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	types1 "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/lavanet/lava/v4/utils/lavaslices"
	epochstoragetypes "github.com/lavanet/lava/v4/x/epochstorage/types"
	v2 "github.com/lavanet/lava/v4/x/pairing/migrations/v2"
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
	providerMap := map[string][]*epochstoragetypes.StakeEntry{}

	// map the providers
	for i := range entries {
		providerMap[entries[i].Address] = append(providerMap[entries[i].Address], &entries[i])
	}

	// iterate over each provider
	for address, entries := range providerMap {
		metadata := epochstoragetypes.ProviderMetadata{Provider: address}

		// find the biggest vault
		var biggestVault *epochstoragetypes.StakeEntry
		biggestEntry := entries[0]
		for i := range entries {
			e := entries[i]
			if biggestEntry.Stake.Amount.LT(e.Stake.Amount) {
				biggestEntry = e
			}
			if e.Vault != e.Address {
				if biggestVault == nil {
					biggestVault = e
				} else if biggestVault.Stake.Amount.LT(e.Stake.Amount) {
					biggestVault = e
				}
			}
		}

		// if no vault was found the vault is the address
		metadata.Description = biggestEntry.Description
		metadata.DelegateCommission = biggestEntry.DelegateCommission
		metadata.Provider = address
		if biggestVault != nil {
			metadata.Vault = biggestVault.Vault
		} else {
			metadata.Vault = address
		}

		// get all delegations and sum
		delegations, err := m.keeper.dualstakingKeeper.GetProviderDelegators(ctx, metadata.Provider)
		if err != nil {
			return err
		}
		bondDenom, err := m.keeper.stakingKeeper.BondDenom(ctx)
		if err != nil {
			return err
		}
		metadata.TotalDelegations = sdk.NewCoin(bondDenom, math.ZeroInt())
		for _, d := range delegations {
			if d.Delegator != metadata.Vault {
				metadata.TotalDelegations = metadata.TotalDelegations.Add(d.Amount)
			}
		}

		// fix entries with different vaults
		// count self delegations
		TotalSelfDelegation := math.ZeroInt()
		for _, e := range entries {
			if e.Vault != metadata.Vault {
				fmt.Println(address)
				biggestVault.Stake = biggestVault.Stake.SubAmount(math.NewInt(1))
				e.Stake.Amount = math.OneInt()
				e.Vault = metadata.Vault
			} else {
				TotalSelfDelegation = TotalSelfDelegation.Add(e.Stake.Amount)
			}
		}

		// calculate delegate total and update the entry
		for _, entry := range entries {
			metadata.Chains = lavaslices.AddUnique(metadata.Chains, entry.Chain)
			entry.DelegateTotal = sdk.NewCoin(bondDenom, metadata.TotalDelegations.Amount.Mul(entry.Stake.Amount).Quo(TotalSelfDelegation))
			entry.Description = types1.Description{}
			m.keeper.epochStorageKeeper.SetStakeEntryCurrent(ctx, *entry)
		}

		// set the metadata
		m.keeper.epochStorageKeeper.SetMetadata(ctx, metadata)
	}

	return nil
}
