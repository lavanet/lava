package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// IsEntryStale tests whether an entry is stale, i.e. has refcount zero _and_
// is more than STALE_ENTRY_TIME blocks old (since creation).
func (entry Entry) IsStale(ctx sdk.Context) bool {
	if entry.GetRefcount() == 0 {
		if int64(entry.Block)+STALE_ENTRY_TIME < ctx.BlockHeight() {
			return true
		}
	}
	return false
}
