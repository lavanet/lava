package common

import (
	"fmt"
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common/types"
)

func prefixForErrors(from uint64) string {
	return fmt.Sprintf("FixationStore: migration from version %d", from)
}

var fixationMigrators = map[int]func(sdk.Context, *FixationStore) error{
	1: fixationMigrate1to2,
}

func (fs *FixationStore) MigrateVersion(ctx sdk.Context) (err error) {
	from := fs.getVersion(ctx)
	to := types.FixationVersion()

	for from < to {
		function, ok := fixationMigrators[int(from)]
		if !ok {
			return fmt.Errorf("%s not available", prefixForErrors(from))
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

// fixationMigrate1to2: fix refcounts
//   - correct refcount of head (most recent) by adding one (because new
//     entries used to begin with refcount 0 instead of refcount 1)
//   - correct negative refcounts if found any (for extra care)
func fixationMigrate1to2(ctx sdk.Context, fs *FixationStore) error {
	indices := fs.GetAllEntryIndices(ctx)
	for _, index := range indices {
		safeIndex, err := sanitizeIndex(index)
		if err != nil {
			return fmt.Errorf("%s: failed to sanitize index: %s", prefixForErrors(1), index)
		}
		blocks := fs.GetAllEntryVersions(ctx, index, true)
		if len(blocks) < 1 {
			return fmt.Errorf("%s: no versions for index: %s", prefixForErrors(1), index)
		}
		recent := blocks[len(blocks)-1]
		for _, block := range blocks {
			entry := fs.getEntry(ctx, safeIndex, block)
			// check for refcount overflow due to excessive putEntry
			if entry.Refcount > math.MaxInt64 {
				return fmt.Errorf("%s: entry has negative refcount index: %s", prefixForErrors(1), index)
			}
			// bump refcount of head entries (most recent version of an entry)
			if block == recent {
				entry.Refcount += 1
			}
			// if refcount still zero, make sure StaleAt is set
			if entry.Refcount == 0 && entry.StaleAt == math.MaxUint64 {
				entry.StaleAt = uint64(ctx.BlockHeight()) + uint64(types.STALE_ENTRY_TIME)
			}
			fs.setEntry(ctx, entry)
			// if StaleAt is set, then start corresponding timer
			if entry.StaleAt != math.MaxUint {
				fs.tstore.AddTimerByBlockHeight(ctx, entry.StaleAt, entry.Index)
			}
		}
	}
	return nil
}
