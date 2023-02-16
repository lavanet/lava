package common

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common/types"
)

// GetFixationEntryUniqueIndexCount get the total number of FixationEntryUniqueIndex
func GetFixationEntryUniqueIndexCount(ctx sdk.Context, storeKey sdk.StoreKey, entryKeyPrefix string) uint64 {
	store := prefix.NewStore(ctx.KVStore(storeKey), []byte{})
	byteKey := types.KeyPrefix(entryKeyPrefix + types.UniqueIndexCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil {
		return 0
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetFixationEntryUniqueIndexCount set the total number of FixationEntryUniqueIndex
func SetFixationEntryUniqueIndexCount(ctx sdk.Context, storeKey sdk.StoreKey, entryKeyPrefix string, count uint64) {
	store := prefix.NewStore(ctx.KVStore(storeKey), []byte{})
	byteKey := types.KeyPrefix(entryKeyPrefix + types.UniqueIndexCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// AppendFixationEntryUniqueIndex appends a FixationEntryUniqueIndex in the store with a new id and update the count
func AppendFixationEntryUniqueIndex(
	ctx sdk.Context,
	storeKey sdk.StoreKey,
	cdc codec.BinaryCodec,
	entryKeyPrefix string,
	fixationEntryUniqueIndex types.UniqueIndex,
) uint64 {
	// Create the FixationEntryUniqueIndex
	count := GetFixationEntryUniqueIndexCount(ctx, storeKey, entryKeyPrefix)

	// Set the ID of the appended value
	fixationEntryUniqueIndex.Id = count

	store := prefix.NewStore(ctx.KVStore(storeKey), types.KeyPrefix(entryKeyPrefix+types.UniqueIndexKey))
	appendedValue := cdc.MustMarshal(&fixationEntryUniqueIndex)
	store.Set(GetFixationEntryUniqueIndexIDBytes(fixationEntryUniqueIndex.Id), appendedValue)

	// Update FixationEntryUniqueIndex count
	SetFixationEntryUniqueIndexCount(ctx, storeKey, entryKeyPrefix, count+1)

	return count
}

// SetFixationEntryUniqueIndex set a specific FixationEntryUniqueIndex in the store
func SetFixationEntryUniqueIndex(ctx sdk.Context, storeKey sdk.StoreKey, cdc codec.BinaryCodec, fixationEntryUniqueIndex types.UniqueIndex) {
	store := prefix.NewStore(ctx.KVStore(storeKey), types.KeyPrefix(types.UniqueIndexKey))
	b := cdc.MustMarshal(&fixationEntryUniqueIndex)
	store.Set(GetFixationEntryUniqueIndexIDBytes(fixationEntryUniqueIndex.Id), b)
}

// GetFixationEntryUniqueIndex returns a FixationEntryUniqueIndex from its id
func GetFixationEntryUniqueIndex(ctx sdk.Context, storeKey sdk.StoreKey, cdc codec.BinaryCodec, entryKeyPrefix string, id uint64) (val types.UniqueIndex, found bool) {
	store := prefix.NewStore(ctx.KVStore(storeKey), types.KeyPrefix(entryKeyPrefix+types.UniqueIndexKey))
	b := store.Get(GetFixationEntryUniqueIndexIDBytes(id))
	if b == nil {
		return val, false
	}
	cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveFixationEntryUniqueIndex removes a FixationEntryUniqueIndex from the store
func RemoveFixationEntryUniqueIndex(ctx sdk.Context, storeKey sdk.StoreKey, entryKeyPrefix string, id uint64) {
	store := prefix.NewStore(ctx.KVStore(storeKey), types.KeyPrefix(entryKeyPrefix+types.UniqueIndexKey))
	store.Delete(GetFixationEntryUniqueIndexIDBytes(id))
}

// GetAllFixationEntryUniqueIndex returns all FixationEntryUniqueIndex
func GetAllFixationEntryUniqueIndex(ctx sdk.Context, storeKey sdk.StoreKey, cdc codec.BinaryCodec, entryKeyPrefix string) (list []types.UniqueIndex) {
	store := prefix.NewStore(ctx.KVStore(storeKey), types.KeyPrefix(entryKeyPrefix+types.UniqueIndexKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.UniqueIndex
		cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// GetFixationEntryUniqueIndexIDBytes returns the byte representation of the ID
func GetFixationEntryUniqueIndexIDBytes(id uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return bz
}

// GetFixationEntryUniqueIndexIDFromBytes returns ID in uint64 format from a byte array
func GetFixationEntryUniqueIndexIDFromBytes(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}
