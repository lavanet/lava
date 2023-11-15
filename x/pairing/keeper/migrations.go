package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// MigrateVersion implements store migration: Create a self delegation for all providers
func (m Migrator) MigrateVersion2To3(ctx sdk.Context) error {
	chains := m.keeper.specKeeper.GetAllChainIDs(ctx)
	for _, chainID := range chains {
		storage, found := m.keeper.epochStorageKeeper.GetStakeStorageCurrent(ctx, chainID)
		if found {
			for _, entry := range storage.StakeEntries {
				// first return the providers all their coins
				addr, err := sdk.AccAddressFromBech32(entry.Address)
				if err != nil {
					return err
				}

				moduleBalance := m.keeper.bankKeeper.GetBalance(ctx, m.keeper.accountKeeper.GetModuleAddress(types.ModuleName), epochstoragetypes.TokenDenom)
				if moduleBalance.IsLT(entry.Stake) {
					return fmt.Errorf("insufficient balance to unstake %s (current balance: %s)", entry.Stake, moduleBalance)
				}
				err = m.keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, []sdk.Coin{entry.Stake})
				if err != nil {
					return fmt.Errorf("failed to send coins from module to %s: %w", addr, err)
				}

				// create self delegation
				err = m.keeper.dualstakingKeeper.Delegate(ctx, entry.Address, entry.Address, chainID, entry.Stake)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
