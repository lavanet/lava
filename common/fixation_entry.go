package common

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/utils"
)

/*
	This library provides a standard set of API for managing entries that are meant to be fixated.
	This API set should enable users to easily set, modify, and delete fixated entries.
	A fixated entry is one whose value is retained on-chain with the block in which it was created when it is changed.
	By contrast, when a non-fixated entry is altered, only its most recent value is stored.

	Fixated entry structure:
	Entry {
		string index;			// a unique entry name.
		uint64 block;			// the block the entry was created in.
		uint64 references; 		// Number of references to this entry (number of entities using it).
		bytes data; 			// The data saved in the entry. Can be any type.
	}

	How does it work?
	To help with the explanation, we'll use a reference example. Assume you created a custom module
	named "packages". This module keeps package objects as fixated entries (i.e., the module keeps track
	of every package AND its older versions). Each package has an "index" field which acts as a unique name
	for the package.

	In general, each object that uses this library will have a versioned store field of type VersionedStore.
	The versioned store (from now on: "vs") holds a unique store key, a codec and a map of unique indices.
	Using our reference example, the store key of the module "packages" will be "packages". If another module,
	say module "projects", also uses fixated entries, its store key will be "projects". This way we get a clear
	separation between the fixated entries of "packages" and of "projects". In other words, the store key acts as
	a namespace. The codec is used to determine how the packages should be marshaled, and the unique indices map
	helps to track which packages do we have (not including older versions packages).

	In each store (or namespace), marshaled objects of type Entry are saved. Note that a module can have different
	kind of fixated entries. They are separate between them, we use a fixationKey (a unique key for each type of
	fixated entry).

	Going back to the reference example, let's say I have a package named "myGreatPackage" that was created in block
	101 (it also has additional fields). The package's entry holds the marshaled package object (in the data field)
	and has 33 references. Also, let's say the packages module can hold fixated price objects. An example for a
	price object can be with price of 2ulava which was created on block 203. The price's entry holds the marshaled
	price object and has 12 references.
		The store key to access the package object will be "Entry_<packageFixationKey>". The package entry's key will
		be "Entry_<packageFixationKey>_myGreatPackage_101"

		The store key to access the price object will be "Entry_<priceFixationKey>". The package entry's key will
		be "Entry_<priceFixationKey>_2ulava_203"

	Moreover, to track the indices of fixated objects more easily (without regarding the indices of older versions
	of entries), we keep an EntryIndex list. See fixation_entry_index.go for more details.

*/

type VersionedStore struct {
	storeKey sdk.StoreKey
	cdc      codec.BinaryCodec
}

// AppendEntry adds a new entry to the store
func (vs VersionedStore) AppendEntry(ctx sdk.Context, fixationKey string, index string, block uint64, entryData codec.ProtoMarshaler) error {
	// delete old entries
	vs.deleteStaleEntries(ctx, fixationKey)

	// get the latest entry for this index
	latestEntry := vs.getUnmarshaledEntryForBlock(ctx, fixationKey, index, block, types.DO_NOTHING)

	// if latest entry is not found, this is a first version entry
	firstVersion := false
	if latestEntry == nil {
		firstVersion = true
	}

	// make sure the new entry's block is not smaller than the latest entry's block
	if !firstVersion && block < latestEntry.GetBlock() {
		return utils.LavaError(ctx, ctx.Logger(), "AppendEntry_block_too_early", map[string]string{"latestEntryBlock": strconv.FormatUint(latestEntry.GetBlock(), 10), "block": strconv.FormatUint(block, 10), "index": index, "fixationKey": fixationKey}, "can't append entry, earlier than the latest entry")
	}

	// if the new entry's block is equal to the latest entry, overwrite the latest entry
	if !firstVersion && block == latestEntry.GetBlock() {
		return vs.SetEntry(ctx, fixationKey, index, block, entryData)
	}

	// marshal the new entry's data
	b := vs.cdc.MustMarshal(entryData)
	// create a new entry and marshal it
	entry := types.Entry{Index: index, Block: block, Data: b, References: 0}
	bz := vs.cdc.MustMarshal(&entry)

	// get the relevant store
	store := prefix.NewStore(ctx.KVStore(vs.storeKey), types.KeyPrefix(types.EntryKey+fixationKey+index))
	byteKey := types.KeyPrefix(createEntryKey(index, fixationKey, block))

	// set the new entry to the store
	store.Set(byteKey, bz)

	// if it's a first version entry, add a new key in the uniqueIndices map
	if firstVersion {
		vs.AppendEntryIndex(ctx, fixationKey, index)
	}

	return nil
}

func (vs VersionedStore) deleteStaleEntries(ctx sdk.Context, fixationKey string) {
	entries := vs.getAllEntries(ctx, fixationKey)
	for _, entry := range entries {
		// get the relevant store and init an iterator
		store := prefix.NewStore(ctx.KVStore(vs.storeKey), types.KeyPrefix(types.EntryKey+fixationKey+entry.GetIndex()))
		iterator := sdk.KVStorePrefixIterator(store, []byte{})
		defer iterator.Close()

		// iterate over entries
		for ; iterator.Valid(); iterator.Next() {
			// umarshal the old entry version
			var oldEntry types.Entry
			vs.cdc.MustUnmarshal(iterator.Value(), &oldEntry)

			// if the entry's refs is equal to 0 and it has been longer than STALE_ENTRY_TIME from its creation, delete it
			if oldEntry.GetReferences() == 0 && int64(oldEntry.GetBlock())+types.STALE_ENTRY_TIME < ctx.BlockHeight() {
				vs.removeEntry(ctx, fixationKey, oldEntry.GetIndex(), oldEntry.GetBlock())
			} else {
				// else, break (avoiding removal of entries in the middle of the list)
				break
			}
		}

		// try getting this entry's latest version. If it doesn't exist, remove its index for the entry index list
		latestVersionEntry := vs.getUnmarshaledEntryForBlock(ctx, fixationKey, entry.GetIndex(), uint64(ctx.BlockHeight()), types.DO_NOTHING)
		if latestVersionEntry == nil {
			vs.RemoveEntryIndex(ctx, fixationKey, entry.GetIndex())
		}
	}
}

// SetEntry sets a specific entry in the store
func (vs VersionedStore) SetEntry(ctx sdk.Context, fixationKey string, index string, block uint64, entryData codec.ProtoMarshaler) error {
	// get the relevant store
	store := prefix.NewStore(ctx.KVStore(vs.storeKey), types.KeyPrefix(types.EntryKey+fixationKey+index))
	byteKey := types.KeyPrefix(createEntryKey(index, fixationKey, block))

	// marshal the new entry data
	b := vs.cdc.MustMarshal(entryData)

	// get the entry from the store
	entry := vs.getUnmarshaledEntryForBlock(ctx, fixationKey, index, block, types.DO_NOTHING)
	if entry == nil {
		return utils.LavaError(ctx, ctx.Logger(), "SetEntry_cant_find_entry", map[string]string{"fixationKey": fixationKey, "index": index, "block": strconv.FormatUint(block, 10)}, "can't set non-existent entry")
	}

	// update the entry's data
	entry.Data = b

	// marshal the entry
	bz := vs.cdc.MustMarshal(entry)

	// set the entry
	store.Set(byteKey, bz)

	return nil
}

// handleRefAction handles ref actions: increase refs, decrease refs or do nothing
func handleRefAction(ctx sdk.Context, entry *types.Entry, refAction types.ReferenceAction) error {
	// check if entry is nil
	if entry == nil {
		return utils.LavaError(ctx, ctx.Logger(), "handleRefAction_nil_entry", map[string]string{}, "can't handle reference action, entry is nil")
	}

	// handle the ref action
	switch refAction {
	case types.ADD_REFERENCE:
		entry.References += 1
	case types.SUB_REFERENCE:
		if entry.GetReferences() > 0 {
			entry.References -= 1
		}
	case types.DO_NOTHING:
	}

	return nil
}

// GetEntry gets the exact entry with index and block (block has to be precise). The user should pass an empty pointer of the desired type (e.g. package, subsciption, etc.).
func (vs VersionedStore) GetEntry(ctx sdk.Context, fixationKey string, index string, block uint64, entryData codec.ProtoMarshaler, refAction types.ReferenceAction) bool {
	// create entry key
	entryKey := createEntryKey(index, fixationKey, block)

	// get the relevant store
	store := prefix.NewStore(ctx.KVStore(vs.storeKey), types.KeyPrefix(types.EntryKey+fixationKey+index))

	// get the marshled entry
	bz := store.Get(types.KeyPrefix(entryKey))

	// couldn't find entry
	if bz == nil {
		return false
	}

	// unmarshal the entry
	var entry types.Entry
	vs.cdc.MustUnmarshal(bz, &entry)

	// handle ref action
	err := handleRefAction(ctx, &entry, refAction)
	if err != nil {
		return false
	}

	// unmarshal the entry's data
	vs.cdc.MustUnmarshal(entry.GetData(), entryData)

	return true
}

// GetStoreKey returns the versioned store's store key
func (vs VersionedStore) GetStoreKey() sdk.StoreKey {
	return vs.storeKey
}

// GetCdc returns the versioned store's codec
func (vs VersionedStore) GetCdc() codec.BinaryCodec {
	return vs.cdc
}

// getUnmarshaledEntryForBlock gets an entry by block. Block doesn't have to be precise, it gets the closest entry version
func (vs VersionedStore) getUnmarshaledEntryForBlock(ctx sdk.Context, fixationKey string, index string, block uint64, refAction types.ReferenceAction) *types.Entry {
	// get the relevant store using index
	store := prefix.NewStore(ctx.KVStore(vs.storeKey), types.KeyPrefix(types.EntryKey+fixationKey+index))

	// init a reverse iterator
	iterator := sdk.KVStoreReversePrefixIterator(store, []byte{})
	defer iterator.Close()

	// iterate over entries
	for ; iterator.Valid(); iterator.Next() {
		// unmarshal the entry
		var val types.Entry
		vs.cdc.MustUnmarshal(iterator.Value(), &val)

		// if the entry's block is smaller than or equal to the requested block, unmarshal the entry's data
		if val.GetBlock() <= block {
			err := handleRefAction(ctx, &val, refAction)
			if err != nil {
				return nil
			}

			return &val
		}
	}

	return nil
}

// GetEntryForBlock gets an entry by block. Block doesn't have to be precise, it gets the closest entry version
func (vs VersionedStore) GetEntryForBlock(ctx sdk.Context, fixationKey string, index string, block uint64, entryData codec.ProtoMarshaler, refAction types.ReferenceAction) error {
	// get the unmarshaled entry for block
	entry := vs.getUnmarshaledEntryForBlock(ctx, fixationKey, index, block, refAction)
	if entry == nil {
		return utils.LavaError(ctx, ctx.Logger(), "GetEntryForBlock_cant_get_entry", map[string]string{}, "can't get entry")
	}

	// unmarshal the entry's data
	err := vs.cdc.Unmarshal(entry.GetData(), entryData)
	if err != nil {
		return utils.LavaError(ctx, ctx.Logger(), "GetEntryForBlock_cant_unmarshal", map[string]string{}, "can't unmarshal entry data")
	}

	return nil
}

// RemoveEntry removes an entry from the store
func (vs VersionedStore) removeEntry(ctx sdk.Context, fixationKey string, index string, block uint64) {
	// get the relevant store
	store := prefix.NewStore(ctx.KVStore(vs.storeKey), types.KeyPrefix(types.EntryKey+fixationKey+index))

	// create entry's key
	entryKey := createEntryKey(index, fixationKey, block)

	// delete the entry
	store.Delete(types.KeyPrefix(entryKey))
}

// getAllUnmarshaledEntries gets all the unmarshaled entries from the store (without entries' old versions)
func (vs VersionedStore) getAllEntries(ctx sdk.Context, fixationKey string) []*types.Entry {
	latestVersionEntryList := []*types.Entry{}

	uniqueIndices := vs.GetAllEntryIndices(ctx, fixationKey)
	for _, uniqueIndex := range uniqueIndices {
		latestVersionEntry := vs.getUnmarshaledEntryForBlock(ctx, fixationKey, uniqueIndex, uint64(ctx.BlockHeight()), types.DO_NOTHING)
		latestVersionEntryList = append(latestVersionEntryList, latestVersionEntry)
	}

	return latestVersionEntryList
}

// createEntryKey creates an entry key for the KVStore
func createEntryKey(index string, fixationKey string, block uint64) string {
	return types.EntryKey + fixationKey + index + "_" + strconv.FormatUint(block, 10)
}

// NewVersionedStore returns a new versionedStore object
func NewVersionedStore(storeKey sdk.StoreKey, cdc codec.BinaryCodec) *VersionedStore {
	return &VersionedStore{storeKey: storeKey, cdc: cdc}
}
