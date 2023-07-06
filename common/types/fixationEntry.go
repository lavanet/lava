package types

import (
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

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

// IsMarkedDeleted tests whether an entry is marked for deletion, i.e. has
// entry.Deleted != math.MaxUint64.
func (entry Entry) HasDeleteAt() bool {
	return entry.DeleteAt < math.MaxUint64
}

// IsEntryIndexLive tests whether an entry-index is live
func IsEntryIndexLive(value []byte) bool {
	return value[0] == EntryIndexLive[0]
}
