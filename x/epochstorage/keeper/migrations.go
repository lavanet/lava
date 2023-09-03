package keeper

import (
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
	const ClientKey = "client"
	const ProviderKey = "provider"

	storage := m.keeper.GetAllStakeStorage(ctx)
	for _, storage := range storage {
		// handle client keys
		if storage.Index[:len(ClientKey)] == ClientKey {
			m.keeper.RemoveStakeStorage(ctx, storage.Index)
		} else if storage.Index[:len(ProviderKey)] == ProviderKey { // handle provider keys
			if len(storage.Index) > len(ProviderKey) {
				storage.Index = storage.Index[len(ProviderKey):]
				m.keeper.SetStakeStorage(ctx, storage)
			}
		}
	}
	return nil
}

// Migrate3to4 implements store migration from v3 to v4:
// - initialize DelegateTotal, DelegateLimit, DelegateCommission
func (m Migrator) Migrate3to4(ctx sdk.Context) error {
	const ProviderKey = "provider"

	utils.LavaFormatDebug("migrate: epochstorage")

	storage := m.keeper.GetAllStakeStorage(ctx)
	for _, storage := range storage {
		if storage.Index[:len(ProviderKey)] != ProviderKey {
			utils.LavaFormatDebug("migrate: skip storage with key",
				utils.Attribute{Key: "index", Value: storage.Index})
			continue
		}
		if len(storage.Index) <= len(ProviderKey) {
			utils.LavaFormatDebug("migrate: skip storage with short key",
				utils.Attribute{Key: "index", Value: storage.Index})
			continue
		}
		utils.LavaFormatDebug("migrate: handle storage with key",
			utils.Attribute{Key: "index", Value: storage.Index})
		for i := range storage.StakeEntries {
			utils.LavaFormatDebug("  StakeEntry",
				utils.Attribute{Key: "address", Value: storage.StakeEntries[i].Address})
			storage.StakeEntries[i].DelegateTotal = sdk.NewCoin(types.TokenDenom, sdk.ZeroInt())
			storage.StakeEntries[i].DelegateLimit = sdk.NewCoin(types.TokenDenom, sdk.ZeroInt())
			storage.StakeEntries[i].DelegateCommission = 100
		}
		m.keeper.SetStakeStorage(ctx, storage)
	}
	return nil
}
