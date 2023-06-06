package keeper

import (
	"fmt"
	"unicode"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/epochstorage/types"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate2to3 implements store migration from v2 to v3:
// - refund all clients stake
// - migrate providers to a new key
func (m Migrator) Migrate2to3(ctx sdk.Context) error {
	const PairingModuleName = "pairing"
	const ClientKey = "client"
	const ProviderKey = "provider"

	storage := m.keeper.GetAllStakeStorage(ctx)

	for _, storage := range storage {
		// handle client keys
		if storage.Index[:len(ClientKey)] == ClientKey {
			if len(storage.Index) > len(ClientKey) && unicode.IsNumber(rune(storage.Index[len(ClientKey)])) {
				for i, entry := range storage.StakeEntries {
					moduleBalance := m.keeper.bankKeeper.GetBalance(ctx, m.keeper.accountKeeper.GetModuleAddress(PairingModuleName), types.TokenDenom)
					fmt.Printf("%d / %d unstaking module %s %d have balance to account %s %d \n", i, len(storage.StakeEntries), m.keeper.accountKeeper.GetModuleAddress(PairingModuleName), moduleBalance.Amount.Int64(), entry.Address, entry.Stake.Amount.Int64())
					if moduleBalance.IsLT(entry.Stake) {
						return fmt.Errorf("invalid unstaking module %s %d doesn't have sufficient balance to account %s %d", PairingModuleName, moduleBalance.Amount.Int64(), entry.Address, entry.Stake.Amount.Int64())
					}
					receiverAddr, err := sdk.AccAddressFromBech32(entry.Address)
					if err != nil {
						return fmt.Errorf("error getting AccAddress from : %s error: %s", entry.Address, err)
					}

					err = m.keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, PairingModuleName, receiverAddr, []sdk.Coin{entry.Stake})
					if err != nil {
						return fmt.Errorf("invalid transfer coins from module, %s to account %s", err, receiverAddr)
					}
				}
			}
			m.keeper.RemoveStakeStorage(ctx, storage.Index)
		}
	}
	return nil
}
