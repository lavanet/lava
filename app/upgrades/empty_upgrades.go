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

func defaultUpgradeHandler(
	m *module.Manager,
	c module.Configurator,
	bapm BaseAppParamManager,
	lk *keepers.LavaKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		return m.RunMigrations(ctx, c, vm)
	}
}

// Template for empty/simple upgrades:
// (Note: in cosmos-sdk 0.45.11 "Renamed" did not work properly for attempt v0.8.1)
//
// var Upgrade_0_0_1 = Upgrade{
// 	UpgradeName: "v0.0.1",                          // upgrade name
// 	CreateUpgradeHandler: defaultUpgradeHandler,    // upgrade handler (default)
//  	StoreUpgrades: store.StoreUpgrades{         // store upgrades
// 		Added:   []string{newmoduletypes.StoreKey}, //   new module store to add
// 		Deleted: []string{oldmoduletypes.StoreKey}, //   old module store to delete
// 		Renamed: []store.StoreRename{               //   old/new module store to rename
// 			{OldKey: oldmoduletypes.StoreKey, NewKey: newmoduletypes.StoreKey},
// 		},
// 	},
// }

var Upgrade_0_4_0 = Upgrade{
	UpgradeName:          "v0.4.0",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_4_3 = Upgrade{
	UpgradeName:          "v0.4.3",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_4_4 = Upgrade{
	UpgradeName:          "v0.4.4",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_4_5 = Upgrade{
	UpgradeName:          "v0.4.5",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_6_0 = Upgrade{
	UpgradeName:          "v0.6.0",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_6_0_RC3 = Upgrade{
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_6_1 = Upgrade{
	UpgradeName:          "v0.6.1",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

// Upgrade_0_7_0 adds three new modules: plan, subscription, projects
var Upgrade_0_7_0 = Upgrade{
	UpgradeName:          "v0.7.0",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			plansmoduletypes.StoreKey,
			projectsmoduletypes.StoreKey,
			subscriptionmoduletypes.StoreKey,
		},
	},
}

var Upgrade_0_7_1 = Upgrade{
	UpgradeName:          "v0.7.1",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

//// Upgrade_0_8_0 canceled: module StoreKey "projects" renamed to "project" without store
//// rename/delete/add; outcome not recoverable (due to cosmos-sdk v0.45.11 bug?).
// var Upgrade_0_8_0 = Upgrade{
// 	UpgradeName:          "v0.8.0-RC1",
// 	CreateUpgradeHandler: defaultUpgradeHandler,
// 	StoreUpgrades:        store.StoreUpgrades{},
// }

// for Upgrade_0_8_1 the old StoreKey for Projects module
const oldProjectsModuleStoreKey = "projects"

// Upgrade_0_8_1 fixes StoreKey "projects"->"project": delete old and add new
var Upgrade_0_8_1 = Upgrade{
	UpgradeName:          "v0.8.1",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Deleted: []string{oldProjectsModuleStoreKey},
		Added:   []string{projectsmoduletypes.StoreKey},
	},
}

var Upgrade_0_9_2 = Upgrade{
	UpgradeName:          "v0.9.2",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_9_3 = Upgrade{
	UpgradeName:          "v0.9.3",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_9_5 = Upgrade{
	UpgradeName:          "v0.9.5",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_10_0 = Upgrade{
	UpgradeName:          "v0.10.0",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_10_1 = Upgrade{
	UpgradeName:          "v0.10.1",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}
