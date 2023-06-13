package common

import (
	"fmt"
	"math"
	"strings"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/utils"
)

func (fs *FixationStore) prefixForErrors(from uint64) string {
	return fmt.Sprintf("FixationStore: migration from version %d", from)
}

var fixationMigrators = map[int]func(sdk.Context, *FixationStore) error{
	1: fixationMigrate1to2,
	2: fixationMigrate2to3,
	3: fixationMigrate3to4,
}

// MigrateVerrsion performs pending internal version migration(s), if any.
func (fs *FixationStore) MigrateVersion(ctx sdk.Context) (err error) {
	from := fs.getVersion(ctx)
	return fs.MigrateVersionFrom(ctx, from)
}

func (fs *FixationStore) MigrateVersionFrom(ctx sdk.Context, from uint64) (err error) {
	to := FixationVersion()

	for from < to {
		function, ok := fixationMigrators[int(from)]
		if !ok {
			return fmt.Errorf("%s not available", fs.prefixForErrors(from))
		}

		err = function(ctx, fs)
		if err != nil {
			return err
		}

		from += 1
	}

	fs.setVersion(ctx, to)
	return nil
}

// MigrateVersionAndPrefix performs pending internal version migration(s),
// if any, and then replaces the old prefix with a new (current) one. (For
// the version migration(s) it uses the old prefix).
func (fs *FixationStore) MigrateVersionAndPrefix(ctx sdk.Context, oldPrefix string) (err error) {
	newPrefix := fs.prefix

	// first check if version upgrade is due - must use old prefix
	fs.prefix = oldPrefix
	err = fs.MigrateVersion(ctx)
	fs.prefix = newPrefix

	if err != nil {
		return err
	}

	// prefix migration
	fs.doMigratePrefix(ctx, oldPrefix, newPrefix)
	return nil
}

// doMigratePrefix replaces objects prefix from `oldPrefix` to `newPrefix`
func (fs *FixationStore) doMigratePrefix(ctx sdk.Context, oldPrefix, newPrefix string) {
	store_V1 := prefix.NewStore(ctx.KVStore(fs.storeKey), types.KeyPrefix(oldPrefix))
	store_V2 := prefix.NewStore(ctx.KVStore(fs.storeKey), types.KeyPrefix(newPrefix))

	iterator := sdk.KVStorePrefixIterator(store_V1, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		store_V2.Set(iterator.Key(), iterator.Value())
		store_V1.Delete(iterator.Key())
	}
}

// fixationMigrate1to2: fix refcounts
//   - correct refcount of head (most recent) by adding one (because new
//     entries used to begin with refcount 0 instead of refcount 1)
//   - correct negative refcounts if found any (for extra care)
func fixationMigrate1to2(ctx sdk.Context, fs *FixationStore) error {
	ctxBlock := uint64(ctx.BlockHeight())

	indices := fs.GetAllEntryIndices(ctx)
	for _, index := range indices {
		safeIndex, err := types.SanitizeIndex(index)
		if err != nil {
			return fmt.Errorf("%s: failed to sanitize index: %s", fs.prefixForErrors(1), index)
		}
		blocks := fs.GetAllEntryVersions(ctx, index)
		if len(blocks) < 1 {
			return fmt.Errorf("%s: no versions for index: %s", fs.prefixForErrors(1), index)
		}
		recent := blocks[len(blocks)-1]
		for _, block := range blocks {
			entry := fs.getEntry(ctx, safeIndex, block)
			// check for refcount overflow due to excessive putEntry
			if entry.Refcount > math.MaxInt64 {
				return fmt.Errorf("%s: entry has negative refcount index: %s", fs.prefixForErrors(1), index)
			}
			// bump refcount of head entries (most recent version of an entry)
			if block == recent {
				entry.Refcount += 1
			}
			// if refcount still zero, make sure StaleAt is set
			if entry.Refcount == 0 && entry.StaleAt == math.MaxUint64 {
				entry.StaleAt = ctxBlock + uint64(types.STALE_ENTRY_TIME)
			}
			fs.setEntry(ctx, entry)
			// if StaleAt is set, then start corresponding timer
			if entry.StaleAt != math.MaxUint {
				fs.tstore.AddTimerByBlockHeight(ctx, entry.StaleAt, []byte{}, []byte(entry.Index))
			}
		}
	}
	return nil
}

// fixationMigrate2to3: fix refcounts
//   - instead of "EntryPrefix + fs.prefix" -> "fs.prefix + EntryPrefix"
//   - instead of "EntryIndexPrefix + fs.prefix"(x2) -> "fs.prefix + EntryIndexPrefix"
func fixationMigrate2to3(ctx sdk.Context, fs *FixationStore) error {
	const (
		// copy of old values
		V2_EntryIndexPrefix string = "Entry_Index_"
		V2_EntryPrefix      string = "Entry_Value_"
	)

	// copy EntryIndex
	v1 := strings.Repeat(V2_EntryIndexPrefix+fs.prefix, 2)
	v2 := fs.prefix + types.EntryIndexPrefix
	fs.doMigratePrefix(ctx, v1, v2)

	// copy Entry
	v1 = V2_EntryPrefix + fs.prefix
	v2 = fs.prefix + types.EntryPrefix
	fs.doMigratePrefix(ctx, v1, v2)

	return nil
}

// fixationMigrate3to4: update keys/data for timers
//   - instead of <expiry=block/time, data=index>,
//     use <expiry=block/time, key=timer+entry-index+entry-block, data=[]>
func fixationMigrate3to4(ctx sdk.Context, fs *FixationStore) error {
	err := fs.tstore.MigrateVersion(ctx)
	if err != nil {
		return err
	}

	utils.LavaFormatDebug("migrate fixation timers")

	ctxBlock := uint64(ctx.BlockHeight())

	// apply migration to all entries (even deleted ones as they may still
	// have versions in stale-period); use the raw AllEntryIndicesFilter()
	filter := func(_, v []byte) bool { return true }
	indices := fs.AllEntryIndicesFilter(ctx, "", filter)

	for _, index := range indices {
		utils.LavaFormatDebug("  entry", utils.Attribute{Key: "index", Value: index})

		safeIndex, err := types.SanitizeIndex(index)
		if err != nil {
			return fmt.Errorf("%s: failed to sanitize index: %s", fs.prefixForErrors(1), index)
		}

		fs.setEntryIndex(ctx, safeIndex, true)

		blocks := fs.GetAllEntryVersions(ctx, index)
		if len(blocks) < 1 {
			return fmt.Errorf("%s: no versions for index: %s", fs.prefixForErrors(1), index)
		}

		for _, block := range blocks {
			entry := fs.getEntry(ctx, safeIndex, block)
			utils.LavaFormatDebug("    version", utils.Attribute{Key: "entry", Value: entry})

			// if StaleAt is set, then replace old style timer with new style timer
			if entry.StaleAt != math.MaxUint && entry.StaleAt > ctxBlock {
				fs.tstore.DelTimerByBlockHeight(ctx, entry.StaleAt, []byte{})
				key := encodeForTimer(entry.Index, entry.Block, timerStaleEntry)
				fs.tstore.AddTimerByBlockHeight(ctx, entry.StaleAt, key, []byte{})
			}

			entry.DeleteAt = math.MaxUint64
			fs.setEntry(ctx, entry)
		}
	}

	return nil
}
