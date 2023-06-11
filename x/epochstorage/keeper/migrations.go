package keeper

import (
	"unicode"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
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
		m.keeper.RemoveStakeStorage(ctx, storage.Index)

		// handle client keys
		if storage.Index[:len(ClientKey)] == ClientKey {
			if len(storage.Index) > len(ClientKey) && !unicode.IsNumber(rune(storage.Index[len(ClientKey)])) {
				for _, entry := range storage.StakeEntries {
					moduleBalance := m.keeper.bankKeeper.GetBalance(ctx, m.keeper.accountKeeper.GetModuleAddress(PairingModuleName), types.TokenDenom)
					if moduleBalance.IsLT(entry.Stake) {
						utils.LavaFormatError("module pairing does'nt have enough balance", nil,
							utils.Attribute{Key: "balance", Value: moduleBalance.Amount.Int64()},
							utils.Attribute{Key: "client", Value: entry.Address},
							utils.Attribute{Key: "amount", Value: entry.Stake.Amount.Int64()})
						continue
					}
					receiverAddr, err := sdk.AccAddressFromBech32(entry.Address)
					if err != nil {
						utils.LavaFormatError("error getting AccAddress", err, utils.Attribute{Key: "client", Value: entry.Address})
						continue
					}

					err = m.keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, PairingModuleName, receiverAddr, []sdk.Coin{entry.Stake})
					if err != nil {
						utils.LavaFormatError("invalid transfer coins from module to account", err,
							utils.Attribute{Key: "balance", Value: moduleBalance.Amount.Int64()},
							utils.Attribute{Key: "client", Value: entry.Address},
							utils.Attribute{Key: "amount", Value: entry.Stake.Amount.Int64()})
						continue
					}
				}
			}
		} else if storage.Index[:len(ProviderKey)] == ProviderKey { // handle provider keys
			if len(storage.Index) > len(ProviderKey) {
				storage.Index = storage.Index[len(ProviderKey):]
				m.keeper.SetStakeStorage(ctx, storage)
			}
		}
	}
	return nil
}
