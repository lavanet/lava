package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
)

// SetPreviousSessionBlocks set previousSessionBlocks in the store
func (k Keeper) SetPreviousSessionBlocks(ctx sdk.Context, previousSessionBlocks types.PreviousSessionBlocks) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PreviousSessionBlocksKey))
	b := k.cdc.MustMarshal(&previousSessionBlocks)
	store.Set([]byte{0}, b)
}

// GetPreviousSessionBlocks returns previousSessionBlocks
func (k Keeper) GetPreviousSessionBlocks(ctx sdk.Context) (val types.PreviousSessionBlocks, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PreviousSessionBlocksKey))

	b := store.Get([]byte{0})
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemovePreviousSessionBlocks removes previousSessionBlocks from the store
func (k Keeper) RemovePreviousSessionBlocks(ctx sdk.Context) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PreviousSessionBlocksKey))
	store.Delete([]byte{0})
}
