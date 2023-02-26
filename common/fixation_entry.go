package common

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/utils"
)

// VerionedStore manages lists of entries with versions in the store.
//
// Its primary use it to implemented "fixated entries": entries that may change over
// time, and whose versions must be retained on-chain as long as they are referenced.
// For examples, an older version of a package is needed as long as the subscription
// that uses it lives.
//
// Once instantiated, VersionedStore offers 4 methods:
//    - SetEntry(index, block, *entry): add a new "block" version of an entry "index".
//    - GetEntry(index, block, *entry): get a copy (and reference) an version of an entry.
//    - PutEntry(index, block): drop a reference to a version of an entry.
//    - RemoveEntry(index): mark an entry as unavailable for new GetEntry() calls.
//    - GetAllEntryIndex(index): get all the entries indices (without versions).
//
// How does it work? The explanation below illustrates how the data is stored, assuming the
// user is the module "packages":
//
// 1. When instantiated, VersionedStore gets a `prefix string` - used as a namespace
// to separate between instances of VersionedStore. For instance, module "packages"
// would use its module name for prefix.
//
// 2. Each entry is wrapped by a RawEntry, that holds the index, block, ref-count,
// and the (marshalled) data of the entry.
//
// 3. VersionedStore keeps the entry indices with a special prefix; and it keeps the
// and the RawEntres with a prefix that includes their index (using the block as key).
// For instance, modules "packages" may have a package named "YourPackage" created at
// block 110, and it was updated at block 220, and package named "MyPakcage" created
// at block 150. The store with prefix "packages" will hold the following objects:
//
//     prefix: packages_Entry_Index_            key: MyPackage      data: MyPackage
//     prefix: packages_Entry_Index_            key: YourPackage    data: YourPackage
//     prefix: packages_Entry_Raw_MyPackage     key: 150            data: RawEntry
//     prefix: packages_Entry_Raw_YourPackage   key: 220            data: RawEntry
//     prefix: packages_Entry_Raw_YourPackage   key: 110            data: RawEntry
//
// Thus, iterating on the prefix "packages_Entry_Index_" would yield all the package
// indices. Reverse iterating on the prefix "packages_Entry_Raw_<INDEX>" would yield
// all the versioned of the entry named <INDEX> in descending order.
//
// 4. VersionedStore keeps a reference count of versioned of entries, and when the
// count reaches 0 it marks them for deletion. The actual deletion takes place after
// a fixed number of epochs has passed.

type VersionedStore struct {
	storeKey sdk.StoreKey
	cdc      codec.BinaryCodec
	prefix   string
}

// AppendEntry adds a new entry to the store
func (vs VersionedStore) AppendEntry(ctx sdk.Context, index string, block uint64, entryData codec.ProtoMarshaler) error {
	// get the latest entry for this index
	latestEntry := vs.getUnmarshaledEntryForBlock(ctx, index, block, types.DO_NOTHING)

	// if latest entry is not found, this is a first version entry
	if latestEntry == nil {
		vs.SetEntryIndex(ctx, index)
	} else {
		// make sure the new entry's block is not smaller than the latest entry's block
		if block < latestEntry.GetBlock() {
			return utils.LavaError(ctx, ctx.Logger(), "AppendEntry_block_too_early", map[string]string{"latestEntryBlock": strconv.FormatUint(latestEntry.GetBlock(), 10), "block": strconv.FormatUint(block, 10), "index": index, "vs.prefix": vs.prefix}, "can't append entry, earlier than the latest entry")
		}

		// if the new entry's block is equal to the latest entry, overwrite the latest entry
		if block == latestEntry.GetBlock() {
			return vs.SetEntry(ctx, index, block, entryData)
		}
	}

	// marshal the new entry's data
	b := vs.cdc.MustMarshal(entryData)
	// create a new entry and marshal it
	entry := types.Entry{Index: index, Block: block, Data: b, Refcount: 0}
	bz := vs.cdc.MustMarshal(&entry)

	// get the relevant store
	store := prefix.NewStore(ctx.KVStore(vs.storeKey), types.KeyPrefix(vs.createStoreKey(index)))
	byteKey := types.KeyPrefix(createEntryKey(block))

	// set the new entry to the store
	store.Set(byteKey, bz)

	// delete old entries
	vs.deleteStaleEntries(ctx, index)

	return nil
}

func (vs VersionedStore) deleteStaleEntries(ctx sdk.Context, index string) {
	// get the relevant store and init an iterator
	store := prefix.NewStore(ctx.KVStore(vs.storeKey), types.KeyPrefix(vs.createStoreKey(index)))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	// iterate over entries
	for iterator.Valid() {
		// umarshal the old entry version
		var oldEntry types.Entry
		vs.cdc.MustUnmarshal(iterator.Value(), &oldEntry)

		iterator.Next()

		// skipping removal of latest version
		if !iterator.Valid() {
			break
		}

		// if the entry's refs is equal to 0 and it has been longer than STALE_ENTRY_TIME from its creation, delete it
		if oldEntry.GetRefcount() == 0 && int64(oldEntry.GetBlock())+types.STALE_ENTRY_TIME < ctx.BlockHeight() {
			vs.removeEntry(ctx, oldEntry.GetIndex(), oldEntry.GetBlock())
		} else {
			// else, break (avoiding removal of entries in the middle of the list)
			break
		}
	}
}

// SetEntry sets a specific entry in the store
func (vs VersionedStore) SetEntry(ctx sdk.Context, index string, block uint64, entryData codec.ProtoMarshaler) error {
	// get the relevant store
	store := prefix.NewStore(ctx.KVStore(vs.storeKey), types.KeyPrefix(vs.createStoreKey(index)))
	byteKey := types.KeyPrefix(createEntryKey(block))

	// marshal the new entry data
	b := vs.cdc.MustMarshal(entryData)

	// get the entry from the store
	entry := vs.getUnmarshaledEntryForBlock(ctx, index, block, types.DO_NOTHING)
	if entry == nil {
		return utils.LavaError(ctx, ctx.Logger(), "SetEntry_cant_find_entry", map[string]string{"vs.prefix": vs.prefix, "index": index, "block": strconv.FormatUint(block, 10)}, "can't set non-existent entry")
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
		entry.Refcount += 1
	case types.SUB_REFERENCE:
		if entry.GetRefcount() > 0 {
			entry.Refcount -= 1
		}
	case types.DO_NOTHING:
	}

	return nil
}

// GetStoreKey returns the versioned store's store key
func (vs VersionedStore) GetStoreKey() sdk.StoreKey {
	return vs.storeKey
}

// GetCdc returns the versioned store's codec
func (vs VersionedStore) GetCdc() codec.BinaryCodec {
	return vs.cdc
}

// Getprefix returns the versioned store's fixation key
func (vs VersionedStore) GetPrefix() string {
	return vs.prefix
}

// Setprefix sets the versioned store's fixation key
func (vs VersionedStore) SetPrefix(prefix string) VersionedStore {
	vs.prefix = prefix
	return vs
}

// getUnmarshaledEntryForBlock gets an entry by block. Block doesn't have to be precise, it gets the closest entry version
func (vs VersionedStore) getUnmarshaledEntryForBlock(ctx sdk.Context, index string, block uint64, refAction types.ReferenceAction) *types.Entry {
	// get the relevant store using index
	store := prefix.NewStore(ctx.KVStore(vs.storeKey), types.KeyPrefix(vs.createStoreKey(index)))

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
func (vs VersionedStore) GetEntry(ctx sdk.Context, index string, block uint64, entryData codec.ProtoMarshaler, refAction types.ReferenceAction) error {
	// get the unmarshaled entry for block
	entry := vs.getUnmarshaledEntryForBlock(ctx, index, block, refAction)
	if entry == nil {
		return utils.LavaError(ctx, ctx.Logger(), "GetEntry_cant_get_entry", map[string]string{}, "can't get entry")
	}

	// unmarshal the entry's data
	err := vs.cdc.Unmarshal(entry.GetData(), entryData)
	if err != nil {
		return utils.LavaError(ctx, ctx.Logger(), "GetEntry_cant_unmarshal", map[string]string{}, "can't unmarshal entry data")
	}

	return nil
}

// RemoveEntry removes an entry from the store
func (vs VersionedStore) removeEntry(ctx sdk.Context, index string, block uint64) {
	// get the relevant store
	store := prefix.NewStore(ctx.KVStore(vs.storeKey), types.KeyPrefix(vs.createStoreKey(index)))

	// create entry's key
	entryKey := createEntryKey(block)

	// delete the entry
	store.Delete(types.KeyPrefix(entryKey))
}

// getAllUnmarshaledEntries gets all the unmarshaled entries from the store (without entries' old versions)
func (vs VersionedStore) getAllEntries(ctx sdk.Context) []*types.Entry {
	latestVersionEntryList := []*types.Entry{}

	uniqueIndices := vs.GetAllEntryIndices(ctx)
	for _, uniqueIndex := range uniqueIndices {
		latestVersionEntry := vs.getUnmarshaledEntryForBlock(ctx, uniqueIndex, uint64(ctx.BlockHeight()), types.DO_NOTHING)
		latestVersionEntryList = append(latestVersionEntryList, latestVersionEntry)
	}

	return latestVersionEntryList
}

// createEntryKey creates an entry key for the KVStore
func createEntryKey(block uint64) string {
	return strconv.FormatUint(block, 10)
}

func (vs VersionedStore) createStoreKey(index string) string {
	return types.EntryKey + vs.prefix + index
}

// NewVersionedStore returns a new versionedStore object
func NewVersionedStore(storeKey sdk.StoreKey, cdc codec.BinaryCodec, prefix string) *VersionedStore {
	return &VersionedStore{storeKey: storeKey, cdc: cdc, prefix: prefix}
}
