package v4

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/lavanet/lava/app/keepers"
	"github.com/lavanet/lava/app/upgrades"
	"github.com/lavanet/lava/x/epochstorage/types"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	bpm upgrades.BaseAppParamManager,
	keepers *keepers.LavaKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {

		// 1. push fixations
		newParams := types.Params{
			UnstakeHoldBlocks: keepers.EpochstorageKeeper.UnstakeHoldBlocksRaw(ctx),
			EpochBlocks:       keepers.EpochstorageKeeper.EpochBlocksRaw(ctx),
			EpochsToSave:      keepers.EpochstorageKeeper.EpochsToSaveRaw(ctx),
			LatestParamChange: 0,
		}
		keepers.EpochstorageKeeper.SetParams(ctx, newParams)
		keepers.EpochstorageKeeper.PushFixatedParams(ctx, uint64(ctx.BlockHeight()), uint64(ctx.BlockHeight()))
		keepers.ConflictKeeper.PushFixations(ctx)

		// 2. get all client payments.
		currentEpoch := keepers.EpochstorageKeeper.GetEpochStart(ctx)
		currentBlock := ctx.BlockHeight()
		keepers.EpochstorageKeeper.RemoveAllEntriesPriorToBlockNumber(ctx, uint64(currentBlock), keepers.SpecKeeper.GetAllChainIDs(ctx))

		// 3. earliest block == this epoch.
		keepers.EpochstorageKeeper.SetEarliestEpochStart(ctx, currentEpoch)
		err := keepers.PairingKeeper.RemoveOldEpochPayment(ctx)
		if err != nil {
			panic("failed to remove old epoch payments killing upgrade")
		}
		return mm.RunMigrations(ctx, configurator, vm)
	}
}
