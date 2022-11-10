package v5

import (
	"log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/lavanet/lava/app/keepers"
	"github.com/lavanet/lava/app/upgrades"
	"github.com/lavanet/lava/x/conflict/keeper"
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

		//delete all existing conflicts
		migrator := keeper.NewMigrator(keepers.ConflictKeeper)
		migrator.MigrateToV5(ctx)

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
