package upgrades

import (
	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

type BaseAppParamManager interface {
	GetConsensusParams(ctx sdk.Context) *abci.ConsensusParams
	StoreConsensusParams(ctx sdk.Context, cp *abci.ConsensusParams)
}

type Upgrade struct {
	// Upgrade version name.
	UpgradeName          string
	CreateUpgradeHandler func(*module.Manager, module.Configurator, BaseAppParamManager) upgradetypes.UpgradeHandler
	StoreUpgrades        store.StoreUpgrades
}
