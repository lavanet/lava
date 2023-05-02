package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func FixationVersion() uint64 {
	return 2
}

// IsStale tests whether an entry is stale, i.e. has refcount zero _and_
// has passed its stale_at time (more than STALE_ENTRY_TIME since deletion).
func (entry Entry) IsStale(ctx sdk.Context) bool {
	if entry.GetRefcount() == 0 {
		if entry.StaleAt <= uint64(ctx.BlockHeight()) {
			return true
		}
	}
	return false
}
