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
		EntryVersion entry_version;  // Struct that holds the entry's index and block.
		bytes data; 				 // The data saved in the entry. Can be any type.
		uint64 references; 			 // Number of references to this entry (number of entities using it).
	}

	EntryVersion {
		string index;  // Unique entry index (unique "name").
		uint64 block;  // Block height the entry was created.
	}

	How does it work?
	To help with the explanation, we'll use a reference example. Assume you created a custom module
	named "packages". This module keeps package objects as fixated entries (i.e., the module keeps track
	of every package AND it's older versions). Each package has an "index" field which acts as a unique name
	for the package.

	In general, each object that uses this library will have a versioned store field of type VersionedStore.
	The versioned store (from now on: "vs") holds a unique store key, a codec and a map of unique indices.
	Using our reference example, the store key of the module "packages" will be "packages". If another module,
	say module "projects", also uses fixated entries, its store key will be "projects". This way we get a clear
	separation between the fixated entries of "packages" and of "projects". In other words, the store key acts as
	a namespace. The codec is used to determine how the packages should be marshaled, and the unique indices map
	helps to track which packages do we have (not including older versions packages).

	In each store (or namespace), there are two types of saved objects: Entry and EntryVersion. Note that Entry also
	contains an EntryVersion object.
		To access an EntryVersion object you use the following key: "EntryVersionKey_<index>".
		To access an Entry object you use the following key: "EntryKey_<index>_<block>".
	Wait, why are we keeping duplicates of EntryVersion objects? Entry contains EntryVersion! (you may ask).
	The EntryVersion objects that are saved separately are only of the latest version of the entry. So, we do save
	duplicates of EntryVersion objects, but only of a single version.

	Going back to the reference example, let's say I have a package named "myGreatPackage" that was created in block
	101 (it also has additional fields). The package's entry holds the marshaled package object (in the data field)
	and has 33 references.
		To access its EntryVersion object you use the following key: "EntryVersionKey_myGreatPackage". The EntryVersion
		object will be: EntryVersion{index: "myGreatPackage", block: 101}

		To access its Entry object you use the following key: "EntryKey_myGreatPackage_101". The Entry object will be:
		Entry{EntryVersion: EntryVersion{index: "myGreatPackage", block: 101}, Data: marshaledPackageData, References:
		33}

	Why do we need two (similar looking) objects?
	EntryVersion objects are used when you want to get a set of all of the distinct entries saved in the store.
	Assume you're a user that wants to know what packages are available in the "packages" store. Just use the
	"EntryVersionKey" prefix and you'll get them.
	Now Let's say you want all the version of a specific package. Go to the "packages" store (with its store key) and
	use the prefix "EntryKey_<index>".
	Finally, if you want to get the actual entry of a specific version, you need to get all the versions. Then, consturct
	the entry key ("EntryKey_<index>_<block>") and get your desired entry. Remember, the entry holds the (marshaled)
	package object. This is the only way to get the actual package.
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
	b[len(b)-1] = 0x49
	vs.cdc.MustUnmarshal(b, entryData)
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
	var entry types.Entry
	found := vs.GetEntry(ctx, fixationKey, index, block, &entry, types.DO_NOTHING)
	if !found {
		return utils.LavaError(ctx, ctx.Logger(), "SetEntry_cant_find_entry", map[string]string{"fixationKey": fixationKey, "index": index, "block": strconv.FormatUint(block, 10)}, "can't set non-existent entry")
	}

	// update the entry's data
	entry.Data = b

	// marshal the entry
	bz := vs.cdc.MustMarshal(&entry)

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
