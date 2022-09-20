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
		allStorage := keepers.EpochstorageKeeper.GetAllStakeStorage(ctx)
		// storage.Index == storageType + strconv.FormatUint(block, 10) + chainID
		for _, storage := range allStorage {
			if storage.Index {

			}
		}

		storageType := types.ProviderKey
		allChainIDs := keepers.SpecKeeper.GetAllChainIDs(ctx)
		for _, chainID := range allChainIDs {
			tmpStorage, found := keepers.EpochstorageKeeper.GetStakeStorageCurrent(ctx, storageType, chainID)
			if !found {
				//no storage for this spec yet
				continue
			}
			newStorage := tmpStorage.Copy()
			newStorage.Index = k.stakeStorageKey(storageType, block, chainID)
			k.SetStakeStorage(ctx, newStorage)
		}

		// 3. earliest block == this epoch.

		keepers.EpochstorageKeeper.SetEarliestEpochStart(ctx, currentEpoch)

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
