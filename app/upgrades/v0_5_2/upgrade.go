package v0_5_2

import (
	"log"

	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/lavanet/lava/app/keepers"
	"github.com/lavanet/lava/app/upgrades"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

const UpgradeName = "v0.5.2"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,           // upgrade name defined few lines above
	CreateUpgradeHandler: CreateUpgradeHandler,  // create CreateUpgradeHandler in upgrades.go below
	StoreUpgrades:        store.StoreUpgrades{}, // StoreUpgrades has 3 fields: Added/Renamed/Deleted any module that fits these description should be added in the way below
}

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

		// set the mistake in all the specs
		specs := keepers.SpecKeeper.GetAllSpec(ctx)
		for _, spec := range specs {
			spec.MinStakeClient = sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(5000000000))
			spec.MinStakeProvider = sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(500000000000))
			spec.ProvidersTypes = spectypes.Spec_dynamic
			keepers.SpecKeeper.SetSpec(ctx, spec)
		}

		// set the param unstakeHoldBlocks
		keepers.EpochstorageKeeper.SetUnstakeHoldBlocksStaticRaw(ctx, 1400)

		// we use a dedicated SET since the upgrade package doesn't have access to the paramstore, thus can't set a parameter directly
		keepers.PairingKeeper.SetRecommendedEpochNumToCollectPayment(ctx, pairingtypes.DefaultRecommendedEpochNumToCollectPayment)

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
