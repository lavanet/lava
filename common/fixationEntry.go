package common

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/utils"
)

// Set entry with full index. Full index is the entry index + version num suffix (like "bundle1_0")
func SetEntry(ctx sdk.Context, storeKey sdk.StoreKey, entryKeyPrefix string, cdc codec.BinaryCodec, entry types.Entry) {
	store := prefix.NewStore(ctx.KVStore(storeKey), []byte(entryKeyPrefix))
	b := cdc.MustMarshal(&entry)
	store.Set(EntryKey(
		entry.Index,
	), b)
}

// Get entry with full index. Full index is the entry index + version num suffix (like "bundle1_0")
func GetEntry(ctx sdk.Context, storeKey sdk.StoreKey, entryKeyPrefix string, cdc codec.BinaryCodec, entryIndex string) (val types.Entry, found bool) {
	store := prefix.NewStore(ctx.KVStore(storeKey), []byte(entryKeyPrefix))

	b := store.Get(EntryKey(
		entryIndex,
	))
	if b == nil {
		return val, false
	}

	cdc.MustUnmarshal(b, &val)
	return val, true
}

// Remove entry with full index. Full index is the entry index + version num suffix (like "bundle1_0")
func RemoveEntry(ctx sdk.Context, storeKey sdk.StoreKey, entryKeyPrefix string, entryIndex string) {
	store := prefix.NewStore(ctx.KVStore(storeKey), []byte(entryKeyPrefix))
	store.Delete(EntryKey(
		entryIndex,
	))
}

// Get all entry with full index. Full index is the entry index + version num suffix (like "bundle1_0")
func GetAllEntry(ctx sdk.Context, storeKey sdk.StoreKey, entryKeyPrefix string, cdc codec.BinaryCodec) (list []types.Entry) {
	store := prefix.NewStore(ctx.KVStore(storeKey), []byte(entryKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.Entry
		cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func EntryKey(
	entryIndex string,
) []byte {
	var key []byte

	entryIndexBytes := []byte(entryIndex)
	key = append(key, entryIndexBytes...)
	key = append(key, []byte("/")...)

	return key
}

// Function to create a new fixation entry, add it to the KVStore and update the entry's older versions indices. Note, the entryIndex should be without the version num suffix
func AddFixatedEntry(ctx sdk.Context, storeKey sdk.StoreKey, entryKeyPrefix string, cdc codec.BinaryCodec, entryIndex string, marshalledData []byte) (bool, error) {
	// create a new fixated entry
	entryToSet, err := CreateNewFixatedEntry(ctx, entryIndex, uint64(ctx.BlockHeight()), marshalledData)
	if err != nil {
		return false, err
	}

	isFirstVersion := false
	// check if there were more entries proposed in this block that have the same entryIndex as entryToSet
	if !checkEntryProposedInThisBlock(ctx, storeKey, entryKeyPrefix, cdc, entryToSet) {
		// update the older versions entries indices. Also return whether the entry is a first version entry (no older versions saved in the KVStore)
		isFirstVersion, err = UpdateEntryIndices(ctx, storeKey, entryKeyPrefix, cdc, entryToSet.GetIndex())
		if err != nil {
			return isFirstVersion, utils.LavaError(ctx, ctx.Logger(), "AddFixatedEntry_entry_indices_update_failed", map[string]string{"entryToSetIndex": entryToSet.Index}, "could not update entries indices")
		}
	}

	// set the new entry as the latest version (by using the index without number suffix)
	SetEntry(ctx, storeKey, entryKeyPrefix, cdc, *entryToSet)

	return isFirstVersion, nil
}

// Function to check whether two entries with the same index are added in the same block
func checkEntryProposedInThisBlock(ctx sdk.Context, storeKey sdk.StoreKey, entryKeyPrefix string, cdc codec.BinaryCodec, newEntryToPropose *types.Entry) bool {
	// try getting an entry with the same index as newEntryToPropose
	oldEntry, found := GetEntry(ctx, storeKey, entryKeyPrefix, cdc, newEntryToPropose.GetIndex())
	if found {
		// if found, check if the new entry has the same block field
		if newEntryToPropose.GetBlock() == oldEntry.GetBlock() {
			return true
		}
	}
	return false
}

// Function to update the indices of older versions of some entry due to new entry version (number suffix is increased by 1)
func UpdateEntryIndices(ctx sdk.Context, storeKey sdk.StoreKey, entryKeyPrefix string, cdc codec.BinaryCodec, entryIndex string) (bool, error) {
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
func CreateNewFixatedEntry(ctx sdk.Context, index string, block uint64, marshaledData []byte) (*types.Entry, error) {
	// check that marshaledData is not nil
	if marshaledData == nil {
		return nil, utils.LavaError(ctx, ctx.Logger(), "CreateNewFixatedEntry_failed", nil, "can't create new fixated entry, marshaled data is nil")
	}

	// create new entry
	newEntry := types.Entry{Index: index, Block: block, MarshaledData: marshaledData}

	return &newEntry, nil
}

// Function to search for an entry in the storage (latest version's index is index, older versions' index is index_0, index_1, ...)
func GetEntryOlderVersionByBlock(ctx sdk.Context, storeKey sdk.StoreKey, entryKeyPrefix string, cdc codec.BinaryCodec, index string, block uint64) (*types.Entry, bool) {
	// try getting the entry with the original index. return only if the requested block is larger than the entry's block
	entry, found := GetEntry(ctx, storeKey, entryKeyPrefix, cdc, index)
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
		entry, found := GetEntry(ctx, storeKey, entryKeyPrefix, cdc, versionIndex)
		if found {
			if block > entry.GetBlock() {
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
	entry, found := GetEntry(ctx, storeKey, entryKeyPrefix, cdc, index)
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
		oldVersionEntry, found := GetEntry(ctx, storeKey, entryKeyPrefix, cdc, versionIndex)
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
