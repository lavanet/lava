package common

import (
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/utils"
)

// Function to get a specific fixated entry (by epoch) from the entry list
func GetFixatedEntry(entryList []*types.Entry, epoch uint64) (*types.Entry, int, bool) {
	// check that entryList is not nil
	if entryList == nil {
		return nil, 0, false
	}

	// search for the entry with the requested epoch
	for entryIndex, entry := range entryList {
		if entry.GetEpoch() == epoch {
			return entry, entryIndex, true
		}
	}

	return nil, 0, false
}

// Function to get the latest entry added to the entry list. Since we prepend entries that we want to add, we return the first entry on the list
func GetLatestFixatedEntry(entrylist []*types.Entry) *types.Entry {
	if entrylist == nil {
		return nil
	}

	return entrylist[0]
}

// Function to set fixated entry (add/delete). Note that the function returns the new entryList so it can be updated in the relevant KVStore by the module
func SetFixatedEntry(ctx sdk.Context, entryList []*types.Entry, entry *types.Entry, shouldAdd bool) ([]*types.Entry, error) {
	// check that entryList and entry are not nil
	if entryList == nil || entry == nil {
		return nil, utils.LavaError(ctx, ctx.Logger(), "set_fixated_entry", nil, "entry list or modified entry is nil. Can't set fixated entry")
	}

	// handle package addition
	if shouldAdd {
		// make sure that the new entry's epoch field is bigger than the epoch field of the latest entry in entryList
		if entry.GetEpoch() < entryList[0].GetEpoch() {
			return nil, utils.LavaError(ctx, ctx.Logger(), "set_fixated_entry", nil, "the new entry's epoch is smaller than or equal to the latest entry in entryList. The new entry's epoch must be larger")
		} else if entry.GetEpoch() == entryList[0].GetEpoch() {
			// the new entry has the same epoch as the latest entry -> switch the latest entry
			entryList[0] = entry
		} else {
			// the new entry is in an epoch that is bigger than the latest entry -> prepend the new entry
			entryList = append([]*types.Entry{entry}, entryList...)
		}

		// handle package removal
	} else {
		// get the entry to delete's index in entryList
		_, entryToDeleteIndex, found := GetFixatedEntry(entryList, entry.GetEpoch())
		if !found {
			return nil, utils.LavaError(ctx, ctx.Logger(), "get_fixated_entry", map[string]string{"epoch": strconv.FormatUint(entry.GetEpoch(), 10)}, "could not get the entry to delete")
		}

		// create a tempList that will be entryList without the entryToDelete
		tempList := make([]*types.Entry, 0)
		tempList = append(tempList, entryList[:entryToDeleteIndex]...)
		tempList = append(tempList, entryList[entryToDeleteIndex+1:]...)

		// assign entryList to be tempList
		entryList = tempList
	}

	return entryList, nil
}

// Function to create a fixated entry from epoch and marshaled data
func CreateNewFixatedEntry(ctx sdk.Context, epoch uint64, marshaledData []byte) (*types.Entry, error) {
	// check that marshaledData is not nil
	if marshaledData == nil {
		return nil, utils.LavaError(ctx, ctx.Logger(), "create_new_fixated_entry", nil, "marshaled data is nil. Can't create fixated entry")
	}

	// create new entry
	newEntry := types.Entry{Epoch: epoch, MarshaledData: marshaledData}

	return &newEntry, nil
}

func CreateNewFixatedEntryStorage(ctx sdk.Context, newEntry *types.Entry) (*types.EntryStorage, error) {
	// check that newEntry is not nil
	if newEntry == nil {
		return nil, utils.LavaError(ctx, ctx.Logger(), "create_new_fixated_entry_storage", nil, "new entry is nil. Can't create fixated storage entry")
	}

	// create new entry list that contains the new entry
	newEntryList := []*types.Entry{newEntry}

	// create new storage entry
	newStorageEntry := types.EntryStorage{EntryList: newEntryList}

	return &newStorageEntry, nil
}

func GetMarshaledDataFromEntry(ctx sdk.Context, entry *types.Entry) ([]byte, error) {
	// check that entry is not nil
	if entry == nil {
		return nil, utils.LavaError(ctx, ctx.Logger(), "get_marshaled_data_from_entry", nil, "entry is nil. Can't get marshaled data")
	}

	return entry.GetMarshaledData(), nil
}
