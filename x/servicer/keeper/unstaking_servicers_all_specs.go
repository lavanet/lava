package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
)

// GetUnstakingServicersAllSpecsCount get the total number of unstakingServicersAllSpecs
func (k Keeper) GetUnstakingServicersAllSpecsCount(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.UnstakingServicersAllSpecsCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil {
		return 0
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetUnstakingServicersAllSpecsCount set the total number of unstakingServicersAllSpecs
func (k Keeper) SetUnstakingServicersAllSpecsCount(ctx sdk.Context, count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.UnstakingServicersAllSpecsCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// AppendUnstakingServicersAllSpecs appends a unstakingServicersAllSpecs in the store with a new id and update the count
func (k Keeper) AppendUnstakingServicersAllSpecs(
	ctx sdk.Context,
	unstakingServicersAllSpecs types.UnstakingServicersAllSpecs,
) uint64 {
	// Create the unstakingServicersAllSpecs
	count := k.GetUnstakingServicersAllSpecsCount(ctx)

	// Set the ID of the appended value
	unstakingServicersAllSpecs.Id = count

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.UnstakingServicersAllSpecsKey))
	appendedValue := k.cdc.MustMarshal(&unstakingServicersAllSpecs)
	store.Set(GetUnstakingServicersAllSpecsIDBytes(unstakingServicersAllSpecs.Id), appendedValue)

	// Update unstakingServicersAllSpecs count
	k.SetUnstakingServicersAllSpecsCount(ctx, count+1)

	return count
}

// SetUnstakingServicersAllSpecs set a specific unstakingServicersAllSpecs in the store
func (k Keeper) SetUnstakingServicersAllSpecs(ctx sdk.Context, unstakingServicersAllSpecs types.UnstakingServicersAllSpecs) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.UnstakingServicersAllSpecsKey))
	b := k.cdc.MustMarshal(&unstakingServicersAllSpecs)
	store.Set(GetUnstakingServicersAllSpecsIDBytes(unstakingServicersAllSpecs.Id), b)
}

// GetUnstakingServicersAllSpecs returns a unstakingServicersAllSpecs from its id
func (k Keeper) GetUnstakingServicersAllSpecs(ctx sdk.Context, id uint64) (val types.UnstakingServicersAllSpecs, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.UnstakingServicersAllSpecsKey))
	b := store.Get(GetUnstakingServicersAllSpecsIDBytes(id))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveUnstakingServicersAllSpecs removes a unstakingServicersAllSpecs from the store
func (k Keeper) RemoveUnstakingServicersAllSpecs(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.UnstakingServicersAllSpecsKey))
	store.Delete(GetUnstakingServicersAllSpecsIDBytes(id))
}

// GetAllUnstakingServicersAllSpecs returns all unstakingServicersAllSpecs
func (k Keeper) GetAllUnstakingServicersAllSpecs(ctx sdk.Context) (list []types.UnstakingServicersAllSpecs) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.UnstakingServicersAllSpecsKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.UnstakingServicersAllSpecs
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// GetUnstakingServicersAllSpecsIDBytes returns the byte representation of the ID
func GetUnstakingServicersAllSpecsIDBytes(id uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return bz
}

// GetUnstakingServicersAllSpecsIDFromBytes returns ID in uint64 format from a byte array
func GetUnstakingServicersAllSpecsIDFromBytes(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}
