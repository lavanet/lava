package v4

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/lavanet/lava/app/keepers"
	"github.com/lavanet/lava/app/upgrades"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	bpm upgrades.BaseAppParamManager,
	keepers *keepers.LavaKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {

		// 1. push fixations
		keepers.ConflictKeeper.PushFixations(ctx)
		// 2. get all client payments.

		// epochStart, _, err := keepers.EpochstorageKeeper.GetEpochStartForBlock(ctx, uint64(ctx.BlockHeight()))
		// if err != nil {
		// 	panic(fmt.Sprintf("failed to upgrade %s", err))
		// }
		// delete all data prior to this epoch.

		// 3. earliest block == this epoch.
		// params and fixation? latestParamChange?

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
