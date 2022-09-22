package v4

import (
	"log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/lavanet/lava/app/keepers"
	"github.com/lavanet/lava/app/upgrades"
	"github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/keeper"
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
		// 1. push fixations
		newParams := types.Params{
			UnstakeHoldBlocks: keepers.EpochstorageKeeper.UnstakeHoldBlocksRaw(ctx),
			EpochBlocks:       keepers.EpochstorageKeeper.EpochBlocksRaw(ctx),
			EpochsToSave:      keepers.EpochstorageKeeper.EpochsToSaveRaw(ctx),
			LatestParamChange: 0,
		}
		keepers.EpochstorageKeeper.SetParams(ctx, newParams)
		// keepers.EpochstorageKeeper.PushFixatedParams(ctx, uint64(ctx.BlockHeight()), uint64(ctx.BlockHeight()))
		keepers.EpochstorageKeeper.PushFixatedParams(ctx, 0, 0)
		keepers.ConflictKeeper.PushFixations(ctx)

		// 2. get all client payments. and clean previous data
		currentEpoch := keepers.EpochstorageKeeper.GetEpochStart(ctx)
		currentBlock := ctx.BlockHeight()

		// stake entries cleanup:
		keepers.EpochstorageKeeper.RemoveAllEntriesPriorToBlockNumber(ctx, uint64(currentBlock), keepers.SpecKeeper.GetAllChainIDs(ctx))

		// previous epoch payments cleanup:
		allEpochPayments := keepers.PairingKeeper.GetAllEpochPayments(ctx)
		for _, payment := range allEpochPayments {
			keepers.PairingKeeper.RemoveEpochPayments(ctx, payment.Index)
		}
		// previous UniquePaymentStorageClientProvider cleanup:
		allUniquePayments := keepers.PairingKeeper.GetAllUniquePaymentStorageClientProvider(ctx)
		for _, uniquePayment := range allUniquePayments {
			keepers.PairingKeeper.RemoveUniquePaymentStorageClientProvider(ctx, uniquePayment.Index)
		}

		// previous ClientPaymentStorage cleanup using the migrator:
		pairingMigrator := keeper.NewMigrator(keepers.PairingKeeper)
		pairingMigrator.MigrateToV4(ctx)

		// 3. earliest block == this epoch.
		keepers.EpochstorageKeeper.SetEarliestEpochStart(ctx, currentEpoch)
		return mm.RunMigrations(ctx, configurator, vm)
	}
}
