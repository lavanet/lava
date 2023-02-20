package common

import (
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/utils"
)

/*
This library provides a standard set of API for managing parameters that are meant to be fixated.
These API should enable users to easily set, modify, and delete fixated parameters.
A fixated parameter is one whose value is retained on-chain with the block in which it was created when it is changed.
By contrast, when a non-fixated parameter is altered, only its most recent value is stored.

Fixated entry structure:
Entry {
    string index; 			// a unique entry index
    uint64 block; 			// block the entry was created
    bytes marshaled_data;   // marshaled data of the entry
}

How does it work?
All the entries have one-of-a-kind indices that are in the following structure:
	- Latest version: `entryIndex`
	- Older versions: `entryIndex_0`, `entryIndex_1`, and so on.

Entry Addition:
When adding a new entry, we set it in the store and add a unique index to the uniqueIndex list.
When adding an update to an entry, we update all the indices of past version (increase the version number suffix by 1) and
set the updated entry as the latest version (its index is without a version number suffix).
Note, if two entries with the same index are added in the same block, we only keep the latest one.

Entry Removal:
Entries can be removed with the RemoveEntry API. Currently, there is no deletion of the corresponding
fixationEntryUniqueIndex (should be deleted when the deleted entry is the last remaining version).
*/

// Set entry with full index. Full index is the entry index + version num suffix (like "bundle1_0")
func SetEntry(ctx sdk.Context, storeKey sdk.StoreKey, entryKeyPrefix string, cdc codec.BinaryCodec, entry types.Entry) {
	store := prefix.NewStore(ctx.KVStore(storeKey), []byte(entryKeyPrefix))
	b := cdc.MustMarshal(&entry)
	store.Set(entryKey(
		entry.Index,
	), b)
}

// Get entry with full index. Full index is the entry index + version num suffix (like "bundle1_0")
func GetEntry(ctx sdk.Context, storeKey sdk.StoreKey, entryKeyPrefix string, cdc codec.BinaryCodec, entryIndex string, referenceAction int) (val types.Entry, found bool) {
	store := prefix.NewStore(ctx.KVStore(storeKey), []byte(entryKeyPrefix))

	b := store.Get(entryKey(
		entryIndex,
	))
	if b == nil {
		return val, false
	}

	cdc.MustUnmarshal(b, &val)

	switch referenceAction {
	case types.ADD_REFERENCE:
		val.References += 1
		SetEntry(ctx, storeKey, entryKeyPrefix, cdc, val)
	case types.SUB_REFERENCE:
		if val.GetReferences() > 0 {
			val.References -= 1
			SetEntry(ctx, storeKey, entryKeyPrefix, cdc, val)
		}
	case types.DO_NOTHING:
	}

	return val, true
}

// Helper function to extract the original index from an older version index (for example: myIndex_2 -> myIndex)
func extractOriginalIndexFromOlderVersionIndex(index string) (originalIndexWithoutVersionNumSuffix string) {
	// check that the index's length is non-zero
	if len(index) == 0 {
		return ""
	}

	// if the index doesn't contain "_", it's not an older version index (as defined in this code) so there's nothing to do
	if !strings.Contains(index, "_") {
		return index
	}

	// trim the index string from the start to the last "_"
	if i := strings.LastIndex(index, "_"); i > 0 {
		index = index[:i]
	}

	return index
}

// Remove entry with full index. Full index is the entry index + version num suffix (like "bundle1_0")
func RemoveEntry(ctx sdk.Context, storeKey sdk.StoreKey, cdc codec.BinaryCodec, entryKeyPrefix string, entryIndex string) {
	store := prefix.NewStore(ctx.KVStore(storeKey), []byte(entryKeyPrefix))
	store.Delete(entryKey(
		entryIndex,
	))

	originalIndex := extractOriginalIndexFromOlderVersionIndex(entryIndex)

	entryList := GetAllEntriesForIndex(ctx, storeKey, entryKeyPrefix, cdc, originalIndex)
	if len(entryList) == 0 {
		uniqueIndices := GetAllFixationEntryUniqueIndex(ctx, storeKey, cdc, entryKeyPrefix)
		for _, uniqueIndex := range uniqueIndices {
			if originalIndex == uniqueIndex.GetUniqueIndex() {
				RemoveFixationEntryUniqueIndex(ctx, storeKey, entryKeyPrefix, uniqueIndex.GetId())
			}
		}
	}
}

func entryKey(
	entryIndex string,
) []byte {
	var key []byte

	entryIndexBytes := []byte(entryIndex)
	key = append(key, entryIndexBytes...)
	key = append(key, []byte("/")...)

	return key
}

func deleteStaleEntries(ctx sdk.Context, storeKey sdk.StoreKey, entryKeyPrefix string, cdc codec.BinaryCodec, index string) {
	// get current block
	currentBlock := ctx.BlockHeight()

	// get all the entries of the entry's index
	entryList := GetAllEntriesForIndex(ctx, storeKey, entryKeyPrefix, cdc, index)

	// iterate over the entries
	for _, entry := range entryList {
		// if the entry's references is zero and the entry's block + STALE_ENTRY_TIME (the time it takes for an entry to be stale) is smaller than the current block, remove it
		if entry.GetReferences() == 0 && int64(entry.GetBlock())+types.STALE_ENTRY_TIME < currentBlock {
			RemoveEntry(ctx, storeKey, cdc, entryKeyPrefix, index)
		}
	}
}

// Function to create a new fixation entry, add it to the KVStore and update the entry's older versions indices. Note, the entryIndex should be without the version num suffix
func AddEntry(ctx sdk.Context, storeKey sdk.StoreKey, entryKeyPrefix string, uniqueIndexEntryKeyPrefix string, cdc codec.BinaryCodec, entryIndex string, marshalledData []byte) error {
	// delete stale entries
	deleteStaleEntries(ctx, storeKey, entryKeyPrefix, cdc, entryIndex)

	// since we're keeping versions with a "_<num>" suffix, to avoid trouble we forbid that the last character would be "_"
	if entryIndex[len(entryIndex)-1:] == "_" {
		return utils.LavaError(ctx, ctx.Logger(), "AddEntry_invalid_entryIndex", map[string]string{"entryIndex": entryIndex}, "entry index must not end with \"_\"")
	}

	// create a new fixated entry
	entryToSet, err := CreateNewEntry(ctx, entryIndex, uint64(ctx.BlockHeight()), marshalledData)
	if err != nil {
		return utils.LavaError(ctx, ctx.Logger(), "AddEntry_create_new_fixated_entry_failed", map[string]string{"err": err.Error()}, "could not create new fixated entry")
	}

	isFirstVersion := false
	// check if there were more entries proposed in this block that have the same entryIndex as entryToSet
	if !checkEntryProposedInThisBlock(ctx, storeKey, entryKeyPrefix, cdc, entryToSet) {
		// update the older versions entries indices. Also return whether the entry is a first version entry (no older versions saved in the KVStore)
		isFirstVersion, err = updateEntryIndices(ctx, storeKey, entryKeyPrefix, cdc, entryToSet.GetIndex())
		if err != nil {
			return utils.LavaError(ctx, ctx.Logger(), "AddEntry_entry_indices_update_failed", map[string]string{"entryToSetIndex": entryToSet.Index}, "could not update entries indices")
		}
	}

	// set the new entry as the latest version (by using the index without number suffix)
	SetEntry(ctx, storeKey, entryKeyPrefix, cdc, *entryToSet)

	// set a unique index if the entry is a first version entry
	if isFirstVersion {
		AppendFixationEntryUniqueIndex(ctx, storeKey, cdc, uniqueIndexEntryKeyPrefix, types.UniqueIndex{UniqueIndex: entryToSet.GetIndex()})
	}

	return nil
}

// Function to check whether two entries with the same index are added in the same block
func checkEntryProposedInThisBlock(ctx sdk.Context, storeKey sdk.StoreKey, entryKeyPrefix string, cdc codec.BinaryCodec, newEntryToPropose *types.Entry) bool {
	// try getting an entry with the same index as newEntryToPropose
	oldEntry, found := GetEntry(ctx, storeKey, entryKeyPrefix, cdc, newEntryToPropose.GetIndex(), types.DO_NOTHING)
	if found {
		// if found, check if the new entry has the same block field
		if newEntryToPropose.GetBlock() == oldEntry.GetBlock() {
			return true
		}
	}
	return false
}

// Function to update the indices of older versions of some entry due to new entry version (number suffix is increased by 1)
func updateEntryIndices(ctx sdk.Context, storeKey sdk.StoreKey, entryKeyPrefix string, cdc codec.BinaryCodec, entryIndex string) (bool, error) {
	// get all the entries for a given index (all its versions)
	entries := GetAllEntriesForIndex(ctx, storeKey, entryKeyPrefix, cdc, entryIndex)

	// if there are no older versions of this entry -> there is no need to update indices, return that it's a first version entry
	if len(entries) == 0 {
		return true, nil
	}

	// go over all the older entries and update their indices
	for i := len(entries) - 1; i >= 0; i-- {
		// get the old version entry
		oldVersionEntry := *entries[i]

		// construct an updated index for the old version entry (increase the number suffix by 1)
		oldVersionEntry.Index = CreateOldVersionIndex(entryIndex, uint64(i))

		// set the old version entry with the updated index (overwrite)
		SetEntry(ctx, storeKey, entryKeyPrefix, cdc, oldVersionEntry)
	}

	return false, nil
}

// Function to create a fixated entry from block and marshaled data
func CreateNewEntry(ctx sdk.Context, index string, block uint64, marshaledData []byte) (*types.Entry, error) {
	// check that marshaledData is not nil
	if len(marshaledData) == 0 {
		return nil, utils.LavaError(ctx, ctx.Logger(), "CreateNewFixatedEntry_failed", nil, "can't create new fixated entry, marshaled data is nil")
	}

	// create new entry
	newEntry := types.Entry{Index: index, Block: block, MarshaledData: marshaledData, References: 0}

	return &newEntry, nil
}

// Function to search for an entry in the storage (latest version's index is index, older versions' index is index_0, index_1, ...)
func GetEntryOlderVersionByBlock(ctx sdk.Context, storeKey sdk.StoreKey, entryKeyPrefix string, cdc codec.BinaryCodec, index string, block uint64) (*types.Entry, bool) {
	// try getting the entry with the original index. return only if the requested block is larger than the entry's block
	entry, found := GetEntry(ctx, storeKey, entryKeyPrefix, cdc, index, types.DO_NOTHING)
	if found {
		if block >= entry.GetBlock() {
			return &entry, true
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
		entry, found := GetEntry(ctx, storeKey, entryKeyPrefix, cdc, versionIndex, types.DO_NOTHING)
		if found {
			if block >= entry.GetBlock() {
				// entry old version found and the requested block is larger than the entry's block -> found the right entry
				return &entry, true
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

// Function to create an old version index
func CreateOldVersionIndex(index string, suffixNum uint64) string {
	return index + "_" + strconv.FormatUint(suffixNum, 10)
}

// Function that gets an index for storage and returns all the entries and their corresponding indices (i.e., the latest version entry and all of its older versions)
func GetAllEntriesForIndex(ctx sdk.Context, storeKey sdk.StoreKey, entryKeyPrefix string, cdc codec.BinaryCodec, index string) []*types.Entry {
	entryList := []*types.Entry{}

	// try getting the entry with the original index
	entry, found := GetEntry(ctx, storeKey, entryKeyPrefix, cdc, index, types.DO_NOTHING)
	if found {
		entryList = append(entryList, &entry)
	} else {
		// couldn't find the entry
		return nil
	}

	// get the older versions of this entry
	versionSuffixCounter := 0
	for {
		// construct the older version index
		versionIndex := CreateOldVersionIndex(index, uint64(versionSuffixCounter))

		// get the older version entry
		oldVersionEntry, found := GetEntry(ctx, storeKey, entryKeyPrefix, cdc, versionIndex, types.DO_NOTHING)
		if found {
			// entry old version found -> append to entry list and increase suffix counter to look for older versions
			entryList = append(entryList, &oldVersionEntry)
			versionSuffixCounter += 1
		} else {
			// entry old version wasn't found -> break
			break
		}
	}

	return entryList
}
