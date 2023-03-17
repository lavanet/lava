package upgrades

import (
	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/lavanet/lava/app/keepers"
	plansmoduletypes "github.com/lavanet/lava/x/plans/types"
	projectsmoduletypes "github.com/lavanet/lava/x/projects/types"
	subscriptionmoduletypes "github.com/lavanet/lava/x/subscription/types"
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

var Upgrade_0_4_5 = Upgrade{
	UpgradeName: "v0.4.5", // upgrade name defined few lines above
	CreateUpgradeHandler: func(m *module.Manager, c module.Configurator, bapm BaseAppParamManager, lk *keepers.LavaKeepers) upgradetypes.UpgradeHandler {
		return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			return m.RunMigrations(ctx, c, vm)
		}
	}, // create CreateUpgradeHandler in upgrades.go below
	StoreUpgrades: store.StoreUpgrades{}, // StoreUpgrades has 3 fields: Added/Renamed/Deleted any module that fits these description should be added in the way below
}

var Upgrade_0_6_0 = Upgrade{
	UpgradeName: "v0.6.0", // upgrade name defined few lines above
	CreateUpgradeHandler: func(m *module.Manager, c module.Configurator, bapm BaseAppParamManager, lk *keepers.LavaKeepers) upgradetypes.UpgradeHandler {
		return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			return m.RunMigrations(ctx, c, vm)
		}
	}, // create CreateUpgradeHandler in upgrades.go below
	StoreUpgrades: store.StoreUpgrades{}, // StoreUpgrades has 3 fields: Added/Renamed/Deleted any module that fits these description should be added in the way below
}

var Upgrade_0_6_0_RC3 = Upgrade{
	UpgradeName: "v0.6.0-RC3", // upgrade name defined few lines above
	CreateUpgradeHandler: func(m *module.Manager, c module.Configurator, bapm BaseAppParamManager, lk *keepers.LavaKeepers) upgradetypes.UpgradeHandler {
		return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			return m.RunMigrations(ctx, c, vm)
		}
	}, // create CreateUpgradeHandler in upgrades.go below
	StoreUpgrades: store.StoreUpgrades{}, // StoreUpgrades has 3 fields: Added/Renamed/Deleted any module that fits these description should be added in the way below
}

var Upgrade_0_6_1 = Upgrade{
	UpgradeName: "v0.6.1", // upgrade name defined few lines above
	CreateUpgradeHandler: func(m *module.Manager, c module.Configurator, bapm BaseAppParamManager, lk *keepers.LavaKeepers) upgradetypes.UpgradeHandler {
		return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			return m.RunMigrations(ctx, c, vm)
		}
	}, // create CreateUpgradeHandler in upgrades.go below
	StoreUpgrades: store.StoreUpgrades{}, // StoreUpgrades has 3 fields: Added/Renamed/Deleted any module that fits these description should be added in the way below
}

var Upgrade_0_7_0 = Upgrade{
	UpgradeName: "v0.7.0", // upgrade name defined few lines above
	CreateUpgradeHandler: func(m *module.Manager, c module.Configurator, bapm BaseAppParamManager, lk *keepers.LavaKeepers) upgradetypes.UpgradeHandler {
		return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			return m.RunMigrations(ctx, c, vm)
		}
	}, // create CreateUpgradeHandler in upgrades.go below
	StoreUpgrades: store.StoreUpgrades{Added: []string{plansmoduletypes.StoreKey, projectsmoduletypes.StoreKey, subscriptionmoduletypes.StoreKey}}, // StoreUpgrades has 3 fields: Added/Renamed/Deleted any module that fits these description should be added in the way below
}

var Upgrade_0_7_1 = Upgrade{
	UpgradeName: "v0.7.1", // upgrade name defined few lines above
	CreateUpgradeHandler: func(m *module.Manager, c module.Configurator, bapm BaseAppParamManager, lk *keepers.LavaKeepers) upgradetypes.UpgradeHandler {
		return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			return m.RunMigrations(ctx, c, vm)
		}
	}, // create CreateUpgradeHandler in upgrades.go below
	StoreUpgrades: store.StoreUpgrades{}, // StoreUpgrades has 3 fields: Added/Renamed/Deleted any module that fits these description should be added in the way below
}

var Upgrade_0_8_0 = Upgrade{
	UpgradeName: "v0.8.0",
	CreateUpgradeHandler: func(m *module.Manager, c module.Configurator, bapm BaseAppParamManager, lk *keepers.LavaKeepers) upgradetypes.UpgradeHandler {
		return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			return m.RunMigrations(ctx, c, vm)
		}
	},
	StoreUpgrades: store.StoreUpgrades{},
}
