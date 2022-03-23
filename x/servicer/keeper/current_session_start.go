package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
)

// SetCurrentSessionStart set currentSessionStart in the store
func (k Keeper) SetCurrentSessionStart(ctx sdk.Context, currentSessionStart types.CurrentSessionStart) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.CurrentSessionStartKey))
	b := k.cdc.MustMarshal(&currentSessionStart)
	store.Set([]byte{0}, b)
}

// GetCurrentSessionStart returns currentSessionStart
func (k Keeper) GetCurrentSessionStart(ctx sdk.Context) (val types.CurrentSessionStart, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.CurrentSessionStartKey))

	b := store.Get([]byte{0})
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveCurrentSessionStart removes currentSessionStart from the store
func (k Keeper) RemoveCurrentSessionStart(ctx sdk.Context) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.CurrentSessionStartKey))
	store.Delete([]byte{0})
}
