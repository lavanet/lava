package v0_4_6

import (
	"log"

	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/lavanet/lava/app/keepers"
	"github.com/lavanet/lava/app/upgrades"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

const UpgradeName = "v0.4.6"

var Upgrade_v0_4_6 = upgrades.Upgrade{
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

		specs := keepers.SpecKeeper.GetAllSpec(ctx)
		for _, spec := range specs {
			spec.MinStakeClient = sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(5000000000))
			spec.MinStakeProvider = sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(500000000000))
			spec.ProvidersTypes = spectypes.Spec_dynamic
			keepers.SpecKeeper.SetSpec(ctx, spec)
		}

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
