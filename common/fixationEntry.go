package common

import (
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/utils"
)

type (
	GetterFunc  func(ctx sdk.Context, index string) (*types.Entry, bool)
	setterFunc  func(ctx sdk.Context, index string, entry *types.Entry) error
	removerFunc func(ctx sdk.Context, index string)
)

// Function to add an entry to the KVStore and update the older versions entries' indices that are saved in the KVStore
func AddFixatedEntryToStorage(ctx sdk.Context, entryToSet *types.Entry, index string, getEntry GetterFunc, setEntry setterFunc, removeEntry removerFunc) (bool, error) {
	// check that the function pointers are not nil
	if setEntry == nil || getEntry == nil || removeEntry == nil {
		return false, utils.LavaError(ctx, ctx.Logger(), "set_fixated_entry", nil, "setterFunc and/or GetterFunc input is nil")
	}

	isFirstVersion := false
	if !checkEntryProposedInThisBlock(ctx, entryToSet, index, getEntry) {
		// update the older versions entries indices
		isFirstVersionTemp, err := UpdateEntryIndices(ctx, index, getEntry, setEntry, removeEntry)
		if err != nil {
			return isFirstVersion, utils.LavaError(ctx, ctx.Logger(), "update_entry_indices", map[string]string{"entryToSetIndex": index}, "could not update entries indices (due to entry addition)")
		}
		isFirstVersion = isFirstVersionTemp
	}

	// set the new entry as the latest version (by using the index without number suffix)
	err := setEntry(ctx, index, entryToSet)
	if err != nil {
		return isFirstVersion, utils.LavaError(ctx, ctx.Logger(), "set_entry", map[string]string{"entryToSetIndex": index}, "could not set entry with index")
	}

	return isFirstVersion, nil
}

func checkEntryProposedInThisBlock(ctx sdk.Context, newEntryToPropose *types.Entry, newEntryIndex string, getEntry GetterFunc) bool {
	oldEntry, found := getEntry(ctx, newEntryIndex)
	if found {
		if newEntryToPropose.GetBlock() == oldEntry.GetBlock() {
			return true
		}
	}
	return false
}

// Function to update the indices of older versions of some entry due to new entry version (number suffix is increased by 1)
func UpdateEntryIndices(ctx sdk.Context, index string, getEntry GetterFunc, setEntry setterFunc, removeEntry removerFunc) (bool, error) {
	// get the index list of version entries saved in the KVStore
	entryList, _ := GetAllEntriesFromStorageByIndex(ctx, index, getEntry)

	// get the number of versions saved in the KVStore
	entryVersionsAmount := len(entryList)

	// check if the added package is the first version (no older versions available)
	isFirstVersion := false
	if entryVersionsAmount == 0 {
		isFirstVersion = true
	}

	// update the older version entries' indices (the number suffix is increased by 1). Note that we start from the oldest version back, and handle the latest version package separately (see below)
	for i := 0; i < entryVersionsAmount; i++ {
		// get the older version entry and its index (the suffix number of the index will be entryVersionsAmount-i-1 since the first entry (the latest version) is without suffix)
		olderVersionEntry := entryList[entryVersionsAmount-i-1]

		// construct an updated index for the older version entry (increase the number suffix by 1)
		olderVersionEntryUpdatedIndex := CreateOldVersionIndex(index, uint64(entryVersionsAmount-i-1))

		// set the older version entry with the updated index (overwrite)
		err := setEntry(ctx, olderVersionEntryUpdatedIndex, olderVersionEntry)
		if err != nil {
			return isFirstVersion, utils.LavaError(ctx, ctx.Logger(), "set_entry", map[string]string{"entryToSetIndex": olderVersionEntryUpdatedIndex, "isFirstVersion": strconv.FormatBool(isFirstVersion)}, "could not set entry with index")
		}
	}

	return isFirstVersion, nil
}

// Function to create a fixated entry from block and marshaled data
func CreateNewFixatedEntry(ctx sdk.Context, block uint64, marshaledData []byte) (*types.Entry, error) {
	// check that marshaledData is not nil
	if marshaledData == nil {
		return nil, utils.LavaError(ctx, ctx.Logger(), "create_new_fixated_entry", nil, "marshaled data is nil. Can't create fixated entry")
	}

	// create new entry
	newEntry := types.Entry{Block: block, MarshaledData: marshaledData}

	return &newEntry, nil
}

// Function to search for an entry in the storage (latest version's index is index, older versions' index is index_0, index_1, ...)
func GetEntryFromStorage(ctx sdk.Context, index string, block uint64, getEntry GetterFunc) (*types.Entry, bool) {
	// try getting the entry with the original index. return only if the requested block is larger than the entry's block
	entry, found := getEntry(ctx, index)
	if found {
		if block >= entry.GetBlock() {
			return entry, true
		}
	} else {
		// couldn't find the entry
		return nil, false
	}

	// couldn't find the entry for requested block (block too small) -> may be an older version. older version indices are "index_0" (or other numbers)
	versionSuffixCounter := 0
	for {
		// construct the older version index
		versionIndex := CreateOldVersionIndex(index, uint64(versionSuffixCounter))

		// get the older version entry
		entry, found := getEntry(ctx, versionIndex)
		if found {
			if block > entry.GetBlock() {
				// entry old version found and the requested block is larger than the entry's block -> found the right entry
				return entry, true
			} else {
				// entry old version found and the requested block is smaller than the entry's block -> not the right entry version, update suffix to check older versions
				versionSuffixCounter += 1
			}
		} else {
			// entry wasn't found, break
			break
		}
	}

	return nil, false
}

func CreateOldVersionIndex(index string, suffixNum uint64) string {
	return index + "_" + strconv.FormatUint(suffixNum, 10)
}

// Function that gets an index for storage and returns all the entries and their corresponding indices (i.e., the latest version entry and all of its older versions)
func GetAllEntriesFromStorageByIndex(ctx sdk.Context, index string, getEntry GetterFunc) ([]*types.Entry, []string) {
	entryList := []*types.Entry{}
	indexList := []string{}

	// try getting the entry with the original index
	entry, found := getEntry(ctx, index)
	if found {
		entryList = append(entryList, entry)
		indexList = append(indexList, index)
	} else {
		// couldn't find the entry
		return nil, nil
	}

	// get the older versions of this entry
	versionSuffixCounter := 0
	for {
		// construct the older version index
		versionIndex := CreateOldVersionIndex(index, uint64(versionSuffixCounter))

		// get the older version entry
		oldVersionEntry, found := getEntry(ctx, versionIndex)
		if found {
			// entry old version found -> append to entry list and increase suffix counter to look for older versions
			entryList = append(entryList, oldVersionEntry)
			indexList = append(indexList, versionIndex)
			versionSuffixCounter += 1
		} else {
			// entry old version wasn't found -> break
			break
		}
	}

	return entryList, indexList
}

// Function to remove an entry from the KVStore
func RemoveEntryFromStorage(ctx sdk.Context, index string, removeEntry removerFunc) error {
	// check that the removerFunc function pointer is not nil
	if removeEntry == nil {
		return utils.LavaError(ctx, ctx.Logger(), "remove_entry_from_storage", nil, "removerFunc input is nil")
	}

	// remove the entry from the KVStore
	removeEntry(ctx, index)

	return nil
}
