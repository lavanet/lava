package fixationstore

import (
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common"
)

func NewKeeper(cdc codec.BinaryCodec) *Keeper {
	return &Keeper{
		cdc: cdc,
	}
}

// Keeper is the fixationstore keeper. The keeper retains all the fixation stores used by modules,
// it also manages their lifecycle.
type Keeper struct {
	fixationsStores []*common.FixationStore
	cdc             codec.BinaryCodec
}

func (k *Keeper) NewFixationStore(storeKey storetypes.StoreKey, prefix string) *common.FixationStore {
	fs := common.NewFixationStore(storeKey, k.cdc, prefix)
	k.fixationsStores = append(k.fixationsStores, fs)
	return fs
}

func (k *Keeper) BeginBlock(ctx sdk.Context) {
	for _, fs := range k.fixationsStores {
		fs.AdvanceBlock(ctx)
	}
}
