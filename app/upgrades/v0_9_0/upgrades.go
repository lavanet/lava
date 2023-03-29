package v0_9_0

import (
	"log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/lavanet/lava/app/keepers"
	"github.com/lavanet/lava/app/upgrades"
)

const (
	// ClientPaymentStorageKeyPrefix is the prefix to retrieve all ClientPaymentStorage
	ClientPaymentStorageKeyPrefix = "ClientPaymentStorage/value/"
	// UniquePaymentStorageClientProviderKeyPrefix is the prefix to retrieve all UniquePaymentStorageClientProvider
	UniquePaymentStorageClientProviderKeyPrefix = "UniquePaymentStorageClientProvider/value/"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	bpm upgrades.BaseAppParamManager,
	keepers *keepers.LavaKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		log.Println("########################")
		log.Println("#   STARTING UPGRADE   #")
		log.Println("########################")

		chainIDs := keepers.SpecKeeper.GetAllChainIDs(ctx)
		for _, chainID := range chainIDs {
			entries, err := keepers.EpochstorageKeeper.GetStakeEntryForAllProvidersEpoch(ctx, chainID, keepers.EpochstorageKeeper.GetEpochStart(ctx))
			if err != nil {
				continue
			}

			for _, entry := range *entries {
				err := keepers.PairingKeeper.FreezeProvider(ctx, entry.Address, []string{chainID}, "")
				if err != nil {
					return nil, err
				}
			}
		}

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
