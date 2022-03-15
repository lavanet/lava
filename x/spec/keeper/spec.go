package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/spec/types"
)

// GetSpecCount get the total number of spec
func (k Keeper) GetSpecCount(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.SpecCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil {
		return 0
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetSpecCount set the total number of spec
func (k Keeper) SetSpecCount(ctx sdk.Context, count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.SpecCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// AppendSpec appends a spec in the store with a new id and update the count
func (k Keeper) AppendSpec(
	ctx sdk.Context,
	spec types.Spec,
) uint64 {
	// Create the spec
	count := k.GetSpecCount(ctx)

	// Set the ID of the appended value
	spec.Id = count

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SpecKey))
	appendedValue := k.cdc.MustMarshal(&spec)
	store.Set(GetSpecIDBytes(spec.Id), appendedValue)

	// Update spec count
	k.SetSpecCount(ctx, count+1)

	return count
}

// SetSpec set a specific spec in the store
func (k Keeper) SetSpec(ctx sdk.Context, spec types.Spec) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SpecKey))
	b := k.cdc.MustMarshal(&spec)
	store.Set(GetSpecIDBytes(spec.Id), b)
}

// GetSpec returns a spec from its id
func (k Keeper) GetSpec(ctx sdk.Context, id uint64) (val types.Spec, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SpecKey))
	b := store.Get(GetSpecIDBytes(id))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveSpec removes a spec from the store
func (k Keeper) RemoveSpec(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SpecKey))
	store.Delete(GetSpecIDBytes(id))
}

// GetAllSpec returns all spec
func (k Keeper) GetAllSpec(ctx sdk.Context) (list []types.Spec) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SpecKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Spec
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) IsSpecFoundAndActive(ctx sdk.Context, specName string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.SpecKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Spec
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		if val.Name == specName {
			if val.Status == "enabled" {
				return true
			}
			// specs are unique, theres no reason to keep iterating
			return false
		}
	}
	return false
}

// GetSpecIDBytes returns the byte representation of the ID
func GetSpecIDBytes(id uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return bz
}

// GetSpecIDFromBytes returns ID in uint64 format from a byte array
func GetSpecIDFromBytes(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}
