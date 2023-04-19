package common

import (
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/utils"
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
	fs.assertSanitizedIndex(safeIndex)
	store := fs.getEntryIndexStore(ctx)
	appendedValue := []byte(safeIndex) // convert the index value to a byte array
	store.Set(types.KeyPrefix(types.EntryIndexKey+fs.prefix+safeIndex), appendedValue)
}

// removeEntryIndex removes an Entry index from the store
func (fs FixationStore) removeEntryIndex(ctx sdk.Context, safeIndex string) {
	fs.assertSanitizedIndex(safeIndex)
	store := fs.getEntryIndexStore(ctx)
	store.Delete(types.KeyPrefix(fs.createEntryIndexKey(safeIndex)))
}

// GetAllEntryIndexWithPrefix returns all Entry indices with a given prefix
func (fs FixationStore) GetAllEntryIndicesWithPrefix(ctx sdk.Context, prefix string) []string {
	store := fs.getEntryIndexStore(ctx)
	entryPrefix := types.KeyPrefix(fs.createEntryIndexKey(prefix))
	iterator := sdk.KVStorePrefixIterator(store, entryPrefix)
	defer iterator.Close()

	// iterate over the store's values and save the indices in a list
	indexList := []string{}
	for ; iterator.Valid(); iterator.Next() {
		safeIndex := string(iterator.Value())
		fs.assertSanitizedIndex(safeIndex)
		indexList = append(indexList, desanitizeIndex(safeIndex))
	}

	return indexList
}

// GetAllEntryIndices returns all Entry indices
func (fs FixationStore) GetAllEntryIndices(ctx sdk.Context) []string {
	return fs.GetAllEntryIndicesWithPrefix(ctx, "")
}

// GetAllEntryVersions returns a list of all versions (blocks) of an entry.
// If stale == true, then the output will include stale versions (for testing).
func (fs *FixationStore) GetAllEntryVersions(ctx sdk.Context, index string, stale bool) (blocks []uint64) {
	safeIndex, err := sanitizeIndex(index)
	if err != nil {
		details := map[string]string{"index": index}
		utils.LavaError(ctx, ctx.Logger(), "GetAllEntryVersions", details, "invalid non-ascii entry")
		return nil
	}

	store := fs.getStore(ctx, safeIndex)

	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var entry types.Entry
		fs.cdc.MustUnmarshal(iterator.Value(), &entry)
		if !stale && entry.IsStale(ctx) {
			continue
		}
		blocks = append(blocks, entry.Block)
	}

	return blocks
}

func (fs FixationStore) createEntryIndexStoreKey() string {
	return types.EntryIndexKey + fs.prefix
}

func (fs FixationStore) createEntryIndexKey(safeIndex string) string {
	return types.EntryIndexKey + fs.prefix + safeIndex
}
