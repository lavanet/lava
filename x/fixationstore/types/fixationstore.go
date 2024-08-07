package types

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/v2/utils"
	timerstoretypes "github.com/lavanet/lava/v2/x/timerstore/types"
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
//    - FindEntry2(index, block, *entry): get a copy (no reference) of a version of an entry (**)
//    - GetEntry(index, *entry): get a copy (and reference) of the latest version of an entry
//    - PutEntry(index, block): drop reference to an existing entry with "index" and exact "block" (*)
//    - DelEntry(index, block): mark an entry as unavailable for new GetEntry() calls
//    - HasEntry(index, block): checks for existence of a specific version of an entry
//    - IsEntryStale(index, block): checks if a specific version of an entry is stale
//    - GetAllEntryIndices(): get all the entries indices (without versions)
//    - GetAllEntryVersions(index): get all the versions of an entry (for testing)
//    - GetEntryVersionsRange(index, block, delta): get range of entry versions (**)
// Note:
//    - methods marked with (*) expect an exact existing method, or otherwise will panic
//    - methods marked with (**) will match an entry with the nearest-no-later block version
//
// Usage and behavior:
//
// An entry version is identified by its index (name) and block (version). The "latest" entry
// version is the one with the highest block that is not greater than the current (ctx) block
// height. If an entry version is in the future (with respect to current block height), then
// it will become the new latest entry when its block is reached.
//
// A new entry version is added using AppendEntry(). The version must be the current or future
// block, and higher than the latest entry's. Appending the same version again will override
// its previous data. An entry version can be modified using ModifyEntry().
//
// GetEntry() returns the latest (with respect to current, not future) version of an entry. A
// nearest-no-later (than a given block) entry version can be obtained with FindEntry(); And
// FindEntry2() is similar, but also returns the entry's version (block).
//
// Entry versions maintain reference count (refcount) that determine their lifetime. New entry
// versions (appended) start with refcount 1. The refcount of the latest version is incremented
// using GetEntry(). The refcount of the latest version is decremented when a newer version is
// appended, or when a future version becomes in effect. References taken with GetEntry() can
// be dropped (and the refcount decremented) by using PutEntry(). PutEntry() can also be used
// to cancel/remove a future entry.
//
// When an entry's refcount reaches 0, it remains partly visible for a predefined period of
// blocks (stale-period), and then becomes stale and fully invisible. During the stale-period
// GetEntry() will ignore the entry, however FindEntry() will find it. If the nearest-no-later
// (then a given block) entry is already stale, FindEntry() will return not-found. ReadEntry()
// will fetch any specific entry version, including of stale entries. (Thus, it is suitable for
// testing and migrations, or when the caller knows that the entry version exists, being stale
// or not). Stale entries eventually get cleaned up automatically.
//
// An entry can be deleted using DelEntry(), after which is will not possible to GetEntry(),
// and calls to FindEntry() for a block beyond that time of deletion (at or later) would fail
// too. (Note that DelEntry() will also discard any pending future versions of the entry).
//
// The per-entry APIs are and their actions are summarized below:
//
// o AppendEntry() adds a new version of an entry (could be a future version)
// o ModifyEntry() updates an existing version of an entry (could be a future version)
// o DelEntry() deletes and entry and make it invisible to GetEntry()
// o GetEntry() gets the latest (up to current) version of an entry, except if in stale-period
// o HasEntry() checks for existence of a specific version of an entry
// o FindEntry() gets the nearest-no-later version of an entry, including if in stale-period
// o FindEntry2() same as FindEntry(), and also returns the version (block) of the entry
// o ReadEntry() gets a specific entry version (stale or not)
// o PutEntry() drops reference counts taken by GetEntry()
//
// GetAllEntryVersions() and GetEntryVersionsRange() give the all -or some- versions (blocks)
// of an entry. GetAllEntryIndices() return all the entry indices (names).
//
//
// Entry names (index) must contain only visible ascii characters (ascii values 32-126).
// The ascii 'DEL' invisible character is used internally to terminate the index values
// when stored, to ensure that no two indices can ever overlap, i.e. one being the prefix
// of the other.
//
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
// and the RawEntries with a prefix that includes their index (using the block as key).
// For instance, modules "packages" may have a package named "YourPackage" created at
// block 110, and it was updated at block 220, and package named "MyPackage" created
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
// 5. When an entry (index) is deleted, it is kept as a placeholder entry to mark the
// block at which deletion took place. This ensures that FindEntry() for that latest
// entry version would continue to work until that block and fail thereafter.
//
// State invariants:
//
// o Refcount: tracks the reference count of entry versions
//   - extra refcount kept for latest and future entry versions
//   - incremented for latest entry version with GetEntry()
//   - decremented for specific entry version with PutEntry()
//   - every PutEntry() must match a previous GetEntry()
//   - PutEntry() can be used to remove a specific future entry
//   - PutEntry() is illegal to call on last refcount of latest entry
//
// o IsLatest: tracks the entry version which is now the latest
//   - transferred to the next (future) entry when it matures
//   - removed from delete entries
//
// o DeleteAt: tracks the block upon which the entry will be deleted
//   - held by the latest entry version smaller than or equal to that DeleteAt
//   - prohibits future entry versions on or beyond the DeleteAt block
//
// o StaleAt: tracks when a removed entry version becomes invisible
//   - set on non-future entries that reach refcount == 0
//
// Note that AppendEntry() may add retroactive entry only if it does not precede an
// existing latest entry. Also, AppendEntry() may not add future entries on or beyond
// a DeletedAt in the future (adding a future entry maturing on a future DeleteAt is
// permitted but useless, because it will be deleted straight away). However once the
// entry is deleted, new entries (future or not) can be appended again.
//
// Note also that DelEntry() discards all future entries on or beyond the DeletedAt
// block (and owing to the previous rule, none will be added until deletes occurs).

type GetStaleBlocks func(sdk.Context) uint64

type FixationStore struct {
	storeKey       storetypes.StoreKey
	cdc            codec.BinaryCodec
	prefix         string
	tstore         timerstoretypes.TimerStore
	getStaleBlocks GetStaleBlocks
}

var fixationVersion uint64 = 5

func FixationVersion() uint64 {
	return fixationVersion
}

// we use timers for three kinds of timeouts: when a future entry becomes in effect,
// when a delete of entry becomes in effect, and when the stale-period ends and an
// entry should become stale.
// for the timers, we use <block,kind,index> tuple as a unique key to identify some
// entry version and the desired timeout type. this choice ensures timeouts will
// fire in order of expiry blocks, and in order of timeout kind.

const (
	// NOTE: timerFutureEntry should be smaller than timerDeleteEntry, to ensure
	// that if both expire on the same block then future-entry timer fires before
	// delete-entry timer, so that the latter would remove the brand newly matured
	// future-entry.
	timerFutureEntry = 0x01
	timerDeleteEntry = 0x02
	timerStaleEntry  = 0x03
)

func encodeForTimer(safeIndex SafeIndex, block uint64, kind byte) []byte {
	// NOTE: the encoding places callback type first to ensure the order of
	// callbacks when there are multiple at the same block (for some entry);
	// it is followed by the entry version (block) and index.
	encodedKey := make([]byte, 8+1+len(safeIndex))
	copy(encodedKey[9:], []byte(safeIndex))
	binary.BigEndian.PutUint64(encodedKey[1:9], block)
	encodedKey[0] = kind
	return encodedKey
}

func decodeFromTimer(encodedKey []byte) (safeIndex SafeIndex, block uint64, kind byte) {
	safeIndex = SafeIndex(encodedKey[9:])
	block = binary.BigEndian.Uint64(encodedKey[1:9])
	kind = encodedKey[0]
	return safeIndex, block, kind
}

func (fs *FixationStore) getEntryStore(ctx sdk.Context, safeIndex SafeIndex) *prefix.Store {
	store := prefix.NewStore(
		ctx.KVStore(fs.storeKey),
		KeyPrefix(fs.createEntryStoreKey(string(safeIndex))))
	return &store
}

func (fs *FixationStore) GetStoreKey() storetypes.StoreKey {
	return fs.storeKey
}

func (fs *FixationStore) GetStorePrefix() string {
	return fs.prefix
}

// transferTimer moves a timer (unexpired) from a previous entry to a new entry, with
// the same expiry block. Useful, for example, when a newer entry takes responsibility
// for a pending deletion from the previous owner.
func (fs *FixationStore) transferTimer(ctx sdk.Context, prev, next Entry, block uint64, kind byte) {
	key := encodeForTimer(prev.SafeIndex(), prev.Block, kind)
	fs.tstore.DelTimerByBlockHeight(ctx, block, key)

	key = encodeForTimer(next.SafeIndex(), next.Block, kind)
	fs.tstore.AddTimerByBlockHeight(ctx, block, key, []byte{})
}

// hasEntry returns whether a specific entry exists in the store
// (any kind of entry, even deleted or stale)
func (fs *FixationStore) hasEntry(ctx sdk.Context, safeIndex SafeIndex, block uint64) bool {
	store := fs.getEntryStore(ctx, safeIndex)
	byteKey := EncodeKey(block)
	return store.Has(byteKey)
}

// getEntry returns an existing entry in the store
// (any kind of entry, even deleted or stale)
func (fs *FixationStore) getEntry(ctx sdk.Context, safeIndex SafeIndex, block uint64) (entry Entry) {
	store := fs.getEntryStore(ctx, safeIndex)
	byteKey := EncodeKey(block)
	b := store.Get(byteKey)
	if b == nil {
		// panic:ok: internal API that expects the <entry, block> to exist
		utils.LavaFormatPanic("fixation: getEntry failed (unknown entry)", legacyerrors.ErrNotFound,
			utils.Attribute{Key: "prefix", Value: fs.prefix},
			utils.Attribute{Key: "index", Value: DesanitizeIndex(safeIndex)},
			utils.Attribute{Key: "block", Value: block},
		)
	}
	fs.cdc.MustUnmarshal(b, &entry)
	return entry
}

// setEntry modifies an existing entry in the store
func (fs *FixationStore) setEntry(ctx sdk.Context, entry Entry) {
	store := fs.getEntryStore(ctx, entry.SafeIndex())
	byteKey := EncodeKey(entry.Block)
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
	safeIndex, err := SanitizeIndex(index)
	if err != nil {
		return utils.LavaFormatError("AppendEntry", err,
			utils.Attribute{Key: "index", Value: index},
		)
	}

	ctxBlock := uint64(ctx.BlockHeight())

	// find latest entry, including possible future entries
	latestEntry, found := fs.getUnmarshaledEntryForBlock(ctx, safeIndex, block)

	var deleteAt uint64 = math.MaxUint64
	var latest bool

	// if latest entry is not found, this is a first version entry
	// if latest entry is found but is deleted, also treat like a first version entry
	if !found || latestEntry.IsDeleted(ctx) {
		fs.setEntryIndex(ctx, safeIndex, true)
		if block <= ctxBlock {
			latest = true
		}
	} else {
		if block < latestEntry.Block {
			// how come getUnmarshaledEntryForBlock lied to us?!
			return utils.LavaFormatError("critical: AppendEntry block smaller than latest",
				fmt.Errorf("block %d < latest entry block %d", block, latestEntry.Block),
				utils.Attribute{Key: "prefix", Value: fs.prefix},
				utils.Attribute{Key: "index", Value: index},
				utils.Attribute{Key: "block", Value: block},
				utils.Attribute{Key: "latest", Value: latestEntry.Block},
			)
		}

		// the latestEntry we found is actually the nearest-no-later entry with respect to
		// our block; if it is not marked IsLatest, it means that the real latest entry is
		// still ahead of us (thus: entry-found < entry-appended < latest entry). however,
		// appending an entry, while can happen with a past block, must not succeed if it
		// precedes en existing latest entry.

		if block < ctxBlock && !latestEntry.IsLatest {
			return utils.LavaFormatWarning("AppendEntry",
				fmt.Errorf("entry is older than existing latest entry"),
				utils.Attribute{Key: "index", Value: index},
				utils.Attribute{Key: "block", Value: block},
			)
		}

		// if the new entry's block is equal to the nearest-no-later entry, overwrite that
		// entry (only possible for latest and future entries, as the previous test would
		// exclude earlier-than-latest entries).

		if block == latestEntry.Block {
			fs.ModifyEntry(ctx, index, block, entryData)
			return nil
		}

		// we know the entry is not yet deleted (as tested above), so this is an attempt
		// to append on or beyond a pending delete, which is not permitted.

		if latestEntry.IsDeletedBy(block) {
			return utils.LavaFormatWarning("AppendEntry",
				fmt.Errorf("entry disallows future version on or beyond pending delete"),
				utils.Attribute{Key: "index", Value: index},
				utils.Attribute{Key: "block", Value: block},
				utils.Attribute{Key: "delete", Value: latestEntry.DeleteAt},
			)
		}

		// if the previous latest entry is marked with DeleteAt, and this future entry is set
		// to mature strictly before that DeleteAt block (and specifically not at that block),
		// then transfer this DeleteAt to the future entry, and then replace the old timer
		// with a new timer (see below).

		if latestEntry.HasDeleteAt() {
			// marked for delete that would take place after this future entry - transfer
			// the marking to this future entry (which by now we know is later)
			deleteAt = latestEntry.DeleteAt
			latestEntry.DeleteAt = math.MaxUint64
			fs.setEntry(ctx, latestEntry)
		}
	}

	// if we are superseding a previous latest entry, then drop the latter's refcount (but
	// not if the latest entry is deleted, since it lost its extra reference upon deletion);
	// otherwise, if we are a future entry version, set a timer for when it will become the
	// new latest entry.

	if block <= ctxBlock {
		// the new entry is becoming the latest now: mark as such
		latest = true
		if latestEntry.IsLatest {
			// the previous entry was the latest and will no longer be so going forward, so
			// drop the "latest" reference from that previous entry.
			latestEntry.IsLatest = false
			fs.putEntry(ctx, latestEntry) // also saves updated latestEntry
		}
	} else {
		// the new entry is not yet in effect, create a timer to put reference back to
		// the latest once this is changed.
		key := encodeForTimer(safeIndex, block, timerFutureEntry)
		fs.tstore.AddTimerByBlockHeight(ctx, block, key, []byte{})
	}

	// marshal the new entry's data
	marshaledEntryData := fs.cdc.MustMarshal(entryData)

	// create a new entry
	entry := Entry{
		Index:    string(safeIndex),
		Block:    block,
		StaleAt:  math.MaxUint64,
		DeleteAt: deleteAt,
		Data:     marshaledEntryData,
		IsLatest: latest,
		Refcount: 1,
	}

	if entry.HasDeleteAt() {
		fs.transferTimer(ctx, latestEntry, entry, entry.DeleteAt, timerDeleteEntry)
	}

	fs.setEntry(ctx, entry)
	return nil
}

func (fs *FixationStore) entryCallbackBeginBlock(ctx sdk.Context, key, data []byte) {
	safeIndex, block, kind := decodeFromTimer(key)

	AssertSanitizedIndex(safeIndex, fs.prefix)

	switch kind {
	case timerFutureEntry:
		fs.updateFutureEntry(ctx, safeIndex, block)
	case timerDeleteEntry:
		fs.deleteMarkedEntry(ctx, safeIndex, block)
	case timerStaleEntry:
		fs.deleteStaleEntries(ctx, safeIndex, block)
	default:
		utils.LavaFormatPanic("fixation: timer callback unknown type",
			// panic:ok: state is badly invalid, because we expect the kind of the timer
			// to always be one of the above.
			fmt.Errorf("unknown callback kind = %x", kind),
			utils.Attribute{Key: "prefix", Value: fs.prefix},
			utils.Attribute{Key: "index", Value: DesanitizeIndex(safeIndex)},
			utils.Attribute{Key: "block", Value: ctx.BlockHeight()},
		)
	}
}

func (fs *FixationStore) updateFutureEntry(ctx sdk.Context, safeIndex SafeIndex, block uint64) {
	// sanity check: future entries should get timeout on their block reaches now
	ctxBlock := uint64(ctx.BlockHeight())
	if block != ctxBlock {
		// panic:ok: state is badly invalid, because we expect the expiry block of a
		// timer that expired now to be same as current block height.
		// TODO: make this a panic
		utils.LavaFormatWarning("fixation: future callback block mismatch",
			fmt.Errorf("wrong expiry block %d != current block %d", block, ctxBlock),
			utils.Attribute{Key: "prefix", Value: fs.prefix},
			utils.Attribute{Key: "index", Value: DesanitizeIndex(safeIndex)},
			utils.Attribute{Key: "expiry", Value: block},
			utils.Attribute{Key: "block", Value: ctxBlock},
		)
	}

	latestEntry, found := fs.getUnmarshaledEntryForBlock(ctx, safeIndex, block-1)
	if found && latestEntry.IsLatest {
		// previous latest entry should never have its DeleteAt set for this block:
		// if our AppendEntry happened before some DelEntry, we would get marked for
		// delete with DeleteAt (and not the previous latest entry); if the DelEntry
		// happened first, then upon our AppendEntry() we would inherit the DeleteAt
		// from the previous latest entry.

		if latestEntry.IsDeletedBy(block) {
			// panic:ok: internal state mismatch, unknown outcome if we proceed
			utils.LavaFormatPanic("fixation: future entry callback invalid state",
				fmt.Errorf("previous latest entry marked delete at %d", latestEntry.DeleteAt),
				utils.Attribute{Key: "prefix", Value: fs.prefix},
				utils.Attribute{Key: "index", Value: DesanitizeIndex(safeIndex)},
				utils.Attribute{Key: "block", Value: block},
				utils.Attribute{Key: "latest", Value: latestEntry.Block},
			)
		}

		// latest entry had extra refcount for being the latest; so drop that refcount
		// because from now on it is no longer so.

		latestEntry.IsLatest = false
		fs.putEntry(ctx, latestEntry) // also saves updated latestEntry
	}

	// we are now the latest entry - make it official by setting IsLatest

	entry := fs.getEntry(ctx, safeIndex, block)
	entry.IsLatest = true
	fs.setEntry(ctx, entry)
}

func (fs *FixationStore) deleteMarkedEntry(ctx sdk.Context, safeIndex SafeIndex, block uint64) {
	entry := fs.getEntry(ctx, safeIndex, block)
	ctxBlock := uint64(ctx.BlockHeight())

	if entry.DeleteAt != ctxBlock {
		// panic:ok: internal state mismatch, unknown outcome if we proceed
		// TODO: make this a panic
		utils.LavaFormatWarning("fixation: delete entry callback invalid state",
			fmt.Errorf("entry delete at %d != current block %d", entry.DeleteAt, ctxBlock),
			utils.Attribute{Key: "prefix", Value: fs.prefix},
			utils.Attribute{Key: "index", Value: DesanitizeIndex(safeIndex)},
			utils.Attribute{Key: "block", Value: ctxBlock},
			utils.Attribute{Key: "delete", Value: entry.DeleteAt},
		)
	}

	entry.IsLatest = false

	fs.setEntryIndex(ctx, safeIndex, false)
	fs.putEntry(ctx, entry)

	// no need to trim future entries: they were trimmed at the time of DelEntry, and
	// since then new entries (on or beyond DeleteAt block) could not be appended.
}

func (fs *FixationStore) deleteStaleEntries(ctx sdk.Context, safeIndex SafeIndex, _ uint64) {
	store := fs.getEntryStore(ctx, safeIndex)

	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	// "stale" entry versions are ones that reached refcount zero at least
	// STALE_TIME blocks ago; they are not visible in lookups, hence may be
	// discarded. specifically, a stale entry version becomes "eligible for
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
		// unmarshal the old entry version
		var entry Entry
		fs.cdc.MustUnmarshal(iterator.Value(), &entry)

		// entry marked deleted and is not oldest: skip
		if entry.HasDeleteAt() && !safeToDeleteIndex {
			safeToDeleteEntry = false
			safeToDeleteIndex = false
			continue
		}

		// entry is not stale: skip
		if !entry.IsStale(ctx) {
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

// trimFutureEntries discards all future entries (relative to the DeleteAt of the
// entry version marked for deletion)
func (fs *FixationStore) trimFutureEntries(ctx sdk.Context, lastEntry Entry) {
	store := fs.getEntryStore(ctx, lastEntry.SafeIndex())

	iterator := sdk.KVStoreReversePrefixIterator(store, []byte{})
	defer iterator.Close()

	// first collect all the candidates, and later remove them (as the iterator
	// logic gets confused with ongoing changes)
	var entriesToRemove []Entry
	for ; iterator.Valid(); iterator.Next() {
		var entry Entry
		fs.cdc.MustUnmarshal(iterator.Value(), &entry)

		if entry.Block < lastEntry.DeleteAt {
			break
		}

		entriesToRemove = append(entriesToRemove, entry)
	}

	// forcefully remove the future entries:
	// they were never referenced hence do not require stale-period.

	for _, entry := range entriesToRemove {
		fs.putFutureEntry(ctx, entry.SafeIndex(), entry.Block)
	}
}

// ReadEntry returns and existing entry with index and specific block
// (should be called only for existing entries; will panic otherwise)
func (fs *FixationStore) ReadEntry(ctx sdk.Context, index string, block uint64, entryData codec.ProtoMarshaler) {
	safeIndex, err := SanitizeIndex(index)
	if err != nil {
		// panic:ok: entry expected to exist as is
		utils.LavaFormatPanic("fixation: ReadEntry failed (invalid index)", err,
			utils.Attribute{Key: "prefix", Value: fs.prefix},
			utils.Attribute{Key: "index", Value: index},
		)
	}

	entry := fs.getEntry(ctx, safeIndex, block)
	fs.cdc.MustUnmarshal(entry.GetData(), entryData)
}

// ModifyEntry modifies an existing entry in the store
// (should be called only for existing entries; will panic otherwise)
func (fs *FixationStore) ModifyEntry(ctx sdk.Context, index string, block uint64, entryData codec.ProtoMarshaler) {
	safeIndex, err := SanitizeIndex(index)
	if err != nil {
		// panic:ok: entry expected to exist as is
		utils.LavaFormatPanic("fixation: ModifyEntry failed (invalid index)", err,
			utils.Attribute{Key: "prefix", Value: fs.prefix},
			utils.Attribute{Key: "index", Value: index},
		)
	}

	entry := fs.getEntry(ctx, safeIndex, block)
	entry.Data = fs.cdc.MustMarshal(entryData)
	fs.setEntry(ctx, entry)
}

// FindRawEntry returns the raw entry by index and block (even deleted or stale entries)
func (fs *FixationStore) FindRawEntry(ctx sdk.Context, index string, block uint64) (Entry, error) {
	safeIndex, err := SanitizeIndex(index)
	if err != nil {
		return Entry{}, err
	}

	store := fs.getEntryStore(ctx, safeIndex)
	iterator := sdk.KVStoreReversePrefixIterator(store, []byte{})
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var entry Entry
		fs.cdc.MustUnmarshal(iterator.Value(), &entry)
		if entry.Block <= block {
			return entry, nil
		}
	}

	return Entry{}, fmt.Errorf("FindRawEntry: entry not found")
}

// getUnmarshaledEntryForBlock gets an entry version for an index that has
// nearest-no-later block version for the given block arg.
func (fs *FixationStore) getUnmarshaledEntryForBlock(ctx sdk.Context, safeIndex SafeIndex, block uint64) (Entry, bool) {
	AssertSanitizedIndex(safeIndex, fs.prefix)
	store := fs.getEntryStore(ctx, safeIndex)
	ctxBlock := uint64(ctx.BlockHeight())

	iterator := sdk.KVStoreReversePrefixIterator(store, []byte{})
	defer iterator.Close()

	// iterate over entries in reverse order of version (block), and return the
	// first version that is smaller-or-equal to the requested block. Thus, the
	// caller gets the latest version at the time of the block (arg).
	// For example, assuming two entries for blocks 100 and 200, asking for the
	// entry with block 199 would return the entry with block 100, which was in
	// effect at the time of block 199.

	for ; iterator.Valid(); iterator.Next() {
		var entry Entry
		fs.cdc.MustUnmarshal(iterator.Value(), &entry)

		if entry.Block <= block {
			// stale entries work as markers of the extent to where the preceding
			// (non stale) entry is valid. So if we meet one, we bail.
			// however if that stale entry is also deleted (which means, it is the
			// latest too) then we need to not bail, and do return that entry, to
			// comply with the expectations of AppendEntry(). thus, we may return an
			// entry that is both stale and deleted.
			// for this reason, we explicitly test for a deleted entry in GetEntry()
			// and AppendEntry(), and for stale entry in GetEntry(). so GetEntry()
			// and FindEntry() would return not-found, and AppendEntry() would fail.
			//
			// Note that we test for stale-ness against ctx.BlockHeight since it is
			// meant to mark when an unreferenced only entry becomes invisible; and
			// we test for delete-ness against the target block since it would mark
			// that entry immediately (as in: at deletion block) invisible.

			if entry.IsStaleBy(ctxBlock) && !entry.IsDeletedBy(block) {
				break
			}

			return entry, true
		}
	}

	return Entry{}, false
}

// FindEntryDetailed returns the entry by index, whether it's deleted, and block without changing the refcount
func (fs *FixationStore) FindEntryDetailed(ctx sdk.Context, index string, block uint64, entryData codec.ProtoMarshaler) (entryBlock uint64, isDeleted bool, isLatest bool, found bool) {
	safeIndex, err := SanitizeIndex(index)
	if err != nil {
		utils.LavaFormatError("FindEntry failed (invalid index)", err,
			utils.Attribute{Key: "index", Value: index},
		)
		return 0, false, false, false
	}

	entry, found := fs.getUnmarshaledEntryForBlock(ctx, safeIndex, block)

	// if an entry was found, then it is either not stale -or- it is both stale
	// (by ctx.BlockHeight) and deleted (by the given block) - see the logic in
	// in getUnmarshalledEntryForBlock(). An entry of the latter kind - both
	// stale and deleted - should not be visible, so we explicitly skip it. Note
	// that it's enough to test only one of stale/deleted, because both must be
	// true in this case.

	if !found || entry.IsDeletedBy(block) {
		return 0, entry.IsDeletedBy(block), false, false
	}

	fs.cdc.MustUnmarshal(entry.GetData(), entryData)
	return entry.Block, entry.IsDeleted(ctx), entry.IsLatest, true
}

// FindEntry returns the entry by index and block without changing the refcount
func (fs *FixationStore) FindEntry(ctx sdk.Context, index string, block uint64, entryData codec.ProtoMarshaler) bool {
	_, _, _, found := fs.FindEntryDetailed(ctx, index, block, entryData)
	return found
}

// IsEntryStale returns true if an entry version exists and is stale.
func (fs *FixationStore) IsEntryStale(ctx sdk.Context, index string, block uint64) bool {
	safeIndex, err := SanitizeIndex(index)
	if err != nil {
		utils.LavaFormatError("IsEntryStale failed (invalid index)", err,
			utils.Attribute{Key: "index", Value: index},
		)
		return false
	}
	entry := fs.getEntry(ctx, safeIndex, block)
	return entry.IsStale(ctx)
}

// HasEntry returns true if an entry version exists for the given index, block tuple
// (any kind of entry, even deleted or stale).
func (fs *FixationStore) HasEntry(ctx sdk.Context, index string, block uint64) bool {
	safeIndex, err := SanitizeIndex(index)
	if err != nil {
		utils.LavaFormatError("HasEntry failed (invalid index)", err,
			utils.Attribute{Key: "index", Value: index},
		)
		return false
	}
	return fs.hasEntry(ctx, safeIndex, block)
}

// GetEntry returns the latest entry by index and increments the refcount
// (while ignoring entries in stale-period)
func (fs *FixationStore) GetEntry(ctx sdk.Context, index string, entryData codec.ProtoMarshaler) bool {
	safeIndex, err := SanitizeIndex(index)
	if err != nil {
		utils.LavaFormatError("GetEntry failed (invalid index)", err,
			utils.Attribute{Key: "index", Value: index},
		)
		return false
	}

	ctxBlock := uint64(ctx.BlockHeight())

	entry, found := fs.getUnmarshaledEntryForBlock(ctx, safeIndex, ctxBlock)
	if !found || entry.IsDeletedBy(ctxBlock) {
		return false
	}

	fs.cdc.MustUnmarshal(entry.GetData(), entryData)

	entry.Refcount += 1
	fs.setEntry(ctx, entry)
	return true
}

// putEntry decrements the refcount of an entry and marks for staleness if needed
func (fs *FixationStore) putEntry(ctx sdk.Context, entry Entry) {
	if entry.Refcount == 0 {
		// panic:ok: double putEntry() is bad news like double free
		safeIndex := SafeIndex(entry.Index)
		utils.LavaFormatPanic("fixation: putEntry invalid refcount state",
			fmt.Errorf("unable to put entry with refcount 0"),
			utils.Attribute{Key: "prefix", Value: fs.prefix},
			utils.Attribute{Key: "index", Value: DesanitizeIndex(safeIndex)},
		)
	}

	entry.Refcount -= 1

	if entry.Refcount == 0 {
		block := uint64(ctx.BlockHeight())

		// future entries (i.e. cancel of a future append) are deleted immediately:
		// they were never in use so they need not undergo stale-period.
		if entry.Block > block {
			fs.putFutureEntry(ctx, entry.SafeIndex(), entry.Block)
			return
		}

		// non-future entries must pass "stale period"; setup a timer for that
		// (the computation never overflows because ctx.BlockHeight is int64)

		entry.StaleAt = block + fs.getStaleBlocks(ctx)
		key := encodeForTimer(entry.SafeIndex(), entry.Block, timerStaleEntry)
		fs.tstore.AddTimerByBlockHeight(ctx, entry.StaleAt, key, []byte{})
	}

	fs.setEntry(ctx, entry)
}

// PutEntry finds the entry by index and block and decrements the refcount
// (should be called only for existing entries; will panic otherwise)
func (fs *FixationStore) PutEntry(ctx sdk.Context, index string, block uint64) {
	safeIndex, err := SanitizeIndex(index)
	if err != nil {
		// panic:ok: entry expected to exist as is
		utils.LavaFormatPanic("fixation: PutEntry failed (invalid index)", err,
			utils.Attribute{Key: "prefix", Value: fs.prefix},
			utils.Attribute{Key: "index", Value: index},
		)
	}

	entry := fs.getEntry(ctx, safeIndex, block)

	// every call to PutEntry() must match a prior call to GetEntry(); Therefore
	// it is illegal to call PutEntry() on the last reference of the latest entry.
	// if this happens, we complain loudly (instead of panic) and refuse to drop
	// the reference count.
	if entry.IsLatest && entry.Refcount == 1 {
		utils.LavaFormatError("critical: last PutEntry on latest entry",
			fmt.Errorf("PutEntry requested on latest entry with refcount 1"),
			utils.Attribute{Key: "index", Value: index},
			utils.Attribute{Key: "block", Value: block},
		)
		return
	}

	fs.putEntry(ctx, entry)
}

// DelEntry marks the entry for deletion so that future GetEntry will not find it
// anymore. (Cleanup of the EntryIndex is done when the last reference is removed).
func (fs *FixationStore) DelEntry(ctx sdk.Context, index string, block uint64) error {
	safeIndex, err := SanitizeIndex(index)
	if err != nil {
		return legacyerrors.ErrNotFound.Wrapf("invalid non-ascii index: %s", index)
	}

	ctxBlock := uint64(ctx.BlockHeight())

	if block < ctxBlock {
		return utils.LavaFormatError("critical: DelEntry of past block",
			fmt.Errorf("delete requested at an old block %d < %d", block, ctxBlock),
			utils.Attribute{Key: "index", Value: index},
		)
	}

	// if this is a future delete, then we search for a strictly nearest-smaller
	// previous entry (rather than nearest-no-later): if there exists an entry at
	// the target block, then we would trim it away. to illustrate, consider this
	// example (with ctx.BlockHeight=100, "C"/"F" for current/future):
	//   (C)Entry_95 -> (F)Entry_105 -> (F)Entry_110
	// in this case, DelEntry(105) will use deleteAt=104 and find (C)Entry_95 and
	// then trim the two future entries. DelEntry(110) will find (F)Entry_105 and
	// trim the last future entry.
	// this is correct because, if we left it intact, it would anyway be removed
	// when DeleteAt is reached later; this is also useful because without it an
	// append of a new entry immediately after (and in same block as) this delete
	// would be messy: the new entry would need to overwrite the deleted previous
	// entry (since same block); thus it's simpler to not permit such future entry
	// on the DeleteAt block).
	// the exception is a (first) future entry that has not matured yet, i.e. that
	// has no current version of the entry in place (e.g. after adding a new entry
	// as a future one but before it becomes actual). in this case, a search for
	// the strictly nearest-smaller previous entry would fail (since none exists
	// yet). on the other hand, the entry surely has not been used, because there
	// isn't a current version, so it can be deleted as is, and if there are more
	// future versions beyond it they can be trimmed too. to illustrate, consider
	// this example (with ctx.BlockHeight=100, "C"/"F" for current/future):
	//   (no current) -> (F)Entry_105 -> (F)Entry_110
	// in this case, DelEntry(105) will use deleteAt=104 and fail to find previous
	// entry, then will find (F)Entry_105 and trim both future entries.

	deleteAt := block
	if block > ctxBlock {
		// future delete: find strictly-nearest-smaller previous entry
		deleteAt--
	}

	entry, found := fs.getUnmarshaledEntryForBlock(ctx, safeIndex, deleteAt)
	if !found {
		// strictly-nearest-smaller previous entry not found, so test for the
		// exception: (first) future non-matured entry without current version.
		entry, found = fs.getUnmarshaledEntryForBlock(ctx, safeIndex, block)
		if found {
			// future delete without previous entry version: trim all
			entry.DeleteAt = block
			fs.trimFutureEntries(ctx, entry)
			return nil
		}
	}
	if !found || entry.HasDeleteAt() {
		return legacyerrors.ErrNotFound
	}

	entry.DeleteAt = block
	fs.setEntry(ctx, entry)

	// if we are deleting the entry at this block, then invoke deleteMarkedEntry()
	// to do the work. otherwise this is a future entry delete, so set a timer for
	// when it will become imminent.

	if block == ctxBlock {
		fs.deleteMarkedEntry(ctx, safeIndex, entry.Block)
	} else {
		key := encodeForTimer(safeIndex, entry.Block, timerDeleteEntry)
		fs.tstore.AddTimerByBlockHeight(ctx, block, key, []byte{})
	}

	// discard all pending future entries on or beyond the DeleteAt block, since they
	// will never become the latest; (newer entries on or beyond DeleteAt block may
	// be added with AppendEntry() going forward).

	fs.trimFutureEntries(ctx, entry)

	return nil
}

// removeEntry removes an entry from the store
func (fs *FixationStore) removeEntry(ctx sdk.Context, safeIndex SafeIndex, block uint64) {
	store := fs.getEntryStore(ctx, safeIndex)
	store.Delete(EncodeKey(block))
}

// putFutureEntry cancels a future entry from the store
func (fs *FixationStore) putFutureEntry(ctx sdk.Context, safeIndex SafeIndex, block uint64) {
	key := encodeForTimer(safeIndex, block, timerFutureEntry)
	fs.tstore.DelTimerByBlockHeight(ctx, block, key)
	fs.removeEntry(ctx, safeIndex, block)

	// in the unusual case that a future entry is added and then removed without
	// having turned latest or having another non-future entry added (i.e. there
	// has not yet been a "latest" entry), then we need to remove the EntryIndex
	// if we are the last in the chain.

	store := fs.getEntryStore(ctx, safeIndex)

	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	if !iterator.Valid() {
		fs.removeEntryIndex(ctx, safeIndex)
	}
}

func (fs *FixationStore) getEntryVersionsFilter(ctx sdk.Context, index string, block uint64, filter func(*Entry) bool) (blocks []uint64) {
	safeIndex, err := SanitizeIndex(index)
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
		var entry Entry
		fs.cdc.MustUnmarshal(iterator.Value(), &entry)

		if filter(&entry) {
			blocks = append(blocks, entry.Block)
		}

		if entry.Block <= block {
			break
		}
	}

	// reverse the result slice to return the blocks in ascending order
	length := len(blocks)
	for i := 0; i < length/2; i++ {
		blocks[i], blocks[length-i-1] = blocks[length-i-1], blocks[i]
	}

	return blocks
}

// GetEntryVersionsRange returns a list of versions from nearest-no-later block
// and onward, and not more than delta blocks further (skip stale entries).
func (fs *FixationStore) GetEntryVersionsRange(ctx sdk.Context, index string, block, delta uint64) (blocks []uint64) {
	filter := func(entry *Entry) bool {
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

// GetAllEntryVersions returns a list of all versions (blocks) of an entry, for
// use in testing and migrations (includes stale and deleted entry versions).
func (fs *FixationStore) GetAllEntryVersions(ctx sdk.Context, index string) (blocks []uint64) {
	filter := func(entry *Entry) bool { return true }
	return fs.getEntryVersionsFilter(ctx, index, 0, filter)
}

func (fs *FixationStore) createEntryStoreKey(index string) string {
	return fs.prefix + EntryPrefix + index
}

func (fs *FixationStore) Export(ctx sdk.Context) GenesisState {
	gs := GenesisState{}

	for _, index := range fs.AllEntryIndicesFilter(ctx, "", nil) {
		var entries GenesisEntries
		entries.Index = index
		safeIndex, err := SanitizeIndex(index)
		if err != nil {
			utils.LavaFormatPanic("export genesis failed", err)
		}
		entries.IsLive = fs.isEntryIndexLive(ctx, safeIndex)
		blocks := fs.GetAllEntryVersions(ctx, index)
		for _, block := range blocks {
			entry := fs.getEntry(ctx, safeIndex, block)
			entry.Index = index
			entries.Entries = append(entries.Entries, entry)
		}
		gs.Entries = append(gs.Entries, entries)
	}

	gs.Timerstore = fs.tstore.Export(ctx)

	return gs
}

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Version:    FixationVersion(),
		Timerstore: *timerstoretypes.DefaultGenesis(),
		Entries:    []GenesisEntries{},
	}
}

func (fs *FixationStore) Init(ctx sdk.Context, gs GenesisState) {
	for _, entries := range gs.Entries {
		safeIndex, err := SanitizeIndex(entries.Index)
		if err != nil {
			utils.LavaFormatPanic("cannot import genesis", err)
		}
		entries.Index = string(safeIndex)
		fs.setEntryIndex(ctx, SafeIndex(entries.Index), entries.IsLive)

		for _, entry := range entries.Entries {
			safeIndex, err := SanitizeIndex(entry.Index)
			if err != nil {
				utils.LavaFormatPanic("cannot import genesis", err)
			}
			entry.Index = string(safeIndex)
			fs.setEntry(ctx, entry)
		}
	}

	fs.tstore.Init(ctx, gs.Timerstore)
}

// NewFixationStore returns a new FixationStore object
func NewFixationStore(storeKey storetypes.StoreKey, cdc codec.BinaryCodec, prefix string, tstore *timerstoretypes.TimerStore, getStaleBlocks GetStaleBlocks) *FixationStore {
	fs := FixationStore{storeKey: storeKey, cdc: cdc, prefix: prefix}

	callback := func(ctx sdk.Context, key, data []byte) {
		fs.entryCallbackBeginBlock(ctx, key, data)
	}
	tstore.WithCallbackByBlockHeight(callback)

	fs.tstore = *tstore
	fs.getStaleBlocks = getStaleBlocks
	return &fs
}

// FixationStore manages lists of entries with versions in the store.
// (See also documentation in common/fixation_entry.go)
//
// EntryIndex tracks the indices of every Entry stored in the FixationStore
// (one index regardless of Entry version count). Its goal is to be able to
// efficiently retrieve a list of all Entry indices that are stored. (Without
// it, we would need to iterate through all the Entries *including* all their
// versions - which may be wasteful).
//
// EntryIndex objects are stored with a distinct prefix to avoid confusion
// when iterating over Entries. See example in `x/common/fixation_entry.go`.
//
// EntryIndex objects are stored with the key set to the (sanitized) index,
// and value normally set to EntryIndexLive, or EntryIndexDead if
// the respective Entry is marked deleted.

func (fs *FixationStore) getEntryIndexStore(ctx sdk.Context) *prefix.Store {
	store := prefix.NewStore(
		ctx.KVStore(fs.storeKey),
		KeyPrefix(fs.createEntryIndexStoreKey()))
	return &store
}

// setEntryIndex stores an Entry index in the store
func (fs FixationStore) setEntryIndex(ctx sdk.Context, safeIndex SafeIndex, live bool) {
	AssertSanitizedIndex(safeIndex, fs.prefix)
	store := fs.getEntryIndexStore(ctx)
	value := EntryIndexLive
	if !live {
		value = EntryIndexDead
	}
	store.Set(KeyPrefix(string(safeIndex)), value)
}

// removeEntryIndex removes an Entry index from the store
func (fs FixationStore) removeEntryIndex(ctx sdk.Context, safeIndex SafeIndex) {
	AssertSanitizedIndex(safeIndex, fs.prefix)
	store := fs.getEntryIndexStore(ctx)
	store.Delete(KeyPrefix(string(safeIndex)))
}

// isEntryIndexLive returns isLive from store
func (fs FixationStore) isEntryIndexLive(ctx sdk.Context, safeIndex SafeIndex) bool {
	AssertSanitizedIndex(safeIndex, fs.prefix)
	store := fs.getEntryIndexStore(ctx)
	status := store.Get(KeyPrefix(string(safeIndex)))
	return IsEntryIndexLive(status)
}

// AllEntryIndicesFilter returns all Entry indices with a given prefix and filtered
// by the given filter function.
func (fs FixationStore) AllEntryIndicesFilter(ctx sdk.Context, prefix string, filter func(k, v []byte) bool) []string {
	store := fs.getEntryIndexStore(ctx)
	entryPrefix := KeyPrefix(prefix)
	iterator := sdk.KVStorePrefixIterator(store, entryPrefix)
	defer iterator.Close()

	// iterate over the store's values and save the indices in a list
	indexList := []string{}
	for ; iterator.Valid(); iterator.Next() {
		key, value := iterator.Key(), iterator.Value()
		safeIndex := SafeIndex(key)
		AssertSanitizedIndex(safeIndex, fs.prefix)
		if filter == nil || filter(key, value) {
			indexList = append(indexList, DesanitizeIndex(safeIndex))
		}
	}

	return indexList
}

// GetAllEntryIndicesWithPrefix returns all Entry indices with a given prefix
func (fs FixationStore) GetAllEntryIndicesWithPrefix(ctx sdk.Context, prefix string) []string {
	filter := func(_, v []byte) bool { return IsEntryIndexLive(v) }
	return fs.AllEntryIndicesFilter(ctx, prefix, filter)
}

// GetAllEntryIndices returns all Entry indices
func (fs FixationStore) GetAllEntryIndices(ctx sdk.Context) []string {
	filter := func(_, v []byte) bool { return IsEntryIndexLive(v) }
	return fs.AllEntryIndicesFilter(ctx, "", filter)
}

func (fs FixationStore) createEntryIndexStoreKey() string {
	return fs.prefix + EntryIndexPrefix
}
