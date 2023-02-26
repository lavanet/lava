package common

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common/types"
)

/*
The EntryIndex is a list which keeps all the latest version indices (e.g. indices without the version
number suffix) of fixation entries of a particular module in the KVStore.
Generally, modules can hold several types of objects that need to be fixated. To avoid a mix of indices between different
kinds of fixation entries, the module that wishes to save an entry index must save it with a prefix that
identifies it uniquely - this is the fixation key. For example, when the packages module wants to add a new
entry index of a package object to the store, it acceses the store using the key prefix:
"Entry_Index_package" (prefix = "package"). It saves the index under the key: "Entry_Index_package_<packageIndex>".
*/

// SetEntryIndex appends an entry index in the store with a new id and updates the count. It returns the index in the list of the added value (for example, if the first value of the list is added, it'll return 0 (the first index in the list))
func (vs VersionedStore) SetEntryIndex(ctx sdk.Context, index string) {
	// get the index store with the fixation key
	store := prefix.NewStore(ctx.KVStore(vs.GetStoreKey()), types.KeyPrefix(types.EntryIndexKey+vs.prefix))

	// convert the index value to a byte array
	appendedValue := []byte(index)

	// set the index in the index store
	store.Set(types.KeyPrefix(types.EntryIndexKey+vs.prefix+index), appendedValue)
}

// RemoveEntryIndex removes an EntryIndex from the store
func (vs VersionedStore) removeEntryIndex(ctx sdk.Context, index string) {
	// get the index store with the fixation key
	store := prefix.NewStore(ctx.KVStore(vs.GetStoreKey()), types.KeyPrefix(types.EntryIndexKey+vs.prefix))

	// remove the index from the store
	store.Delete(types.KeyPrefix(types.EntryIndexKey + vs.prefix + index))
}

// GetAllEntryIndex returns all EntryIndex
func (vs VersionedStore) GetAllEntryIndices(ctx sdk.Context) []string {
	// get the index store with the fixation key and init an iterator
	store := prefix.NewStore(ctx.KVStore(vs.GetStoreKey()), types.KeyPrefix(types.EntryIndexKey+vs.prefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	// iterate over the store's values and save the indices in a list
	indexList := []string{}
	for ; iterator.Valid(); iterator.Next() {
		index := string(iterator.Value())
		indexList = append(indexList, index)
	}

	return indexList
}
