package common

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/utils"
)

// Set entry with full index
func SetEntry(ctx sdk.Context, storeKey sdk.StoreKey, entryKey string, cdc codec.BinaryCodec, entry types.Entry) {
	store := prefix.NewStore(ctx.KVStore(storeKey), []byte(entryKey))
	b := cdc.MustMarshal(&entry)
	store.Set(EntryKey(
		entry.Index,
	), b)
}

// Get entry with full index
func GetEntry(ctx sdk.Context, storeKey sdk.StoreKey, entryKey string, cdc codec.BinaryCodec, entryIndex string) (val types.Entry, found bool) {
	store := prefix.NewStore(ctx.KVStore(storeKey), []byte(entryKey))

	b := store.Get(EntryKey(
		entryIndex,
	))
	if b == nil {
		return val, false
	}

	cdc.MustUnmarshal(b, &val)
	return val, true
}

// Remove entry with full index
func RemoveEntry(ctx sdk.Context, storeKey sdk.StoreKey, entryKey string, entryIndex string) {
	store := prefix.NewStore(ctx.KVStore(storeKey), []byte(entryKey))
	store.Delete(EntryKey(
		entryIndex,
	))
}

// Get all entry with full index
func GetAllEntry(ctx sdk.Context, storeKey sdk.StoreKey, entryKey string, cdc codec.BinaryCodec) (list []types.Entry) {
	store := prefix.NewStore(ctx.KVStore(storeKey), []byte(entryKey))
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

// Function to add an entry to the KVStore and update the older versions entries' indices that are saved in the KVStore
func AddFixatedEntry(ctx sdk.Context, storeKey sdk.StoreKey, entryKey string, cdc codec.BinaryCodec, entryIndex string, marshalledData []byte) (bool, error) {
	entryToSet, err := CreateNewFixatedEntry(ctx, entryIndex, uint64(ctx.BlockHeight()), marshalledData)
	if err != nil {
		return false, err
	}

	isFirstVersion := false
	if !checkEntryProposedInThisBlock(ctx, storeKey, entryKey, cdc, entryToSet) {
		// update the older versions entries indices
		isFirstVersionTemp, err := UpdateEntryIndices(ctx, storeKey, entryKey, cdc, entryToSet.Index)
		if err != nil {
			return isFirstVersion, utils.LavaError(ctx, ctx.Logger(), "update_entry_indices", map[string]string{"entryToSetIndex": entryToSet.Index}, "could not update entries indices (due to entry addition)")
		}
		isFirstVersion = isFirstVersionTemp
	}

	// set the new entry as the latest version (by using the index without number suffix)
	SetEntry(ctx, storeKey, entryKey, cdc, *entryToSet)

	return isFirstVersion, nil
}

func checkEntryProposedInThisBlock(ctx sdk.Context, storeKey sdk.StoreKey, entryKey string, cdc codec.BinaryCodec, newEntryToPropose *types.Entry) bool {
	oldEntry, found := GetEntry(ctx, storeKey, entryKey, cdc, newEntryToPropose.Index)
	if found {
		if newEntryToPropose.GetBlock() == oldEntry.GetBlock() {
			return true
		}
	}
	return false
}

// Function to update the indices of older versions of some entry due to new entry version (number suffix is increased by 1)
func UpdateEntryIndices(ctx sdk.Context, storeKey sdk.StoreKey, entryKey string, cdc codec.BinaryCodec, entryIndex string) (bool, error) {
	oldEntry, found := GetEntry(ctx, storeKey, entryKey, cdc, entryIndex)
	var olderEntry types.Entry
	var i uint64
	for i = 0; found; i++ {
		// get the older version entry and its index (the suffix number of the index will be entryVersionsAmount-i-1 since the first entry (the latest version) is without suffix)
		olderEntry, found = GetEntry(ctx, storeKey, entryKey, cdc, entryIndex)

		// construct an updated index for the older version entry (increase the number suffix by 1)
		oldEntry.Index = CreateOldVersionIndex(entryIndex, i)

		// set the older version entry with the updated index (overwrite)
		SetEntry(ctx, storeKey, entryKey, cdc, oldEntry)

		oldEntry = olderEntry
	}

	return i == 0, nil
}

// Function to create a fixated entry from block and marshaled data
func CreateNewFixatedEntry(ctx sdk.Context, index string, block uint64, marshaledData []byte) (*types.Entry, error) {
	// check that marshaledData is not nil
	if marshaledData == nil {
		return nil, utils.LavaError(ctx, ctx.Logger(), "create_new_fixated_entry", nil, "marshaled data is nil. Can't create fixated entry")
	}

	// create new entry
	newEntry := types.Entry{Index: index, Block: block, MarshaledData: marshaledData}

	return &newEntry, nil
}

// Function to search for an entry in the storage (latest version's index is index, older versions' index is index_0, index_1, ...)
func GetEntryForBlock(ctx sdk.Context, storeKey sdk.StoreKey, entryKey string, cdc codec.BinaryCodec, index string, block uint64) (*types.Entry, bool) {
	// try getting the entry with the original index. return only if the requested block is larger than the entry's block
	entry, found := GetEntry(ctx, storeKey, entryKey, cdc, index)
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
		entry, found := GetEntry(ctx, storeKey, entryKey, cdc, versionIndex)
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

func CreateOldVersionIndex(index string, suffixNum uint64) string {
	return index + "_" + strconv.FormatUint(suffixNum, 10)
}

// Function that gets an index for storage and returns all the entries and their corresponding indices (i.e., the latest version entry and all of its older versions)
func GetAllEntriesForIndex(ctx sdk.Context, storeKey sdk.StoreKey, entryKey string, cdc codec.BinaryCodec, index string) []*types.Entry {
	entryList := []*types.Entry{}

	// try getting the entry with the original index
	entry, found := GetEntry(ctx, storeKey, entryKey, cdc, index)
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
		oldVersionEntry, found := GetEntry(ctx, storeKey, entryKey, cdc, versionIndex)
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
