package keeper

import (
	"encoding/binary"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/packages/types"
)

// GetPackageUniqueIndexCount get the total number of packageUniqueIndex
func (k Keeper) GetPackageUniqueIndexCount(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.PackageUniqueIndexCountKey)
	bz := store.Get(byteKey)

	// Count doesn't exist: no element
	if bz == nil {
		return 0
	}

	// Parse bytes
	return binary.BigEndian.Uint64(bz)
}

// SetPackageUniqueIndexCount set the total number of packageUniqueIndex
func (k Keeper) SetPackageUniqueIndexCount(ctx sdk.Context, count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte{})
	byteKey := types.KeyPrefix(types.PackageUniqueIndexCountKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(byteKey, bz)
}

// AppendPackageUniqueIndex appends a packageUniqueIndex in the store with a new id and update the count
func (k Keeper) AppendPackageUniqueIndex(
	ctx sdk.Context,
	packageUniqueIndex types.PackageUniqueIndex,
) uint64 {
	// Create the packageUniqueIndex
	count := k.GetPackageUniqueIndexCount(ctx)

	// Set the ID of the appended value
	packageUniqueIndex.Id = count

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PackageUniqueIndexKey))
	appendedValue := k.cdc.MustMarshal(&packageUniqueIndex)
	store.Set(GetPackageUniqueIndexIDBytes(packageUniqueIndex.Id), appendedValue)

	// Update packageUniqueIndex count
	k.SetPackageUniqueIndexCount(ctx, count+1)

	return count
}

// SetPackageUniqueIndex set a specific packageUniqueIndex in the store
func (k Keeper) SetPackageUniqueIndex(ctx sdk.Context, packageUniqueIndex types.PackageUniqueIndex) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PackageUniqueIndexKey))
	b := k.cdc.MustMarshal(&packageUniqueIndex)
	store.Set(GetPackageUniqueIndexIDBytes(packageUniqueIndex.Id), b)
}

// GetPackageUniqueIndex returns a packageUniqueIndex from its id
func (k Keeper) GetPackageUniqueIndex(ctx sdk.Context, id uint64) (val types.PackageUniqueIndex, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PackageUniqueIndexKey))
	b := store.Get(GetPackageUniqueIndexIDBytes(id))
	if b == nil {
		return val, false
	}
	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemovePackageUniqueIndex removes a packageUniqueIndex from the store
func (k Keeper) RemovePackageUniqueIndex(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PackageUniqueIndexKey))
	store.Delete(GetPackageUniqueIndexIDBytes(id))
}

// GetAllPackageUniqueIndex returns all packageUniqueIndex
func (k Keeper) GetAllPackageUniqueIndex(ctx sdk.Context) (list []types.PackageUniqueIndex) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PackageUniqueIndexKey))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.PackageUniqueIndex
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

// GetPackageUniqueIndexIDBytes returns the byte representation of the ID
func GetPackageUniqueIndexIDBytes(id uint64) []byte {
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, id)
	return bz
}

// GetPackageUniqueIndexIDFromBytes returns ID in uint64 format from a byte array
func GetPackageUniqueIndexIDFromBytes(bz []byte) uint64 {
	return binary.BigEndian.Uint64(bz)
}
