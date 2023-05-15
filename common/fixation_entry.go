package common

import (
	"fmt"
	"math"

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
//    - ModifyEntry(index, block, *entry): modify an existing entry with "index" and exact "block" (*)
//    - ReadEntry(index, block, *entry): copy an existing entry with "index" and exact "block" (*)
//    - FindEntry(index, block, *entry): get a copy (no reference) of a version of an entry (**)
//    - GetEntry(index, *entry): get a copy (and reference) of the latest version of an entry
//    - PutEntry(index, block): drop reference to an existing entry with "index" and exact "block" (*)
//    - [TBD] RemoveEntry(index): mark an entry as unavailable for new GetEntry() calls
//    - GetAllEntryIndices(): get all the entries indices (without versions)
//    - GetAllEntryVersions(index): get all the versions of an entry (for testing)
//    - GetEntryVersionsRange(index, block, delta): get range of entry versions (**)
//    - AdvanceBlock(): notify of block progress (e.g. BeginBlock) for garbage collection
// Note:
//    - methods marked with (*) expect an exact existing method, or otherwise will panic
//    - methods marked with (**) will match an entry with the nearest-smaller block version
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

func (fs *FixationStore) getEntryStore(ctx sdk.Context, index string) *prefix.Store {
	store := prefix.NewStore(
		ctx.KVStore(fs.storeKey),
		types.KeyPrefix(fs.createEntryStoreKey(index)))
	return &store
}

// getEntry returns an existing entry in the store
func (fs *FixationStore) getEntry(ctx sdk.Context, safeIndex string, block uint64) (entry types.Entry) {
	store := fs.getEntryStore(ctx, safeIndex)
	byteKey := types.EncodeKey(block)
	b := store.Get(byteKey)
	if b == nil {
		panic(fmt.Sprintf("getEntry: unknown entry: %s block: %d", types.DesanitizeIndex(safeIndex), block))
	}
	fs.cdc.MustUnmarshal(b, &entry)
	return entry
}

// setEntry modifies an existing entry in the store
func (fs *FixationStore) setEntry(ctx sdk.Context, entry types.Entry) {
	store := fs.getEntryStore(ctx, entry.Index)
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
	safeIndex, err := types.SanitizeIndex(index)
	if err != nil {
		return utils.LavaFormatError("AppendEntry failed", err,
			utils.Attribute{Key: "index", Value: index},
		)
	}

	latestEntry, found := fs.getUnmarshaledEntryForBlock(ctx, safeIndex, block)

	// if latest entry is not found, this is a first version entry
	if !found {
		fs.setEntryIndex(ctx, safeIndex)
	} else {
		// make sure the new entry's block is not smaller than the latest entry's block
		if block < latestEntry.Block {
			return utils.LavaFormatError("entry block earlier than latest entry", err,
				utils.Attribute{Key: "index", Value: index},
				utils.Attribute{Key: "block", Value: block},
				utils.Attribute{Key: "fs.prefix", Value: fs.prefix},
				utils.Attribute{Key: "latestBlock", Value: latestEntry.Block},
			)
		}

		// if the new entry's block is equal to the latest entry, overwrite the latest entry
		if block == latestEntry.Block {
			fs.ModifyEntry(ctx, index, block, entryData)
			return nil
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
	types.AssertSanitizedIndex(safeIndex, fs.prefix)
	store := fs.getEntryStore(ctx, safeIndex)

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

// ReadEntry returns and existing entry with index and specific block
func (fs *FixationStore) ReadEntry(ctx sdk.Context, index string, block uint64, entryData codec.ProtoMarshaler) {
	safeIndex, err := types.SanitizeIndex(index)
	if err != nil {
		panic("ReadEntry invalid non-ascii entry: " + index)
	}

	entry := fs.getEntry(ctx, safeIndex, block)
	fs.cdc.MustUnmarshal(entry.GetData(), entryData)
}

// ModifyEntry modifies an existing entry in the store
func (fs *FixationStore) ModifyEntry(ctx sdk.Context, index string, block uint64, entryData codec.ProtoMarshaler) {
	safeIndex, err := types.SanitizeIndex(index)
	if err != nil {
		panic("ModifyEntry with non-ascii index: " + index)
	}

	entry := fs.getEntry(ctx, safeIndex, block)
	entry.Data = fs.cdc.MustMarshal(entryData)
	fs.setEntry(ctx, entry)
}

// getUnmarshaledEntryForBlock gets an entry version for an index that has
// nearest-smaller block version for the given block arg.
func (fs *FixationStore) getUnmarshaledEntryForBlock(ctx sdk.Context, safeIndex string, block uint64) (types.Entry, bool) {
	types.AssertSanitizedIndex(safeIndex, fs.prefix)
	store := fs.getEntryStore(ctx, safeIndex)

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
	safeIndex, err := types.SanitizeIndex(index)
	if err != nil {
		utils.LavaFormatError("FindEntry failed", err,
			utils.Attribute{Key: "index", Value: index},
		)
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
	safeIndex, err := types.SanitizeIndex(index)
	if err != nil {
		utils.LavaFormatError("GetEntry failed", err,
			utils.Attribute{Key: "index", Value: index},
		)
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
	safeIndex, err := types.SanitizeIndex(index)
	if err != nil {
		panic("PutEntry invalid non-ascii entry: " + index)
	}

	entry := fs.getEntry(ctx, safeIndex, block)
	fs.putEntry(ctx, entry)
}

// removeEntry removes an entry from the store
func (fs *FixationStore) removeEntry(ctx sdk.Context, index string, block uint64) {
	store := fs.getEntryStore(ctx, index)
	store.Delete(types.EncodeKey(block))
}

func (fs *FixationStore) getEntryVersionsFilter(ctx sdk.Context, index string, block uint64, filter func(*types.Entry) bool) (blocks []uint64) {
	safeIndex, err := types.SanitizeIndex(index)
	if err != nil {
		details := map[string]string{"index": index}
		utils.LavaError(ctx, ctx.Logger(), "getEntryVersionsFilter", details, "invalid non-ascii entry")
		return nil
	}

	store := fs.getEntryStore(ctx, safeIndex)

	iterator := sdk.KVStoreReversePrefixIterator(store, []byte{})
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var entry types.Entry
		fs.cdc.MustUnmarshal(iterator.Value(), &entry)

		if filter(&entry) {
			blocks = append(blocks, entry.Block)
		}

		if entry.Block <= block {
			break
		}
	}

	// reverse the result slice to return the block in ascending order
	length := len(blocks)
	for i := 0; i < length/2; i++ {
		blocks[i], blocks[length-i-1] = blocks[length-i-1], blocks[i]
	}

	return blocks
}

// GetEntryVersionsRange returns a list of versions from nearest-smaller block
// and onward, and not more than delta blocks further (skip stale entries).
func (fs *FixationStore) GetEntryVersionsRange(ctx sdk.Context, index string, block, delta uint64) (blocks []uint64) {
	filter := func(entry *types.Entry) bool {
		if entry.IsStale(ctx) {
			return false
		}
		if entry.Block > block+delta {
			return false
		}
		return true
	}

	return fs.getEntryVersionsFilter(ctx, index, block, filter)
}

// GetAllEntryVersions returns a list of all versions (blocks) of an entry.
// If stale == true, then the output will include stale versions (for testing).
func (fs *FixationStore) GetAllEntryVersions(ctx sdk.Context, index string, stale bool) (blocks []uint64) {
	filter := func(entry *types.Entry) bool {
		if !stale && entry.IsStale(ctx) {
			return false
		}
		return true
	}

	return fs.getEntryVersionsFilter(ctx, index, 0, filter)
}

func (fs *FixationStore) createEntryStoreKey(index string) string {
	return fs.prefix + types.EntryPrefix + index
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
