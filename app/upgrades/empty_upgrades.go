package upgrades

import (
	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	"github.com/cosmos/cosmos-sdk/x/group"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	packetforwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v7/packetforward/types"
	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	icahosttypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	"github.com/lavanet/lava/app/keepers"
	v1 "github.com/lavanet/lava/x/downtime/v1"
	dualstakingtypes "github.com/lavanet/lava/x/dualstaking/types"
	fixationtypes "github.com/lavanet/lava/x/fixationstore/types"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
	rewardstypes "github.com/lavanet/lava/x/rewards/types"
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
		lk.PairingKeeper.InitReputations(ctx, *fixationtypes.DefaultGenesis())
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

var Upgrade_0_24_0 = Upgrade{
	UpgradeName:          "v0.24.0",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_25_0 = Upgrade{
	UpgradeName:          "v0.25.0",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_25_1 = Upgrade{
	UpgradeName:          "v0.25.1",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_25_2 = Upgrade{
	UpgradeName:          "v0.25.2",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_26_0 = Upgrade{
	UpgradeName:          "v0.26.0",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_26_1 = Upgrade{
	UpgradeName:          "v0.26.1",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_26_2 = Upgrade{
	UpgradeName:          "v0.26.2",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_27_0 = Upgrade{
	UpgradeName:          "v0.27.0",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_30_0 = Upgrade{
	UpgradeName:          "v0.30.0",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_30_1 = Upgrade{
	UpgradeName:          "v0.30.1",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_30_2 = Upgrade{
	UpgradeName:          "v0.30.2",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_31_0 = Upgrade{
	UpgradeName:          "v0.31.0",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_31_1 = Upgrade{
	UpgradeName:          "v0.31.1",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_32_0 = Upgrade{
	UpgradeName:          "v0.32.0",
	CreateUpgradeHandler: v0_32_0_UpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added:   []string{rewardstypes.StoreKey},
		Deleted: []string{minttypes.StoreKey},
	},
}

var Upgrade_0_32_3 = Upgrade{
	UpgradeName:          "v0.32.3",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_33_0 = Upgrade{
	UpgradeName:          "v0.33.0",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_0_34_0 = Upgrade{
	UpgradeName:          "v0.34.0",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{icahosttypes.StoreKey, icacontrollertypes.StoreKey},
	},
}

var Upgrade_0_35_0 = Upgrade{
	UpgradeName:          "v0.35.0",
	CreateUpgradeHandler: v_35_0,
	StoreUpgrades:        store.StoreUpgrades{Added: []string{authzkeeper.StoreKey, group.StoreKey}},
}

var Upgrade_1_0_0 = Upgrade{
	UpgradeName:          "v1.0.0",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_1_0_1 = Upgrade{
	UpgradeName:          "v1.0.1",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_1_1_0 = Upgrade{
	UpgradeName:          "v1.1.0",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_1_2_0 = Upgrade{
	UpgradeName:          "v1.2.0",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_2_0_0 = Upgrade{
	UpgradeName:          "v2.0.0",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{packetforwardtypes.StoreKey},
	},
}

var Upgrade_2_1_0 = Upgrade{
	UpgradeName:          "v2.1.0",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_2_1_1 = Upgrade{
	UpgradeName:          "v2.1.1",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}

var Upgrade_2_1_3 = Upgrade{
	UpgradeName:          "v2.1.3",
	CreateUpgradeHandler: defaultUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}
