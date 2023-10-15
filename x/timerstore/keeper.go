package timerstore

import (
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewKeeper(cdc codec.BinaryCodec) *Keeper {
	return &Keeper{
		cdc: cdc,
	}
}

// Keeper is the timerstore keeper. The keeper retains all the fixation stores used by modules,
// it also manages their lifecycle.
type Keeper struct {
	timerStores []*TimerStore
	cdc         codec.BinaryCodec
}

func (k *Keeper) NewTimerStore(storeKey storetypes.StoreKey, prefix string) *TimerStore {
	ts := NewTimerStore(storeKey, k.cdc, prefix)
	k.timerStores = append(k.timerStores, ts)
	return ts
}

func (k *Keeper) BeginBlock(ctx sdk.Context) {
	for _, ts := range k.timerStores {
		ts.Tick(ctx)
	}
}
