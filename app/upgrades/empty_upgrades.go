package upgrades

import (
	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/lavanet/lava/app/keepers"
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

var Upgrade_0_20_1 = Upgrade{
	UpgradeName:          "v0.20.1",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_20_2 = Upgrade{
	UpgradeName:          "v0.20.2",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_20_3 = Upgrade{
	UpgradeName:          "v0.20.3",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}
