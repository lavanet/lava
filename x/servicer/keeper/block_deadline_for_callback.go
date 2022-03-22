package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
)

// SetBlockDeadlineForCallback set blockDeadlineForCallback in the store
func (k Keeper) SetBlockDeadlineForCallback(ctx sdk.Context, blockDeadlineForCallback types.BlockDeadlineForCallback) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.BlockDeadlineForCallbackKey))
	b := k.cdc.MustMarshal(&blockDeadlineForCallback)
	store.Set([]byte{0}, b)
}

// GetBlockDeadlineForCallback returns blockDeadlineForCallback
func (k Keeper) GetBlockDeadlineForCallback(ctx sdk.Context) (val types.BlockDeadlineForCallback, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.BlockDeadlineForCallbackKey))

	b := store.Get([]byte{0})
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveBlockDeadlineForCallback removes blockDeadlineForCallback from the store
func (k Keeper) RemoveBlockDeadlineForCallback(ctx sdk.Context) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.BlockDeadlineForCallbackKey))
	store.Delete([]byte{0})
}
