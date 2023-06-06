package keeper

import (
	"fmt"
	"regexp"

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

	extractEpochAndChainID := func(s string) (string, string, string, bool) {
		re := regexp.MustCompile(`([A-Za-z]+)(\d+)([A-Z]+)(\d+)`)
		matches := re.FindStringSubmatch(s)
		if len(matches) > 0 {
			return matches[1], matches[2], matches[3], true
		}
		return "", "", "", false
	}

	for _, storage := range storage {
		storagetype, epoch, chainID, check := extractEpochAndChainID(storage.Index)
		fmt.Println(storage.Index, ",", storagetype, ",", epoch, ",", chainID, ",", check)
		if check {
			// handle client keys
			if storagetype == ClientKey {
				for i, entry := range storage.StakeEntries {
					if epoch == "" { // current storage only
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
			}

			// handle provider key
			if storagetype == ProviderKey {
				storage.Index = chainID + epoch // new key for epoch storage
				m.keeper.SetStakeStorage(ctx, storage)
			}
			m.keeper.RemoveStakeStorage(ctx, storage.Index)
		}
	}
	return nil
}
