package common

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/utils"
)

// FixationStore manages lists of entries with versions in the store.
//
// Its primary use it to implemented "fixated entries": entries that may change over
// time, and whose versions must be retained on-chain as long as they are referenced.
// For examples, an older version of a package is needed as long as the subscription
// that uses it lives.
//
// Once instantiated, FixationStore offers 4 methods:
//    - AppendEntry(index, block, *entry): add a new "block" version of an entry "index".
//    - ModifyEntry(index, block, *entry): modify existing entry with "index" and "block"
//    - GetEntry(index, block, *entry, refAction): get a copy (and reference) an version of an entry. Also, apply refAction (add/sub refCount)
//    - RemoveEntry(index): mark an entry as unavailable for new GetEntry() calls.
//    - GetAllEntryIndex(index): get all the entries indices (without versions).
//    - SetEntryIndex(index): add a new index to the index list (indices without versions).
//
// How does it work? The explanation below illustrates how the data is stored, assuming the
// user is the module "packages":
//
// 1. When instantiated, FixationStore gets a `prefix string` - used as a namespace
// to separate between instances of FixationStore. For instance, module "packages"
// would use its module name for prefix.
//
// 2. Each entry is wrapped by a RawEntry, that holds the index, block, ref-count,
// and the (marshalled) data of the entry.
//
// 3. FixationStore keeps the entry indices with a special prefix; and it keeps the
// and the RawEntres with a prefix that includes their index (using the block as key).
// For instance, modules "packages" may have a package named "YourPackage" created at
// block 110, and it was updated at block 220, and package named "MyPakcage" created
// at block 150. The store with prefix "packages" will hold the following objects:
//
//     prefix: packages_Entry_Index_            key: MyPackage      data: MyPackage
//     prefix: packages_Entry_Index_            key: YourPackage    data: YourPackage
//     prefix: packages_Entry_Value_MyPackage     key: 150            data: Entry
//     prefix: packages_Entry_Value_YourPackage   key: 220            data: Entry
//     prefix: packages_Entry_Value_YourPackage   key: 110            data: Entry
//
// Thus, iterating on the prefix "packages_Entry_Index_" would yield all the package
// indices. Reverse iterating on the prefix "packages_Entry_Raw_<INDEX>" would yield
// all the Fixation of the entry named <INDEX> in descending order.
//
// 4. FixationStore keeps a reference count of Fixation of entries, and when the
// count reaches 0 it marks them for deletion. The actual deletion takes place after
// a fixed number of epochs has passed.

type FixationStore struct {
	storeKey sdk.StoreKey
	cdc      codec.BinaryCodec
	prefix   string
}

// AppendEntry adds a new entry to the store
func (fs *FixationStore) AppendEntry(ctx sdk.Context, index string, block uint64, entryData codec.ProtoMarshaler) error {
	// get the latest entry for this index
	latestEntry, found := fs.getUnmarshaledEntryForBlock(ctx, index, block)

	// if latest entry is not found, this is a first version entry
	if !found {
		fs.SetEntryIndex(ctx, index)
	} else {
		// make sure the new entry's block is not smaller than the latest entry's block
		if block < latestEntry.GetBlock() {
			return utils.LavaError(ctx, ctx.Logger(), "AppendEntry_block_too_early", map[string]string{"latestEntryBlock": strconv.FormatUint(latestEntry.GetBlock(), 10), "block": strconv.FormatUint(block, 10), "index": index, "fs.prefix": fs.prefix}, "can't append entry, earlier than the latest entry")
		}

		// if the new entry's block is equal to the latest entry, overwrite the latest entry
		if block == latestEntry.GetBlock() {
			return fs.ModifyEntry(ctx, index, block, entryData)
		}
	}

	// marshal the new entry's data
	marshaledEntryData := fs.cdc.MustMarshal(entryData)
	// create a new entry and marshal it
	entry := types.Entry{Index: index, Block: block, Data: marshaledEntryData, Refcount: 0}
	marshaledEntry := fs.cdc.MustMarshal(&entry)

	// get the relevant store
	store := prefix.NewStore(ctx.KVStore(fs.storeKey), types.KeyPrefix(fs.createStoreKey(index)))
	byteKey := types.EncodeKey(block)

	// set the new entry to the store
	store.Set(byteKey, marshaledEntry)

	// delete old entries
	fs.deleteStaleEntries(ctx, index)

	return nil
}

func (fs *FixationStore) deleteStaleEntries(ctx sdk.Context, index string) {
	// get the relevant store and init an iterator
	store := prefix.NewStore(ctx.KVStore(fs.storeKey), types.KeyPrefix(fs.createStoreKey(index)))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	// iterate over entries
	for iterator.Valid() {
		// umarshal the old entry version
		var oldEntry types.Entry
		fs.cdc.MustUnmarshal(iterator.Value(), &oldEntry)

		iterator.Next()

		// skipping removal of latest version
		if !iterator.Valid() {
			break
		}

		// if the entry's refs is equal to 0 and it has been longer than STALE_ENTRY_TIME from its creation, delete it
		if oldEntry.GetRefcount() == 0 && int64(oldEntry.GetBlock())+types.STALE_ENTRY_TIME < ctx.BlockHeight() {
			fs.removeEntry(ctx, oldEntry.GetIndex(), oldEntry.GetBlock())
		} else {
			// else, break (avoiding removal of entries in the middle of the list. because it would
			// break future lookup that may involve that (stale) entry.)
			break
		}
	}
}

// ModifyEntry modifies an exisiting entry in the store
func (fs *FixationStore) ModifyEntry(ctx sdk.Context, index string, block uint64, entryData codec.ProtoMarshaler) error {
	// get the relevant store
	store := prefix.NewStore(ctx.KVStore(fs.storeKey), types.KeyPrefix(fs.createStoreKey(index)))
	byteKey := types.EncodeKey(block)

	// marshal the new entry data
	marshaledEntryData := fs.cdc.MustMarshal(entryData)

	// get the entry from the store
	entry, found := fs.getUnmarshaledEntryForBlock(ctx, index, block)
	if !found {
		return utils.LavaError(ctx, ctx.Logger(), "SetEntry_cant_find_entry", map[string]string{"fs.prefix": fs.prefix, "index": index, "block": strconv.FormatUint(block, 10)}, "can't set non-existent entry")
	}

	// update the entry's data
	entry.Data = marshaledEntryData

	// marshal the entry
	marshaledEntry := fs.cdc.MustMarshal(&entry)

	// set the entry
	store.Set(byteKey, marshaledEntry)

	return nil
}

// GetStoreKey returns the Fixation store's store key
func (fs *FixationStore) GetStoreKey() sdk.StoreKey {
	return fs.storeKey
}

// GetCdc returns the Fixation store's codec
func (fs *FixationStore) GetCdc() codec.BinaryCodec {
	return fs.cdc
}

// Getprefix returns the Fixation store's fixation key
func (fs *FixationStore) GetPrefix() string {
	return fs.prefix
}

// getUnmarshaledEntryForBlock gets an entry by block. Block doesn't have to be precise, it gets the closest entry version
func (fs *FixationStore) getUnmarshaledEntryForBlock(ctx sdk.Context, index string, block uint64) (types.Entry, bool) {
	// get the relevant store using index
	store := prefix.NewStore(ctx.KVStore(fs.storeKey), types.KeyPrefix(fs.createStoreKey(index)))

	// init a reverse iterator
	iterator := sdk.KVStoreReversePrefixIterator(store, []byte{})
	defer iterator.Close()

	// iterate over entries
	for ; iterator.Valid(); iterator.Next() {
		// unmarshal the entry
		var entry types.Entry
		fs.cdc.MustUnmarshal(iterator.Value(), &entry)

		// if the entry's block is smaller than or equal to the requested block, unmarshal the entry's data
		// since the user might not know the precise block that the entry was created on, we get the closest one
		// we get from the past since this was the entry that was valid in the requested block (assume there's an
		// entry in block 100 and block 200 and the user asks for the version of block 199. Since the block 200's
		// didn't exist yet, we get the entry from block 100)
		if entry.GetBlock() <= block {
			return entry, true
		}
	}
	return types.Entry{}, false
}

// get entry with index and block without influencing the entry's refs
func (fs *FixationStore) FindEntry(ctx sdk.Context, index string, block uint64, entryData codec.ProtoMarshaler) (error, bool) {
	// get the unmarshaled entry for block
	entry, found := fs.getUnmarshaledEntryForBlock(ctx, index, block)
	if !found {
		return types.ErrEntryNotFound, false
	}

	// unmarshal the entry's data
	err := fs.cdc.Unmarshal(entry.GetData(), entryData)
	if err != nil {
		return utils.LavaError(ctx, ctx.Logger(), "GetEntry_cant_unmarshal", map[string]string{"err": err.Error()}, "can't unmarshal entry data"), false
	}

	return nil, true
}

// get entry with index and block with ref increase
func (fs *FixationStore) GetEntry(ctx sdk.Context, index string, block uint64, entryData codec.ProtoMarshaler) (error, bool) {
	// get the unmarshaled entry for block
	entry, found := fs.getUnmarshaledEntryForBlock(ctx, index, block)
	if !found {
		return types.ErrEntryNotFound, false
	}

	// unmarshal the entry's data
	err := fs.cdc.Unmarshal(entry.GetData(), entryData)
	if err != nil {
		return utils.LavaError(ctx, ctx.Logger(), "GetEntry_cant_unmarshal", map[string]string{"err": err.Error()}, "can't unmarshal entry data"), false
	}

	entry.Refcount += 1

	// get the relevant byte
	store := prefix.NewStore(ctx.KVStore(fs.storeKey), types.KeyPrefix(fs.createStoreKey(index)))
	byteKey := types.EncodeKey(block)

	// marshal the entry
	marshaledEntry := fs.cdc.MustMarshal(&entry)

	// set the entry
	store.Set(byteKey, marshaledEntry)

	return nil, true
}

// get entry with index and block with ref decrease
func (fs *FixationStore) PutEntry(ctx sdk.Context, index string, block uint64, entryData codec.ProtoMarshaler) (error, bool) {
	// get the unmarshaled entry for block
	entry, found := fs.getUnmarshaledEntryForBlock(ctx, index, block)
	if !found {
		return types.ErrEntryNotFound, false
	}

	// unmarshal the entry's data
	err := fs.cdc.Unmarshal(entry.GetData(), entryData)
	if err != nil {
		return utils.LavaError(ctx, ctx.Logger(), "GetEntry_cant_unmarshal", map[string]string{"err": err.Error()}, "can't unmarshal entry data"), false
	}

	if entry.GetRefcount() > 0 {
		entry.Refcount -= 1
	} else {
		return utils.LavaError(ctx, ctx.Logger(), "handleRefAction_sub_ref_from_non_positive_count", map[string]string{"refCount": strconv.FormatUint(entry.GetRefcount(), 10)}, "refCount is not larger than zero. Can't subtract refcount"), false
	}

	// get the relevant byte
	store := prefix.NewStore(ctx.KVStore(fs.storeKey), types.KeyPrefix(fs.createStoreKey(index)))
	byteKey := types.EncodeKey(block)

	// marshal the entry
	marshaledEntry := fs.cdc.MustMarshal(&entry)

	// set the entry
	store.Set(byteKey, marshaledEntry)

	return nil, true
}

// RemoveEntry removes an entry from the store
func (fs *FixationStore) removeEntry(ctx sdk.Context, index string, block uint64) {
	// get the relevant store
	store := prefix.NewStore(ctx.KVStore(fs.storeKey), types.KeyPrefix(fs.createStoreKey(index)))

	// delete the entry
	store.Delete(types.EncodeKey(block))
}

func (fs *FixationStore) createStoreKey(index string) string {
	return types.EntryKey + fs.prefix + index
}

// NewFixationStore returns a new FixationStore object
func NewFixationStore(storeKey sdk.StoreKey, cdc codec.BinaryCodec, prefix string) *FixationStore {
	fs := FixationStore{storeKey: storeKey, cdc: cdc, prefix: prefix}
	return &fs
}
