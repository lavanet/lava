package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	timerstoretypes "github.com/lavanet/lava/v2/x/timerstore/types"
)

func NewKeeper(cdc codec.BinaryCodec) *Keeper {
	return &Keeper{
		cdc: cdc,
	}
}

// Keeper is the timerstore keeper. The keeper retains all the fixation stores used by modules,
// it also manages their lifecycle.
type Keeper struct {
	timerStoresBegin []*timerstoretypes.TimerStore
	timerStoresEnd   []*timerstoretypes.TimerStore
	cdc              codec.BinaryCodec
}

func (k *Keeper) NewTimerStoreBeginBlock(storeKey storetypes.StoreKey, prefix string) *timerstoretypes.TimerStore {
	ts := timerstoretypes.NewTimerStore(storeKey, k.cdc, prefix)
	k.timerStoresBegin = append(k.timerStoresBegin, ts)
	return ts
}

func (k *Keeper) NewTimerStoreEndBlock(storeKey storetypes.StoreKey, prefix string) *timerstoretypes.TimerStore {
	ts := timerstoretypes.NewTimerStore(storeKey, k.cdc, prefix)
	k.timerStoresEnd = append(k.timerStoresEnd, ts)
	return ts
}

func (k *Keeper) BeginBlock(ctx sdk.Context) {
	for _, ts := range k.timerStoresBegin {
		ts.Tick(ctx)
	}
}

func (k *Keeper) EndBlock(ctx sdk.Context) {
	for _, ts := range k.timerStoresEnd {
		ts.Tick(ctx)
	}
}
