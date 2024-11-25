package upgrades

import (
	store "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/lavanet/lava/v4/app/keepers"
)

type BaseAppParamManager interface {
	GetConsensusParams(ctx sdk.Context) *types.ConsensusParams
	StoreConsensusParams(ctx sdk.Context, cp *types.ConsensusParams)
}

type Upgrade struct {
	// Upgrade version name.
	UpgradeName          string
	CreateUpgradeHandler func(*module.Manager, module.Configurator, BaseAppParamManager, *keepers.LavaKeepers) upgradetypes.UpgradeHandler
	StoreUpgrades        store.StoreUpgrades
}
