package upgrades

import (
	"github.com/cometbft/cometbft/proto/tendermint/types"
	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/lavanet/lava/v2/app/keepers"
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
