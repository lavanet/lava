package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/user/types"
)

// GetUnstakingUsersAllSpecsCount get the total number of unstakingUsersAllSpecs
func (k Keeper) GetUnstakingUsersAllSpecsCount(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.UnstakingUsersAllSpecsCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil {
		return 0
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetUnstakingUsersAllSpecsCount set the total number of unstakingUsersAllSpecs
func (k Keeper) SetUnstakingUsersAllSpecsCount(ctx sdk.Context, count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.UnstakingUsersAllSpecsCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// AppendUnstakingUsersAllSpecs appends a unstakingUsersAllSpecs in the store with a new id and update the count
func (k Keeper) AppendUnstakingUsersAllSpecs(
	ctx sdk.Context,
	unstakingUsersAllSpecs types.UnstakingUsersAllSpecs,
) uint64 {
	// Create the unstakingUsersAllSpecs
	count := k.GetUnstakingUsersAllSpecsCount(ctx)

	// Set the ID of the appended value
	unstakingUsersAllSpecs.Id = count

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.UnstakingUsersAllSpecsKey))
	appendedValue := k.cdc.MustMarshal(&unstakingUsersAllSpecs)
	store.Set(GetUnstakingUsersAllSpecsIDBytes(unstakingUsersAllSpecs.Id), appendedValue)

	// Update unstakingUsersAllSpecs count
	k.SetUnstakingUsersAllSpecsCount(ctx, count+1)

	return count
}

// SetUnstakingUsersAllSpecs set a specific unstakingUsersAllSpecs in the store
func (k Keeper) SetUnstakingUsersAllSpecs(ctx sdk.Context, unstakingUsersAllSpecs types.UnstakingUsersAllSpecs) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.UnstakingUsersAllSpecsKey))
	b := k.cdc.MustMarshal(&unstakingUsersAllSpecs)
	store.Set(GetUnstakingUsersAllSpecsIDBytes(unstakingUsersAllSpecs.Id), b)
}

// GetUnstakingUsersAllSpecs returns a unstakingUsersAllSpecs from its id
func (k Keeper) GetUnstakingUsersAllSpecs(ctx sdk.Context, id uint64) (val types.UnstakingUsersAllSpecs, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.UnstakingUsersAllSpecsKey))
	b := store.Get(GetUnstakingUsersAllSpecsIDBytes(id))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveUnstakingUsersAllSpecs removes a unstakingUsersAllSpecs from the store
func (k Keeper) RemoveUnstakingUsersAllSpecs(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.UnstakingUsersAllSpecsKey))
	store.Delete(GetUnstakingUsersAllSpecsIDBytes(id))
}

// GetAllUnstakingUsersAllSpecs returns all unstakingUsersAllSpecs
func (k Keeper) GetAllUnstakingUsersAllSpecs(ctx sdk.Context) (list []types.UnstakingUsersAllSpecs) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.UnstakingUsersAllSpecsKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.UnstakingUsersAllSpecs
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// GetUnstakingUsersAllSpecsIDBytes returns the byte representation of the ID
func GetUnstakingUsersAllSpecsIDBytes(id uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return bz
}

// GetUnstakingUsersAllSpecsIDFromBytes returns ID in uint64 format from a byte array
func GetUnstakingUsersAllSpecsIDFromBytes(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}
