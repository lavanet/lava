package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/fixationstore/types"
	timerstorekeeper "github.com/lavanet/lava/v2/x/timerstore/keeper"
)

func NewKeeper(cdc codec.BinaryCodec, tsKeeper *timerstorekeeper.Keeper, getStaleBlocks types.GetStaleBlocks) *Keeper {
	return &Keeper{
		cdc:            cdc,
		ts:             tsKeeper,
		getStaleBlocks: getStaleBlocks,
	}
}

// Keeper is the fixationstore keeper. The keeper retains all the fixation stores used by modules,
// it also manages their lifecycle.
type Keeper struct {
	fixationsStores []*types.FixationStore
	ts              *timerstorekeeper.Keeper
	cdc             codec.BinaryCodec
	getStaleBlocks  types.GetStaleBlocks
}

func (k *Keeper) NewFixationStore(storeKey storetypes.StoreKey, prefix string) *types.FixationStore {
	ts := k.ts.NewTimerStoreBeginBlock(storeKey, prefix)
	fs := types.NewFixationStore(storeKey, k.cdc, prefix, ts, k.getStaleBlocks)
	k.fixationsStores = append(k.fixationsStores, fs)
	return fs
}

func (k *Keeper) BeginBlock(ctx sdk.Context) {}
