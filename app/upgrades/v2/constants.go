package v2

import (
	store "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/lavanet/lava/app/upgrades"
)

// UpgradeName defines the on-chain upgrade name for the Osmosis v11 upgrade.
const UpgradeName = "v2"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}
