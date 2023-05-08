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
	types.AssertSanitizedIndex(safeIndex, fs.prefix)
	store := fs.getEntryIndexStore(ctx)
	appendedValue := []byte(safeIndex) // convert the index value to a byte array
	store.Set(types.KeyPrefix(fs.createEntryIndexKey(safeIndex)), appendedValue)
}

// removeEntryIndex removes an Entry index from the store
func (fs FixationStore) removeEntryIndex(ctx sdk.Context, safeIndex string) {
	types.AssertSanitizedIndex(safeIndex, fs.prefix)
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
		types.AssertSanitizedIndex(safeIndex, fs.prefix)
		indexList = append(indexList, types.DesanitizeIndex(safeIndex))
	}

	return indexList
}

// GetAllEntryIndices returns all Entry indices
func (fs FixationStore) GetAllEntryIndices(ctx sdk.Context) []string {
	return fs.GetAllEntryIndicesWithPrefix(ctx, "")
}

func (fs *FixationStore) getEntryVersionsFilter(ctx sdk.Context, index string, block uint64, filter func(*types.Entry) bool) (blocks []uint64) {
	safeIndex, err := types.SanitizeIndex(index)
	if err != nil {
		details := map[string]string{"index": index}
		utils.LavaError(ctx, ctx.Logger(), "getEntryVersionsFilter", details, "invalid non-ascii entry")
		return nil
	}

	store := fs.getStore(ctx, safeIndex)

	iterator := sdk.KVStoreReversePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var entry types.Entry
		fs.cdc.MustUnmarshal(iterator.Value(), &entry)

		if filter(&entry) {
			blocks = append(blocks, entry.Block)
		}

		if entry.Block <= block {
			break
		}
	}

	// reverse the result slice to return the block in ascending order
	length := len(blocks)
	for i := 0; i < length/2; i++ {
		blocks[i], blocks[length-i-1] = blocks[length-i-1], blocks[i]
	}

	return blocks
}

// GetEntryVersionsRange returns a list of versions from nearest-smaller block
// and onward, and not more than delta blocks further (skip stale entries).
func (fs *FixationStore) GetEntryVersionsRange(ctx sdk.Context, index string, block, delta uint64) (blocks []uint64) {
	filter := func(entry *types.Entry) bool {
		if entry.IsStale(ctx) {
			return false
		}
		if entry.Block > block+delta {
			return false
		}
		return true
	}

	return fs.getEntryVersionsFilter(ctx, index, block, filter)
}

// GetAllEntryVersions returns a list of all versions (blocks) of an entry.
// If stale == true, then the output will include stale versions (for testing).
func (fs *FixationStore) GetAllEntryVersions(ctx sdk.Context, index string, stale bool) (blocks []uint64) {
	filter := func(entry *types.Entry) bool {
		if !stale && entry.IsStale(ctx) {
			return false
		}
		return true
	}

	return fs.getEntryVersionsFilter(ctx, index, 0, filter)
}

func (fs FixationStore) createEntryIndexStoreKey() string {
	return types.EntryIndexPrefix + fs.prefix
}

func (fs FixationStore) createEntryIndexKey(safeIndex string) string {
	return types.EntryIndexPrefix + fs.prefix + safeIndex
}
