package common

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common/types"
)

// FixationStore manages lists of entries with versions in the store.
// (See also documentation in common/fixation_entry.go)
//
// EntryIndex tracks the indices of every Entry stored in the FixationStore
// (one index regardless of Entry version count). Its goal is to be able to
// efficiently retrieve a list of all Entry indices that are stored. (Without
// it, we would need to iterate through all the Entries *including* all their
// versions - which may be wasteful).
//
// EntryIndex objects are stored with a distinct prefix to avoid confusion
// when iterating over Entries. See example in `x/common/fixation_entry.go`.

func (fs *FixationStore) getEntryIndexStore(ctx sdk.Context) *prefix.Store {
	store := prefix.NewStore(
		ctx.KVStore(fs.storeKey),
		types.KeyPrefix(fs.createEntryIndexStoreKey()))
	return &store
}

// setEntryIndex stores an Entry index in the store
func (fs FixationStore) setEntryIndex(ctx sdk.Context, safeIndex string) {
	types.AssertSanitizedIndex(safeIndex, fs.prefix)
	store := fs.getEntryIndexStore(ctx)
	appendedValue := []byte(safeIndex) // convert the index value to a byte array
	store.Set(types.KeyPrefix(types.EntryIndexKey+fs.prefix+safeIndex), appendedValue)
}

// removeEntryIndex removes an Entry index from the store
func (fs FixationStore) removeEntryIndex(ctx sdk.Context, safeIndex string) {
	types.AssertSanitizedIndex(safeIndex, fs.prefix)
	store := fs.getEntryIndexStore(ctx)
	store.Delete(types.KeyPrefix(fs.createEntryIndexKey(safeIndex)))
}

// GetAllEntryIndex returns all Entry indices
func (fs FixationStore) GetAllEntryIndices(ctx sdk.Context) []string {
	store := fs.getEntryIndexStore(ctx)
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	// iterate over the store's values and save the indices in a list
	indexList := []string{}
	for ; iterator.Valid(); iterator.Next() {
		safeIndex := string(iterator.Value())
		types.AssertSanitizedIndex(safeIndex, fs.prefix)
		indexList = append(indexList, types.DesanitizeIndex(safeIndex))
	}

	return indexList
}

func (fs FixationStore) createEntryIndexStoreKey() string {
	return types.EntryIndexKey + fs.prefix
}

func (fs FixationStore) createEntryIndexKey(safeIndex string) string {
	return types.EntryIndexKey + fs.prefix + safeIndex
}
