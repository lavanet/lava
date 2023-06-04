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

	for _, storage := range storage {
		// handle client keys
		if storage.Index[:len(ClientKey)] == ClientKey {
			for _, entry := range storage.StakeEntries {
				moduleBalance := m.keeper.bankKeeper.GetBalance(ctx, m.keeper.accountKeeper.GetModuleAddress(PairingModuleName), types.TokenDenom)
				if moduleBalance.IsLT(entry.Stake) {
					return fmt.Errorf("invalid unstaking module %s doesn't have sufficient balance to account %s", PairingModuleName, entry.Address)
				}
				receiverAddr, err := sdk.AccAddressFromBech32(entry.Address)
				if err != nil {
					return fmt.Errorf("error getting AccAddress from : %s error: %s", entry.Address, err)
				}

				err = m.keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, receiverAddr, []sdk.Coin{entry.Stake})
				if err != nil {
					return fmt.Errorf("invalid transfer coins from module, %s to account %s", err, receiverAddr)
				}
			}
		}

		extractEpochAndChainID := func(s string) (string, string, bool) {
			re := regexp.MustCompile(`(\d*)(\w+)`)
			matches := re.FindStringSubmatch(s)
			if len(matches) > 0 {
				return matches[1], matches[2], true
			}
			return "", "", false
		}

		// handle provider key
		if storage.Index[:len(ProviderKey)] == ProviderKey {
			m.keeper.RemoveStakeStorage(ctx, storage.Index)
			epoch, chainID, check := extractEpochAndChainID(storage.Index)
			if check {
				storage.Index = chainID + epoch // new key for epoch storage
				m.keeper.SetStakeStorage(ctx, storage)
			}
		}
	}
	return nil
}
