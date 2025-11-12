package upgrades

import (
	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/lavanet/lava/v5/app/keepers"
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

var Upgrade_3_1_0 = Upgrade{
	UpgradeName:          "v3.1.0",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_3_2_0 = Upgrade{
	UpgradeName:          "v3.2.0",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_4_0_0 = Upgrade{
	UpgradeName:          "v4.0.0",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_4_1_0 = Upgrade{
	UpgradeName:          "v4.1.0",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_4_2_0 = Upgrade{
	UpgradeName:          "v4.2.0",
	CreateUpgradeHandler: v_4_2_0,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_5_0_0 = Upgrade{
	UpgradeName:          "v5.0.0",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_5_1_0 = Upgrade{
	UpgradeName:          "v5.1.0",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_5_2_0 = Upgrade{
	UpgradeName:          "v5.2.0",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_5_3_0 = Upgrade{
	UpgradeName:          "v5.3.0",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_5_4_0 = Upgrade{
	UpgradeName:          "v5.4.0",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_5_5_0 = Upgrade{
	UpgradeName:          "v5.5.0",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}
