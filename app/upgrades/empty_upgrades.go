package upgrades

import (
	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/lavanet/lava/app/keepers"
)

var Upgrade_0_4_0 = Upgrade{
	UpgradeName: "v0.4.0", // upgrade name defined few lines above
	CreateUpgradeHandler: func(m *module.Manager, c module.Configurator, bapm BaseAppParamManager, lk *keepers.LavaKeepers) upgradetypes.UpgradeHandler {
		return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			return m.RunMigrations(ctx, c, vm)
		}
	}, // create CreateUpgradeHandler in upgrades.go below
	StoreUpgrades: store.StoreUpgrades{}, // StoreUpgrades has 3 fields: Added/Renamed/Deleted any module that fits these description should be added in the way below
}

var Upgrade_0_4_3 = Upgrade{
	UpgradeName: "v0.4.3", // upgrade name defined few lines above
	CreateUpgradeHandler: func(m *module.Manager, c module.Configurator, bapm BaseAppParamManager, lk *keepers.LavaKeepers) upgradetypes.UpgradeHandler {
		return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			return m.RunMigrations(ctx, c, vm)
		}
	}, // create CreateUpgradeHandler in upgrades.go below
	StoreUpgrades: store.StoreUpgrades{}, // StoreUpgrades has 3 fields: Added/Renamed/Deleted any module that fits these description should be added in the way below
}

var Upgrade_0_4_4 = Upgrade{
	UpgradeName: "v0.4.4", // upgrade name defined few lines above
	CreateUpgradeHandler: func(m *module.Manager, c module.Configurator, bapm BaseAppParamManager, lk *keepers.LavaKeepers) upgradetypes.UpgradeHandler {
		return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			return m.RunMigrations(ctx, c, vm)
		}
	}, // create CreateUpgradeHandler in upgrades.go below
	StoreUpgrades: store.StoreUpgrades{}, // StoreUpgrades has 3 fields: Added/Renamed/Deleted any module that fits these description should be added in the way below
}
