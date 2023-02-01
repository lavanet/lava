package common

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/utils"
)

// Function to get a specific fixated entry (by epoch) from the entry list
func getFixatedEntry(entryList []*types.Entry, epoch uint64) (*types.Entry, bool) {
	// check that entryList is not nil
	if entryList == nil {
		return nil, false
	}

	// search for the entry with the requested epoch
	for _, entry := range entryList {
		if entry.GetEpoch() == epoch {
			return entry, true
		}
	}

	return nil, false
}

// Function to set fixated entry (edit/add). Note that the function returns the new entryList so it can be updated in the relevant KVStore by the module
func setFixatedEntry(ctx sdk.Context, entryList []*types.Entry, newEntry *types.Entry) ([]*types.Entry, error) {
	// check that entryList and newEntry are not nil
	if entryList == nil || newEntry == nil {
		return nil, utils.LavaError(ctx, ctx.Logger(), "set_fixated_entry", nil, "entry list or modified entry is nil. Can't set fixated entry")
	}

	// try to find the entry. if found, modify the existing entry.
	for i, entry := range entryList {
		if entry.GetEpoch() == newEntry.GetEpoch() {
			entryList[i] = newEntry
			return entryList, nil
		}
	}

	// entry wasn't found in the entryList -> append the new entry
	entryList = append(entryList, newEntry)

	return entryList, nil
}

// Function to create a fixated entry from epoch and marshaled data
func createFixatedEntry(ctx sdk.Context, epoch uint64, marshaledData []byte) (*types.Entry, error) {
	// check that marshaledData is not nil
	if marshaledData == nil {
		return nil, utils.LavaError(ctx, ctx.Logger(), "create_fixated_entry", nil, "marshaled data is nil. Can't create fixated entry")
	}

	// create new entry
	newEntry := types.Entry{Epoch: epoch, MarshaledEntry: marshaledData}

	return &newEntry, nil
}
