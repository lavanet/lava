package upgrades

import (
	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/lavanet/lava/app/keepers"
	"github.com/lavanet/lava/common"
	v1 "github.com/lavanet/lava/x/downtime/v1"
	dualstakingtypes "github.com/lavanet/lava/x/dualstaking/types"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
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

var Upgrade_0_22_0 = Upgrade{
	UpgradeName:          "v0.22.0",
	CreateUpgradeHandler: v0_22_0_UpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

func v0_22_0_UpgradeHandler(
	m *module.Manager,
	c module.Configurator,
	bapm BaseAppParamManager,
	lk *keepers.LavaKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		lk.DowntimeKeeper.SetParams(ctx, v1.DefaultParams())
		lk.ProtocolKeeper.SetParams(ctx, protocoltypes.DefaultParams())
		return m.RunMigrations(ctx, c, vm)
	}
}

var Upgrade_0_23_0 = Upgrade{
	UpgradeName:          "v0.23.0",
	CreateUpgradeHandler: v0_23_0_UpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{Added: []string{dualstakingtypes.StoreKey}},
}

func v0_23_0_UpgradeHandler(
	m *module.Manager,
	c module.Configurator,
	bapm BaseAppParamManager,
	lk *keepers.LavaKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		lk.PairingKeeper.InitProviderQoS(ctx, *common.DefaultGenesis())
		return m.RunMigrations(ctx, c, vm)
	}
}

var Upgrade_0_23_2 = Upgrade{
	UpgradeName:          "v0.23.2",             // upgrade name
	CreateUpgradeHandler: defaultUpgradeHandler, // upgrade handler (default)
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_23_4 = Upgrade{
	UpgradeName:          "v0.23.4",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_23_5 = Upgrade{
	UpgradeName:          "v0.23.5",
	CreateUpgradeHandler: v0_23_0_UpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{Added: []string{dualstakingtypes.StoreKey}},
}
