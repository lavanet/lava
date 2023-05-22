package common

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/common/types"
	"github.com/lavanet/lava/utils"
)

// FixationStore manages lists of entries with versions in the store.
// (See also documentation in common/fixation_entry_index.go)
//
// Purpose and API:
//
// Its primary use it to implement "fixated entries": entries that may change over
// time, and whose versions must be retained on-chain as long as they are referenced.
// For examples, an older version of a package is needed as long as the subscription
// that uses it lives.
//
// Once instantiated with NewFixationStore(), it offers the following methods:
//    - AppendEntry(index, block, *entry): add a new "block" version of an entry "index"
//    - ModifyEntry(index, block, *entry): modify an existing entry with "index" and exact "block" (*)
//    - ReadEntry(index, block, *entry): copy an existing entry with "index" and exact "block" (*)
//    - FindEntry(index, block, *entry): get a copy (no reference) of a version of an entry (**)
//    - GetEntry(index, *entry): get a copy (and reference) of the latest version of an entry
//    - PutEntry(index, block): drop reference to an existing entry with "index" and exact "block" (*)
//    - DelEntry(index, block): mark an entry as unavailable for new GetEntry() calls
//    - GetAllEntryIndices(): get all the entries indices (without versions)
//    - GetAllEntryVersions(index): get all the versions of an entry (for testing)
//    - GetEntryVersionsRange(index, block, delta): get range of entry versions (**)
//    - AdvanceBlock(): notify of block progress (e.g. BeginBlock) for garbage collection
// Note:
//    - methods marked with (*) expect an exact existing method, or otherwise will panic
//    - methods marked with (**) will match an entry with the nearest-smaller block version
//
// Usage and behavior:
//
// An entry version is identified by its index (name) and block (version). The "latest" entry
// version is the one with the highest block that is not greater than the current (ctx) block
// height. If an entry version is in the future (with respect to current block height), then
// it will become the new latest entry when its block is reached.
// A new entry version is added using AppendEntry().The version must be the current or future
// block, and higher than the latest entry's. Appending the same version again will override
// its previous data. An entry version can be modified using ModifyEntry().
// Entry versions maintain reference count (refcount) that determine their lifetime. New entry
// versions (appended) start with refcount 1. The refcount of the latest version is incremented
// using GetEntry(). The refcount of any version is decremented using PutEntry(). The refcount
// of the latest version is also decremented when a newer version is appended.
// When an entry's refcount reaches 0, it remains alive (visible) for a predefined period of
// blocks (stale-period), and then becomes stale (insvisible). Stale entries eventually get
// cleaned up.
// A nearest-smaller entry version can be obtained using FindEntry(). If the nearest smaller
// entry is stale, FindEntry() will return not-found. A specific entry version can be obtained
// using ReadEntry(), including of stale entries. (Thus, ReadEntry() is suitable for testing
// and migrations, or when the caller knows that the entry version is not stale).
// An entry can be deleted using DelEntry(), after which is will not possible to GetEntry(),
// and calls to FindEntry() for a block beyond that at time of deletion (or later) would fail.
// Until the data has been fully cleaned up (i.e. no refcount and beyond all stale-periods),
// calls to AppendEntry() with that entry's index will fail.
// GetAllEntryVersions() and GetEntryVersionsRange() give the all -or some- versions (blocks)
// of an entry. GetAllEntryIndices() return all the entry indices (names).
// On every new block, AdvanceBlock() should be called.
//
// Entry names (index) must contain only visible ascii characters (ascii values 32-125).
// The ascii 'DEL' invisible character is used internally to terminate the index values
// when stored, to ensure that no two indices can ever overlap, i.e. one being the prefix
// of the other.
// (Note that this properly supports Bech32 addresses, which are limited to use only
// visible ascii characters as per https://en.bitcoin.it/wiki/BIP_0173#Specification).
//
// Under the hood:
//
// The explanation below illustrates how data is stored, assuming module "packages" is the user:
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
//
// 5. When an entry (index) is deleted, a placeholder entry is a appended to mark the
// block at which deletion took place. This ensures that FindEntry() for the previous
// latest entry version would continue to work until that block and fail thereafter.

type FixationStore struct {
	storeKey sdk.StoreKey
	cdc      codec.BinaryCodec
	prefix   string
	tstore   TimerStore
}

func FixationVersion() uint64 {
	return 3
}

// we use timers for three kinds of timeouts: when a future entry becomes in effect,
// when a delete of entry becomes in effect, and when the stale-period ends and an
// entry should become stale.
// for the timers, we use <block,kind,index> tuple as a unique key to identify some
// entry version and the desited timeout type. this choice ensures timeouts will
// fire in order of expiry blocks, and in order of timeout kind.

const (
	// NOTE: TimerFutureEntry should be smaller than timerDeleteEntry, to
	// ensure that it fires first (because the latter will removes future entries,
	// so otherwise if they both expire on the same block then the future entry
	// callback will be surprised when it cannot find the respective entry).
	timerFutureEntry = 0x01
	timerDeleteEntry = 0x02
	timerStaleEntry  = 0x03
)

func encodeForTimer(index string, block uint64, kind byte) []byte {
	// NOTE: the encoding places callback type first to ensure the order of
	// callbacks when there are multiple at the same block (for some entry);
	// it is followed by the entry version (block) and index.
	encodedKey := make([]byte, 8+1+len(index))
	copy(encodedKey[9:], []byte(index))
	binary.BigEndian.PutUint64(encodedKey[1:9], block)
	encodedKey[0] = kind
	return encodedKey
}

func decodeFromTimer(encodedKey []byte) (index string, block uint64, kind byte) {
	index = string(encodedKey[9:])
	block = binary.BigEndian.Uint64(encodedKey[1:9])
	kind = encodedKey[0]
	return index, block, kind
}

func (fs *FixationStore) getEntryStore(ctx sdk.Context, index string) *prefix.Store {
	store := prefix.NewStore(
		ctx.KVStore(fs.storeKey),
		types.KeyPrefix(fs.createEntryStoreKey(index)))
	return &store
}

func (fs *FixationStore) replaceTimer(ctx sdk.Context, prev, next types.Entry, kind byte) {
	key := encodeForTimer(prev.Index, prev.Block, kind)
	fs.tstore.DelTimerByBlockHeight(ctx, next.DeleteAt, key)

	key = encodeForTimer(next.Index, next.Block, kind)
	fs.tstore.AddTimerByBlockHeight(ctx, next.DeleteAt, key, []byte{})
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
		return utils.LavaFormatError("AppendEntry", err,
			utils.Attribute{Key: "index", Value: index},
		)
	}

	blockHeight := uint64(ctx.BlockHeight())

	if block < blockHeight {
		panic(fmt.Sprintf("AppendEntry for block %d < current ctx block %d", block, blockHeight))
	}

	// find latest entry, including possible future entries
	latestEntry, found := fs.getUnmarshaledEntryForBlock(ctx, safeIndex, block)

	var deleteAt uint64 = math.MaxUint64

	// if latest entry is not found, this is a first version entry
	if !found {
		fs.setEntryIndex(ctx, safeIndex, true)
	} else {
		if block < latestEntry.Block {
			panic(fmt.Sprintf("AppendEntry for block %d < latest entry block %d", block, latestEntry.Block))
		}

		// temporary: do not allow adding new entries for an index that was deleted
		// and still not fully cleaned up (e.g. not stale or with references held)
		if latestEntry.IsDeletedBy(blockHeight) {
			return utils.LavaFormatError("AppendEntry",
				fmt.Errorf("entry already deleted and pending cleanup"),
				utils.Attribute{Key: "index", Value: index},
				utils.Attribute{Key: "block", Value: block},
			)
		}

		// if the new entry's block is equal to the latest entry, overwrite the latest entry
		if block == latestEntry.Block {
			fs.ModifyEntry(ctx, index, block, entryData)
			return nil
		}

		// if we are superseding a previous latest entry, then drop the latter's refcount;
		// otherwise we are a future entry version, so set a timer for when it will become
		// the new latest entry.
		// and if indeed we are, and the latest entry has DeleteAt set, then transfer this
		// DeleteAt to the new entry, drop the old timer, and start a new timer. (note:
		// deletion, if any, cannot be for the current block, because it would have been
		// processed at the beginning of the block (and AppendEntry would fail earlier).

		if block == blockHeight {
			deleteAt = latestEntry.DeleteAt
			latestEntry.DeleteAt = math.MaxUint64
			fs.putEntry(ctx, latestEntry)
		} else {
			key := encodeForTimer(safeIndex, block, timerFutureEntry)
			fs.tstore.AddTimerByBlockHeight(ctx, block, key, []byte{})
		}
	}

	// marshal the new entry's data
	marshaledEntryData := fs.cdc.MustMarshal(entryData)

	// create a new entry
	entry := types.Entry{
		Index:    safeIndex,
		Block:    block,
		StaleAt:  math.MaxUint64,
		DeleteAt: deleteAt,
		Data:     marshaledEntryData,
		Refcount: 1,
	}

	if entry.HasDeleteAt() {
		fs.replaceTimer(ctx, latestEntry, entry, timerDeleteEntry)
	}

	fs.setEntry(ctx, entry)
	return nil
}

func (fs *FixationStore) entryCallbackBeginBlock(ctx sdk.Context, key []byte, data []byte) {
	safeIndex, block, kind := decodeFromTimer(key)

	types.AssertSanitizedIndex(safeIndex, fs.prefix)

	switch kind {
	case timerFutureEntry:
		fs.updateFutureEntry(ctx, safeIndex, block)
	case timerDeleteEntry:
		fs.deleteMarkedEntry(ctx, safeIndex, block)
	case timerStaleEntry:
		fs.deleteStaleEntries(ctx, safeIndex, block)
	}
}

func (fs *FixationStore) updateFutureEntry(ctx sdk.Context, safeIndex string, block uint64) {
	if block != uint64(ctx.BlockHeight()) {
		panic(fmt.Sprintf("Future entry: future block %d != current block %d", block, ctx.BlockHeight()))
	}

	latestEntry, found := fs.getUnmarshaledEntryForBlock(ctx, safeIndex, block-1)
	if found {
		// if the latest entry has DeleteAt set, then transfer this DeleteAt to this
		// future entry (becoming latest). also, drop the old timer, and start a new
		// timer. (note: // deletion, if any, cannot be for the current block, since
		// it would have been processed at the beginning of the block.

		if latestEntry.HasDeleteAt() {
			entry := fs.getEntry(ctx, safeIndex, block)
			entry.DeleteAt = latestEntry.DeleteAt
			latestEntry.DeleteAt = math.MaxUint64
			fs.setEntry(ctx, entry)

			if entry.IsDeletedBy(block) {
				// if DeleteAt is exactly now as we transition from future entry to
				// be the latest entry, then simply invoke deleteMarkedEntry().
				key := encodeForTimer(latestEntry.Index, latestEntry.Block, timerDeleteEntry)
				fs.tstore.DelTimerByBlockHeight(ctx, block, key)
				fs.deleteMarkedEntry(ctx, safeIndex, block)
			} else {
				// otherwise, then replace the old timer (for the previous latest)
				// with a new timer (for the new latest).
				fs.replaceTimer(ctx, latestEntry, entry, timerDeleteEntry)
			}
		}

		// latest entry had extra refcount for being the latest; so drop that refcount
		// because from now on it is no longer so.

		fs.putEntry(ctx, latestEntry)
	}
}

func (fs *FixationStore) deleteMarkedEntry(ctx sdk.Context, safeIndex string, block uint64) {
	entry := fs.getEntry(ctx, safeIndex, block)
	blockHeight := uint64(ctx.BlockHeight())

	if entry.DeleteAt != blockHeight {
		panic(fmt.Sprintf("DelEntry entry deleted %d != current ctx block %d", entry.DeleteAt, uint64(ctx.BlockHeight())))
	}

	fs.setEntryIndex(ctx, safeIndex, false)
	fs.putEntry(ctx, entry)

	// forcefully remove all future entries: they were never referenced hence do not
	// require stale-period.

	store := fs.getEntryStore(ctx, safeIndex)

	iterator := sdk.KVStoreReversePrefixIterator(store, []byte{})
	defer iterator.Close()

	var entriesToRemove []types.Entry
	for ; iterator.Valid(); iterator.Next() {
		var entry types.Entry
		fs.cdc.MustUnmarshal(iterator.Value(), &entry)

		if entry.Block <= blockHeight {
			break
		}

		entriesToRemove = append(entriesToRemove, entry)
	}

	for _, entry := range entriesToRemove {
		key := encodeForTimer(entry.Index, entry.Block, timerFutureEntry)
		fs.tstore.DelTimerByBlockHeight(ctx, entry.Block, key)
		fs.removeEntry(ctx, entry.Index, entry.Block)
	}
}

func (fs *FixationStore) deleteStaleEntries(ctx sdk.Context, safeIndex string, _ uint64) {
	store := fs.getEntryStore(ctx, safeIndex)
	block := uint64(ctx.BlockHeight())

	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	// "stale" entry versions are ones that reached refcount zero at least
	// STALE_TIME blocks ago; they are not visible in lookups, hence may be
	// discarded. specifically, a stale entry version becomes "elgibile for
	// removal", if either it is:
	//   the oldest entry version, -OR-
	//   one that follows a stale entry version (but not marked deleted)
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
	// the one exception is an entry marked "deleted": it marks a deletion
	// point and affects everything before, and hence must remain in place
	// (unless it is the oldest entry, and then can be removed).

	var removals []uint64

	// if oldest -or-
	// if not marked "deleted" and previous entry was stale
	safeToDeleteEntry := true
	// if none of the entry versions were skipped
	safeToDeleteIndex := true

	for ; iterator.Valid(); iterator.Next() {
		// umarshal the old entry version
		var entry types.Entry
		fs.cdc.MustUnmarshal(iterator.Value(), &entry)

		// entry marked deleted and is not oldest: skip
		if entry.HasDeleteAt() && !safeToDeleteIndex {
			safeToDeleteEntry = false
			safeToDeleteIndex = false
			continue
		}

		// entry is not stale: skip
		if !entry.IsStaleBy(block) {
			safeToDeleteEntry = false
			safeToDeleteIndex = false
			continue
		}

		// entry not safe to delete: update state
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
	blockHeight := uint64(ctx.BlockHeight())

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
			// stale entries work as markers of the extent to where the preceding
			// (non stale) entry is valid. So if we meet one, we bail.
			// however if that stale entry is also deleted (which means, it is the
			// latest too) then we need to not bail, and do return that entry, to
			// comply with the expecations of AppendEntry(). thus, we may return an
			// entry that is both stale and deleted.
			// for this reason, we explicitly test for a deleted entry in EntryGet()
			// and AppendEntry(), and for stale entry in EntryFind(). so EntryGet()
			// and EntryFind() would return not-found, and AppendEntry would fail.
			if entry.IsStaleBy(blockHeight) && !entry.IsDeletedBy(blockHeight) {
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
	if !found || entry.IsStaleBy(block) || entry.IsDeletedBy(block) {
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
	if !found || entry.IsDeletedBy(block) {
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
		key := encodeForTimer(entry.Index, entry.Block, timerStaleEntry)
		fs.tstore.AddTimerByBlockHeight(ctx, entry.StaleAt, key, []byte{})
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

// DelEntry marks the entry for deletion so that future GetEntry will not find it
// anymore. (Cleanup of the EntryIndex is done when the last reference is removed).
func (fs *FixationStore) DelEntry(ctx sdk.Context, index string, block uint64) error {
	safeIndex, err := types.SanitizeIndex(index)
	if err != nil {
		return sdkerrors.ErrNotFound.Wrapf("invalid non-ascii index")
	}

	blockHeight := uint64(ctx.BlockHeight())

	if block < blockHeight {
		panic(fmt.Sprintf("DelEntry for block %d < current ctx block %d", block, blockHeight))
	}

	entry, found := fs.getUnmarshaledEntryForBlock(ctx, safeIndex, blockHeight)
	if !found || entry.HasDeleteAt() {
		return sdkerrors.ErrNotFound
	}

	entry.DeleteAt = block
	fs.setEntry(ctx, entry)

	// if we are deleting the entry at this block, then invoke deleteMarkedEntry()
	// to do the work. otherwise this is a future entry delete, so set a timer for
	// when it will become imminent.

	if block == uint64(ctx.BlockHeight()) {
		fs.deleteMarkedEntry(ctx, safeIndex, entry.Block)
	} else {
		key := encodeForTimer(safeIndex, entry.Block, timerDeleteEntry)
		fs.tstore.AddTimerByBlockHeight(ctx, block, key, []byte{})
	}

	return nil
}

// removeEntry removes an entry from the store
func (fs *FixationStore) removeEntry(ctx sdk.Context, safeIndex string, block uint64) {
	store := fs.getEntryStore(ctx, safeIndex)
	store.Delete(types.EncodeKey(block))
}

func (fs *FixationStore) getEntryVersionsFilter(ctx sdk.Context, index string, block uint64, filter func(*types.Entry) bool) (blocks []uint64) {
	safeIndex, err := types.SanitizeIndex(index)
	if err != nil {
		utils.LavaFormatError("getEntryVersionsFilter failed", err,
			utils.Attribute{Key: "index", Value: index},
		)
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
		if entry.IsStaleBy(uint64(ctx.BlockHeight())) {
			return false
		}
		if entry.Block > block+delta {
			return false
		}
		return true
	}

	return fs.getEntryVersionsFilter(ctx, index, block, filter)
}

// GetAllEntryVersions returns a list of all versions (blocks) of an entry, for
// use in testing and migrations (includes stale and deleted entry versions).
func (fs *FixationStore) GetAllEntryVersions(ctx sdk.Context, index string) (blocks []uint64) {
	filter := func(entry *types.Entry) bool { return true }
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

	callback := func(ctx sdk.Context, key []byte, data []byte) {
		fs.entryCallbackBeginBlock(ctx, key, data)
	}

	tstore := NewTimerStore(storeKey, cdc, prefix).WithCallbackByBlockHeight(callback)
	fs.tstore = *tstore

	return &fs
}
