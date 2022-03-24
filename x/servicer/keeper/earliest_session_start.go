package keeper

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
)

// SetEarliestSessionStart set earliestSessionStart in the store
func (k Keeper) SetEarliestSessionStart(ctx sdk.Context, earliestSessionStart types.EarliestSessionStart) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.EarliestSessionStartKey))
	b := k.cdc.MustMarshal(&earliestSessionStart)
	store.Set([]byte{0}, b)
}

// GetEarliestSessionStart returns earliestSessionStart
func (k Keeper) GetEarliestSessionStart(ctx sdk.Context) (val types.EarliestSessionStart, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.EarliestSessionStartKey))

	b := store.Get([]byte{0})
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveEarliestSessionStart removes earliestSessionStart from the store
func (k Keeper) RemoveEarliestSessionStart(ctx sdk.Context) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.EarliestSessionStartKey))
	store.Delete([]byte{0})
}
