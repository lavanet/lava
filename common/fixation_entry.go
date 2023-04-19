package common

import (
	"math"
	"strconv"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/utils"
)

// FixationStore manages lists of entries with versions in the store.
// (See also documentation in common/fixation_entry_index.go)
//
// Its primary use it to implement "fixated entries": entries that may change over
// time, and whose versions must be retained on-chain as long as they are referenced.
// For examples, an older version of a package is needed as long as the subscription
// that uses it lives.
//
// Once instantiated with NewFixationStore(), it offers the following methods:
//    - AppendEntry(index, block, *entry): add a new "block" version of an entry "index".
//    - ModifyEntry(index, block, *entry): modify existing entry with "index" and "block"
//    - FindEntry(index, block, *entry): get a copy (no reference) of a version of an entry
//    - GetEntry(index, *entry): get a copy (and reference) of the latest version of an entry
//    - PutEntry(index, block): drop a reference of a version of an entry
//    - [TBD] RemoveEntry(index): mark an entry as unavailable for new GetEntry() calls
//    - GetAllEntryIndices(): get all the entries indices (without versions)
//    - GetAllEntryVersions(index): get all the versions of an entry (for testing)
//    - AdvanceBlock(): notify of block progress (e.g. BeginBlock) for garbage collection
//
// Entry names (index) must contain only visible ascii characters (ascii values 32-125).
// The ascii 'DEL' invisible character is used internally to terminate the index values
// when stored, to ensure that no two indices can ever overlap, i.e. one being the prefix
// of the other.
// (Note that this properly supports Bech32 addresses, which are limited to use only
// visible ascii characters as per https://en.bitcoin.it/wiki/BIP_0173#Specification).
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
	tstore   TimerStore
}

const (
	ASCII_MIN = 32  // min visible ascii
	ASCII_MAX = 126 // max visible ascii
	ASCII_DEL = 127 // ascii for DEL
)

// sanitizeIdnex checks that a string contains only visible ascii characters
// (i.e. Ascii 32-126), and appends a (ascii) DEL to the index; this ensures
// that an index can never be a prefix of another index.
func sanitizeIndex(index string) (string, error) {
	for i := 0; i < len(index); i++ {
		if index[i] < ASCII_MIN || index[i] > ASCII_MAX {
			return index, types.ErrInvalidIndex
		}
	}
	return index + string([]byte{ASCII_DEL}), nil
}

// desantizeIndex reverts the effect of sanitizeIndex - removes the trailing
// (ascii) DEL terminator.
func desanitizeIndex(safeIndex string) string {
	return safeIndex[0 : len(safeIndex)-1]
}

func (fs *FixationStore) assertSanitizedIndex(safeIndex string) {
	if []byte(safeIndex)[len(safeIndex)-1] != ASCII_DEL {
		panic("Fixation: prefix " + fs.prefix + ": unsanitized safeIndex: " + safeIndex)
	}
}

func (fs *FixationStore) getStore(ctx sdk.Context, index string) *prefix.Store {
	store := prefix.NewStore(
		ctx.KVStore(fs.storeKey),
		types.KeyPrefix(fs.createStoreKey(index)))
	return &store
}

// setEntry modifies an existing entry in the store
func (fs *FixationStore) setEntry(ctx sdk.Context, entry types.Entry) {
	store := fs.getStore(ctx, entry.Index)
	byteKey := types.EncodeKey(entry.Block)
	marshaledEntry := fs.cdc.MustMarshal(&entry)
	store.Set(byteKey, marshaledEntry)
}

// AppendEntry adds a new entry to the store
func (fs *FixationStore) AppendEntry(
	ctx sdk.Context,
	index string,
	block uint64,
	entryData codec.ProtoMarshaler,
) error {
	safeIndex, err := sanitizeIndex(index)
	if err != nil {
		details := map[string]string{"index": index}
		return utils.LavaError(ctx, ctx.Logger(), "AppendEntry_invalid_index", details, "invalid non-ascii entry")
	}

	latestEntry, found := fs.getUnmarshaledEntryForBlock(ctx, safeIndex, block)

	// if latest entry is not found, this is a first version entry
	if !found {
		fs.setEntryIndex(ctx, safeIndex)
	} else {
		// make sure the new entry's block is not smaller than the latest entry's block
		if block < latestEntry.Block {
			details := map[string]string{
				"latestBlock": strconv.FormatUint(latestEntry.GetBlock(), 10),
				"block":       strconv.FormatUint(block, 10),
				"index":       index,
				"fs.prefix":   fs.prefix,
			}
			return utils.LavaError(ctx, ctx.Logger(), "AppendEntry_block_too_early", details, "entry block earlier than latest entry")
		}

		// if the new entry's block is equal to the latest entry, overwrite the latest entry
		if block == latestEntry.Block {
			return fs.ModifyEntry(ctx, index, block, entryData)
		}

		fs.putEntry(ctx, latestEntry)
	}

	// marshal the new entry's data
	marshaledEntryData := fs.cdc.MustMarshal(entryData)

	// create a new entry and marshal it
	entry := types.Entry{
		Index:    safeIndex,
		Block:    block,
		StaleAt:  math.MaxUint64,
		Data:     marshaledEntryData,
		Refcount: 1,
	}

	fs.setEntry(ctx, entry)
	return nil
}

func (fs *FixationStore) deleteStaleEntries(ctx sdk.Context, safeIndex string) {
	fs.assertSanitizedIndex(safeIndex)
	store := fs.getStore(ctx, safeIndex)

	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	// "stale" entry versions are ones that reached refcount zero at least
	// STALE_TIME blocks ago; they are not visible in lookups, hence may be
	// discarded. specifically, a stale entry version becomes "elgibile for
	// removal" , if either it is:
	//   one that follows a stale entry version, -OR-
	//   the oldest entry version
	// rationale: entries are generally valid from their block time until
	// the block time of the following newer entry. this newer entry marks
	// the end of the previous entry, and hence may not be removed until
	// that previous entry gets discarded. keeping the stale entry versions
	// ensures (future FindEntry) that blocks from that entry onward are
	// stale (otherwise, a lookup might resolve successfully with an older
	// non-stale entry version). For this, one - the oldest - marker is
	// enough, and additional younger markers can be discarded.
	// for example, consider this situation with versions A through E:
	//   A(stale), B, C(stale), D(stale), E
	// in this case, A can be discarded because it is the oldest. C cannot
	// be discarded because it marks that new blocks are stale (while older
	// blocks between B and C map to B). D is unneeded as marker because C
	// is already there, and can be discarded too.

	var removals []uint64
	safeToDeleteEntry := true // if oldest, or if previous entry was stale
	safeToDeleteIndex := true // if non of the entry versions was skipped

	for ; iterator.Valid(); iterator.Next() {
		// umarshal the old entry version
		var entry types.Entry
		fs.cdc.MustUnmarshal(iterator.Value(), &entry)

		if !entry.IsStale(ctx) {
			safeToDeleteEntry = false
			safeToDeleteIndex = false
			continue
		}

		if !safeToDeleteEntry {
			safeToDeleteEntry = true
			safeToDeleteIndex = false
			continue
		}

		removals = append(removals, entry.Block)
	}

	for _, block := range removals {
		fs.removeEntry(ctx, safeIndex, block)
	}

	if safeToDeleteIndex {
		// non was skipped - so all were removed: delete the entry index
		fs.removeEntryIndex(ctx, safeIndex)
	}
}

// ModifyEntry modifies an existing entry in the store
func (fs *FixationStore) ModifyEntry(ctx sdk.Context, index string, block uint64, entryData codec.ProtoMarshaler) error {
	safeIndex, err := sanitizeIndex(index)
	if err != nil {
		details := map[string]string{"index": index}
		return utils.LavaError(ctx, ctx.Logger(), "ModifyEntry_invalid_index", details, "invalid non-ascii entry")
	}

	// get the entry from the store
	entry, found := fs.getUnmarshaledEntryForBlock(ctx, safeIndex, block)
	if !found {
		details := map[string]string{
			"fs.prefix": fs.prefix,
			"index":     index,
			"block":     strconv.FormatUint(block, 10),
		}
		return utils.LavaError(ctx, ctx.Logger(), "SetEntry_cant_find_entry", details, "entry does not exist")
	}

	// update entry data
	marshaledEntryData := fs.cdc.MustMarshal(entryData)
	entry.Data = marshaledEntryData

	fs.setEntry(ctx, entry)
	return nil
}

// getUnmarshaledEntryForBlock gets an entry version for an index that has
// nearest-smaller block version for the given block arg.
func (fs *FixationStore) getUnmarshaledEntryForBlock(ctx sdk.Context, safeIndex string, block uint64) (types.Entry, bool) {
	fs.assertSanitizedIndex(safeIndex)
	store := fs.getStore(ctx, safeIndex)

	// init a reverse iterator
	iterator := sdk.KVStoreReversePrefixIterator(store, []byte{})
	defer iterator.Close()

	// iterate over entries in reverse order of version (block), and return the
	// first version that is smaller-or-equal to the requested block. Thus, the
	// caller gets the latest version at the time of the block (arg).
	// For example, assuming two entries for blocks 100 and 200, asking for the
	// entry with block 199 would return the entry with block 100, which was in
	// effect at the time of block 199.

	for ; iterator.Valid(); iterator.Next() {
		var entry types.Entry
		fs.cdc.MustUnmarshal(iterator.Value(), &entry)

		if entry.Block <= block {
			if entry.IsStale(ctx) {
				break
			}

			return entry, true
		}
	}
	return types.Entry{}, false
}

// FindEntry returns the entry by index and block without changing the refcount
func (fs *FixationStore) FindEntry(ctx sdk.Context, index string, block uint64, entryData codec.ProtoMarshaler) bool {
	safeIndex, err := sanitizeIndex(index)
	if err != nil {
		details := map[string]string{"index": index}
		utils.LavaError(ctx, ctx.Logger(), "FindEntry_invalid_index", details, "invalid non-ascii entry")
		return false
	}

	entry, found := fs.getUnmarshaledEntryForBlock(ctx, safeIndex, block)
	if !found {
		return false
	}

	fs.cdc.MustUnmarshal(entry.GetData(), entryData)
	return true
}

// GetEntry returns the latest entry by index and increments the refcount
func (fs *FixationStore) GetEntry(ctx sdk.Context, index string, entryData codec.ProtoMarshaler) bool {
	safeIndex, err := sanitizeIndex(index)
	if err != nil {
		details := map[string]string{"index": index}
		utils.LavaError(ctx, ctx.Logger(), "GetEntry_invalid_index", details, "invalid non-ascii entry")
		return false
	}

	block := uint64(ctx.BlockHeight())

	entry, found := fs.getUnmarshaledEntryForBlock(ctx, safeIndex, block)
	if !found {
		return false
	}

	fs.cdc.MustUnmarshal(entry.GetData(), entryData)

	entry.Refcount += 1
	fs.setEntry(ctx, entry)
	return true
}

// putEntry decrements the refcount of an entry and marks for staleness if needed
func (fs *FixationStore) putEntry(ctx sdk.Context, entry types.Entry) {
	if entry.Refcount == 0 {
		panic("Fixation: prefix " + fs.prefix + ": negative refcount safeIndex: " + entry.Index)
	}

	entry.Refcount -= 1

	if entry.Refcount == 0 {
		// never overflows because ctx.BlockHeight is int64
		entry.StaleAt = uint64(ctx.BlockHeight()) + uint64(types.STALE_ENTRY_TIME)
		fs.tstore.AddTimerByBlockHeight(ctx, entry.StaleAt, entry.Index)
	}

	fs.setEntry(ctx, entry)
}

// PutEntry finds the entry by index and block and decrements the refcount
func (fs *FixationStore) PutEntry(ctx sdk.Context, index string, block uint64) {
	safeIndex, err := sanitizeIndex(index)
	if err != nil {
		panic("PutEntry with non-ascii index: " + index)
	}

	entry, found := fs.getUnmarshaledEntryForBlock(ctx, safeIndex, block)
	if !found {
		panic("PutEntry with unknown index: " + index)
	}

	if entry.Block != block {
		panic("PutEntry with block mismatch index: " + index +
			" got " + strconv.Itoa(int(entry.Block)) + " expected " + strconv.Itoa(int(block)))
	}

	fs.putEntry(ctx, entry)
}

// removeEntry removes an entry from the store
func (fs *FixationStore) removeEntry(ctx sdk.Context, index string, block uint64) {
	store := fs.getStore(ctx, index)
	store.Delete(types.EncodeKey(block))
}

func (fs *FixationStore) createStoreKey(index string) string {
	return types.EntryPrefix + fs.prefix + index
}

func (fs *FixationStore) AdvanceBlock(ctx sdk.Context) {
	fs.tstore.Tick(ctx)
}

func (fs *FixationStore) getVersion(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(fs.storeKey), types.KeyPrefix(fs.prefix))

	b := store.Get(types.KeyPrefix(types.FixationVersionKey))
	if b == nil {
		return 1
	}

	return types.DecodeKey(b)
}

func (fs *FixationStore) setVersion(ctx sdk.Context, val uint64) {
	store := prefix.NewStore(ctx.KVStore(fs.storeKey), types.KeyPrefix(fs.prefix))
	b := types.EncodeKey(val)
	store.Set(types.KeyPrefix(types.FixationVersionKey), b)
}

// NewFixationStore returns a new FixationStore object
func NewFixationStore(storeKey sdk.StoreKey, cdc codec.BinaryCodec, prefix string) *FixationStore {
	fs := FixationStore{storeKey: storeKey, cdc: cdc, prefix: prefix}

	callback := func(ctx sdk.Context, data string) { fs.deleteStaleEntries(ctx, data) }
	tstore := NewTimerStore(storeKey, cdc, prefix).WithCallbackByBlockHeight(callback)
	fs.tstore = *tstore

	return &fs
}
