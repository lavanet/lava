package types

import (
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils/common/types"
)

// SafeIndex is a sanitized string, i.e. contains only visible ascii characters
// (i.e. Ascii 32-126), and terminates with (ascii) DEL; this ensures that an
// index can never be a prefix of another index.
type SafeIndex string

// sanitizeIndex checks that a string contains only visible ascii characters
// (i.e. Ascii 32-126), and appends a (ascii) DEL to the index; this ensures
// that an index can never be a prefix of another index.
func SanitizeIndex(index string) (SafeIndex, error) {
	for i := 0; i < len(index); i++ {
		if index[i] < types.ASCII_MIN || index[i] > types.ASCII_MAX {
			return SafeIndex(""), ErrInvalidIndex
		}
	}
	return SafeIndex(index + string([]byte{types.ASCII_DEL})), nil
}

// desantizeIndex reverts the effect of SanitizeIndex - removes the trailing
// (ascii) DEL terminator.
func DesanitizeIndex(safeIndex SafeIndex) string {
	return string(safeIndex[0 : len(safeIndex)-1])
}

func AssertSanitizedIndex(safeIndex SafeIndex, prefix string) {
	if []byte(safeIndex)[len(safeIndex)-1] != types.ASCII_DEL {
		// panic:ok: intended assertion
		panic("Fixation: prefix " + prefix + ": unsanitized safeIndex: " + string(safeIndex))
	}
}

// SafeIndex returns the entry's index.
func (entry Entry) SafeIndex() SafeIndex {
	return SafeIndex(entry.Index)
}

// IsStaleBy tests whether an entry is stale, i.e. has refcount zero _and_
// has passed its stale_at time (more than STALE_ENTRY_TIME since deletion).
func (entry Entry) IsStaleBy(block uint64) bool {
	if entry.GetRefcount() == 0 {
		if entry.StaleAt <= block {
			return true
		}
	}
	return false
}

// IsStale tests whether an entry is currently stale, i.e. has refcount zero _and_
// has passed its stale_at time (more than STALE_ENTRY_TIME since deletion).
func (entry Entry) IsStale(ctx sdk.Context) bool {
	return entry.IsStaleBy(uint64(ctx.BlockHeight()))
}

// IsDeletedBy tests whether an entry is deleted, with respect to a given
// block, i.e. has entry.DeletAt smaller or equal to that that block.
func (entry Entry) IsDeletedBy(block uint64) bool {
	return entry.DeleteAt <= block
}

// IsDeleted tests whether an entry is currently deleted, i.e. has its
// entry.DeleteAt below the current ctx block height.
func (entry Entry) IsDeleted(ctx sdk.Context) bool {
	return entry.IsDeletedBy(uint64(ctx.BlockHeight()))
}

// HasDeleteAt tests whether an entry is marked for deletion, i.e. has
// entry.Deleted != math.MaxUint64.
func (entry Entry) HasDeleteAt() bool {
	return entry.DeleteAt < math.MaxUint64
}

// IsEntryIndexLive tests whether an entry-index is live
func IsEntryIndexLive(value []byte) bool {
	return value[0] == EntryIndexLive[0]
}
