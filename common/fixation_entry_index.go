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
//
// EntryIndex objects are stored with the key set to the (sanitized) index,
// and value normally set to types.EntryIndexLive, or types.EntryIndexDead if
// the respective Entry is marked deleted.

func (fs *FixationStore) getEntryIndexStore(ctx sdk.Context) *prefix.Store {
	store := prefix.NewStore(
		ctx.KVStore(fs.storeKey),
		types.KeyPrefix(fs.createEntryIndexStoreKey()))
	return &store
}

// setEntryIndex stores an Entry index in the store
func (fs FixationStore) setEntryIndex(ctx sdk.Context, safeIndex types.SafeIndex, live bool) {
	types.AssertSanitizedIndex(safeIndex, fs.prefix)
	store := fs.getEntryIndexStore(ctx)
	value := types.EntryIndexLive
	if !live {
		value = types.EntryIndexDead
	}
	store.Set(types.KeyPrefix(string(safeIndex)), value)
}

// removeEntryIndex removes an Entry index from the store
func (fs FixationStore) removeEntryIndex(ctx sdk.Context, safeIndex types.SafeIndex) {
	types.AssertSanitizedIndex(safeIndex, fs.prefix)
	store := fs.getEntryIndexStore(ctx)
	store.Delete(types.KeyPrefix(string(safeIndex)))
}

// AllEntryIndicesFilter returns all Entry indices with a given prefix and filtered
// by the given filter function.
func (fs FixationStore) AllEntryIndicesFilter(ctx sdk.Context, prefix string, filter func(k, v []byte) bool) []string {
	store := fs.getEntryIndexStore(ctx)
	entryPrefix := types.KeyPrefix(prefix)
	iterator := sdk.KVStorePrefixIterator(store, entryPrefix)
	defer iterator.Close()

	// iterate over the store's values and save the indices in a list
	indexList := []string{}
	for ; iterator.Valid(); iterator.Next() {
		key, value := iterator.Key(), iterator.Value()
		safeIndex := types.SafeIndex(key)
		types.AssertSanitizedIndex(safeIndex, fs.prefix)
		if filter == nil || filter(key, value) {
			indexList = append(indexList, types.DesanitizeIndex(safeIndex))
		}
	}

	return indexList
}

// GetAllEntryIndicesWithPrefix returns all Entry indices with a given prefix
func (fs FixationStore) GetAllEntryIndicesWithPrefix(ctx sdk.Context, prefix string) []string {
	filter := func(_, v []byte) bool { return types.IsEntryIndexLive(v) }
	return fs.AllEntryIndicesFilter(ctx, prefix, filter)
}

// GetAllEntryIndices returns all Entry indices
func (fs FixationStore) GetAllEntryIndices(ctx sdk.Context) []string {
	filter := func(_, v []byte) bool { return types.IsEntryIndexLive(v) }
	return fs.AllEntryIndicesFilter(ctx, "", filter)
}

func (fs FixationStore) createEntryIndexStoreKey() string {
	return fs.prefix + types.EntryIndexPrefix
}
