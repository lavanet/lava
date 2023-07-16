package common

import (
	"fmt"
	"math"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common/types"
)

// TimerStore manages timers to efficiently support future timeouts. Timeouts
// can be based on either block-height or block-timestamp. When a timeout occurs,
// a designated callback function is called with the details (ctx and data).
//
// Purpose and API:
//
// Once instantiated with NewTimerStore(), it offers the following methods:
//    - WithCallbackByBlockHeight(callback): sets the callback for block-height timers
//    - WithCallbackByBlockTime(callback): sets the callback for block-time timers
//    - AddTimerByBlockHeight(ctx, block, key, data): add a timer to expire at block height
//    - AddTimerByBlockTime(ctx, timestamp, key, data): add timer to expire at block timestamp
//    - HasTimerByBlockHeight(ctx, block, key): check whether a timer exists at block height
//    - HasTimerByBlockTime(ctx, timestamp, key): check whether a timer exists at block timestamp
//    - DelTimerByBlockHeight(ctx, block, key): delete a timer to expire at block height
//    - DelTimerByBlockTime(ctx, timestamp, key): delete timer to expire at block timestamp
//    - Tick(ctx): advance the timer to the ctx's block (height and timestamp)
//
// Usage and behavior:
//
// A timer store is instantiated using NewTimerStore(). Timeout handlers can be registered
// using WithCallbackBlockHeight() and WithCallbackBlockTime() for block-height-based and
// block-time-based timeouts, respectively.
// A timer is identified by a user defined _key_ and an _expiry block_ (or _block time_), and
// is associated with user defined _data_.
// A new timer is added using AddTimerByBlockHeight() and AddTimerByBlockTime(), respectively.
// When the expiry block (or block time) arrives, the respective callback will be invoked with
// the timer's _key_ and _data_. Adding the same timer again (i.e. same expiry block/block-time
// and same key) will overwrite the exiting timer's data.
// Trying to add a timer with expiry block not in the future, or expiry time not later than
// current block's timestamp, will cause a panic.
// Existence of a timer can be checked using HasTimerByBlockHeight() and HasTimerByBlockTimer(),
// respectively. These return true if a timer exists that matches the block/block-time and key.
// An existing timer can be deleted using DelTimerByBlockHeight() and AddTimerByBlockTime(),
// respectively. The timer to be deleted must exactly match the block/block-time and key. Trying
// to delete a non-existing timer will cause a panic.
// On every new block, Tick() should be called.
//
// The timer's _key_ is effectively the identifier of a timer. It decides whether a timer is
// new (to add) or existing (to modify), and to select timers to delete. It can also be used,
// for instance, to encode a timeout "type" using -say- the first byte to specify such "type".
//
// Example:
//     func callback(ctx sdk.Context, data string) {
//         println(data)
//     }
//
//     // create TimerStore with a block-height callback
//     tstore := timerstore.NewTimerStore(ctx).
//         WithCallbackByBlockHeight(callback)
//
//     ...
//     // start a new timer, the last argument will be provided to the callback
//     tstore.AddTimerByBlockHeight(ctx, futureBlock1, []byte("reason1"), []byte{0x1})
//     tstore.AddTimerByBlockHeight(ctx, futureBlock2, []byte("reason2"), []byte{0x2})
//     ...
//
//     // usually called from a module's BeginBlock() callback
//     tstore.Tick(ctx)
//
// Under the hood:
//
// The explanation below illustrates how data is stored, assuming module "package is "the user:
//
// 1. When instantiated, TimerStore gets a `prefix string` - used as a namespace to
// separate between instances of TimerStore. For instance, module "package" would
// use its module name for prefix.
//
// 2. TimerStore keeps the timers with a special prefix; it uses the timeout value (block
// height/timestamp) as the key prefix, so that the standard iterator would yield them
// in the desired chronological order. TimerStore also keeps the next-timeout values
// (of block height/timestamp) to efficiently determine if the iterator is needed.
// For instance, module "packages" may have three (block height) timeouts with their keys
// and data set to "first", "second" and "third" respectively:
//
//     prefix: package_Timer_Next_            key: BlockHeight      value: 150
//     prefix: package_Timer_Next_            key: BLockTimer       value: MaxUint64
//     prefix: package_Timer_Value_Block      key: 150_first        data: "first"
//     prefix: package_Timer_Value_Block      key: 180_second       data: "second"
//     prefix: package_Timer_Value_Block      key: 180_third        data: "third"
//
// 3. TimerStore tracks the next-timeout for both block-height/block-timestamp. On
// every call to Tick(), it tests the current ctx's block height/timestamp against the
// respective next-timeout:
//
// 4. If the next-timeout is reached/passed, then it will iterate through the timer
// entries and invoke the (respective) callback for those entries; And finally it will
// advance the (respective) next-timeout.
// If exact timeouts are needed, the user should call Tick() on every BeginBlock() of
// its own module. If timeouts may occur with delay (e.g. at start of an epoch), then
// the user may call Tick() at other deterministic intervals and reduce the workload.

// TimerCallback defined the callback handler function
type TimerCallback func(ctx sdk.Context, key []byte, data []byte)

// TimerStore represents a timer store to manager timers and timeouts
type TimerStore struct {
	storeKey  sdk.StoreKey
	cdc       codec.BinaryCodec
	prefix    string
	callbacks [2]TimerCallback // as per TimerType
}

// TimerVersion returns the timer library version
func TimerVersion() uint64 {
	return 1
}

// NewTimerStore returns a new TimerStore object
func NewTimerStore(storeKey sdk.StoreKey, cdc codec.BinaryCodec, prefix string) *TimerStore {
	tstore := TimerStore{storeKey: storeKey, cdc: cdc, prefix: prefix}
	return &tstore
}

func (tstore *TimerStore) Export(ctx sdk.Context) []types.RawMessage {
	store := prefix.NewStore(
		ctx.KVStore(tstore.storeKey),
		types.KeyPrefix(tstore.prefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	data := []types.RawMessage{}
	for ; iterator.Valid(); iterator.Next() {
		data = append(data, types.RawMessage{Key: iterator.Key(), Value: iterator.Value()})
	}

	return data
}

func (tstore *TimerStore) getVersion(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(tstore.storeKey), types.KeyPrefix(tstore.prefix))

	b := store.Get(types.KeyPrefix(types.TimerVersionKey))
	if b == nil {
		return 1
	}

	return types.DecodeKey(b)
}

func (tstore *TimerStore) setVersion(ctx sdk.Context, val uint64) {
	store := prefix.NewStore(ctx.KVStore(tstore.storeKey), types.KeyPrefix(tstore.prefix))
	b := types.EncodeKey(val)
	store.Set(types.KeyPrefix(types.TimerVersionKey), b)
}

// WithCallbackByBlockHeight sets a callback handler for timeouts (by block height)
func (tstore *TimerStore) WithCallbackByBlockHeight(callback TimerCallback) *TimerStore {
	tstoreNew := tstore
	tstoreNew.callbacks[types.BlockHeight] = callback
	return tstoreNew
}

// WithCallbackByBlockHeight sets a callback handler for timeouts (by block timestamp)
func (tstore *TimerStore) WithCallbackByBlockTime(callback TimerCallback) *TimerStore {
	tstoreNew := tstore
	tstoreNew.callbacks[types.BlockTime] = callback
	return tstoreNew
}

func (tstore *TimerStore) getStore(ctx sdk.Context, extraPrefix string) *prefix.Store {
	store := prefix.NewStore(
		ctx.KVStore(tstore.storeKey),
		types.KeyPrefix(tstore.prefix+extraPrefix),
	)
	return &store
}

func (tstore *TimerStore) getStoreTimer(ctx sdk.Context, which types.TimerType) *prefix.Store {
	prefix := types.TimerPrefix + types.TimerTypePrefix[which]
	return tstore.getStore(ctx, prefix)
}

func (tstore *TimerStore) getNextTimeout(ctx sdk.Context, which types.TimerType) uint64 {
	store := tstore.getStore(ctx, types.NextTimerPrefix)
	b := store.Get([]byte(types.NextTimerKey[which]))
	if len(b) == 0 {
		return math.MaxUint64
	}
	return types.DecodeKey(b)
}

func (tstore *TimerStore) getNextTimeoutBlockHeight(ctx sdk.Context) uint64 {
	return tstore.getNextTimeout(ctx, types.BlockHeight)
}

func (tstore *TimerStore) getNextTimeoutBlockTime(ctx sdk.Context) uint64 {
	return tstore.getNextTimeout(ctx, types.BlockTime)
}

func (tstore *TimerStore) setNextTimeout(ctx sdk.Context, which types.TimerType, value uint64) {
	store := tstore.getStore(ctx, types.NextTimerPrefix)
	b := types.EncodeKey(value)
	store.Set([]byte(types.NextTimerKey[which]), b)
}

func (tstore *TimerStore) addTimer(ctx sdk.Context, which types.TimerType, value uint64, key []byte, data []byte) {
	store := tstore.getStoreTimer(ctx, which)
	timerKey := types.EncodeBlockAndKey(value, key)
	store.Set(timerKey, data)

	nextValue := tstore.getNextTimeout(ctx, which)
	if value < nextValue {
		tstore.setNextTimeout(ctx, which, value)
	}
}

func (tstore *TimerStore) hasTimer(ctx sdk.Context, which types.TimerType, value uint64, key []byte) bool {
	store := tstore.getStoreTimer(ctx, which)
	timerKey := types.EncodeBlockAndKey(value, key)
	return store.Has(timerKey)
}

func (tstore *TimerStore) delTimer(ctx sdk.Context, which types.TimerType, value uint64, key []byte) {
	store := tstore.getStoreTimer(ctx, which)
	timerKey := types.EncodeBlockAndKey(value, key)
	if !store.Has(timerKey) {
		// panic:ok: caller should only try to delete existing timers
		// (use HasTimerByBlock{Height,Time} to check if a timer exists)
		panic(fmt.Sprintf("delTimer which %d block %d key %v: no such timer", which, value, key))
	}
	store.Delete(timerKey)
}

// AddTimerByBlockHeight adds a new timer to expire on a given block height.
// If a timer for that <block, key> tuple exists, it will be overridden.
func (tstore *TimerStore) AddTimerByBlockHeight(ctx sdk.Context, block uint64, key []byte, data []byte) {
	if block <= uint64(ctx.BlockHeight()) {
		// panic:ok: caller should never add a timer with past expiry
		panic(fmt.Sprintf("timer expiry block %d smaller than ctx block %d",
			block, uint64(ctx.BlockHeight())))
	}
	tstore.addTimer(ctx, types.BlockHeight, block, key, data)
}

// AddTimerByBlockTime adds a new timer to expire on a future block with the given timestamp.
// If a timer for that <timestamp, key> tuple exists, it will be overridden.
func (tstore *TimerStore) AddTimerByBlockTime(ctx sdk.Context, timestamp uint64, key []byte, data []byte) {
	if timestamp <= uint64(ctx.BlockTime().UTC().Unix()) {
		// panic:ok: caller should never add a timer with past expiry
		panic(fmt.Sprintf("timer expiry time %d smaller than ctx time %d",
			timestamp, uint64(ctx.BlockTime().UTC().Unix())))
	}
	tstore.addTimer(ctx, types.BlockTime, timestamp, key, data)
}

// HasTimerByBlockHeight checks whether a timer exists for the <block, key> tuple.
func (tstore *TimerStore) HasTimerByBlockHeight(ctx sdk.Context, block uint64, key []byte) bool {
	return tstore.hasTimer(ctx, types.BlockHeight, block, key)
}

// HasTimerByBlockTime checks whether a timer exists for the <timestamp, key> tuple.
func (tstore *TimerStore) HasTimerByBlockTime(ctx sdk.Context, timestamp uint64, key []byte) bool {
	return tstore.hasTimer(ctx, types.BlockTime, timestamp, key)
}

// DelTimerByBlockHeight removes an existing timer for the <block, key> tuple.
func (tstore *TimerStore) DelTimerByBlockHeight(ctx sdk.Context, block uint64, key []byte) {
	tstore.delTimer(ctx, types.BlockHeight, block, key)
}

// DelTimerByBlockTime removes an existing timer for the <timestamp, key> tuple.
func (tstore *TimerStore) DelTimerByBlockTime(ctx sdk.Context, timestamp uint64, key []byte) {
	tstore.delTimer(ctx, types.BlockTime, timestamp, key)
}

// DumpAllTimers dumps the details of all existing timers (of a type) into a string (for test/debug).
func (tstore *TimerStore) DumpAllTimers(ctx sdk.Context, which types.TimerType) string {
	store := tstore.getStoreTimer(ctx, which)

	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	var b strings.Builder

	for ; iterator.Valid(); iterator.Next() {
		value, key := types.DecodeBlockAndKey(iterator.Key())
		b.WriteString(fmt.Sprintf("block %d key %v data %v\n", value, key, iterator.Value()))
	}

	return b.String()
}

func (tstore *TimerStore) getFrontTimer(ctx sdk.Context, which types.TimerType) (uint64, []byte, []byte) {
	store := tstore.getStoreTimer(ctx, which)

	// because the key is block height/timestamp, the iterator yields entries
	// ordered by height/timestamp. so the front (earliest) one is the first.

	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	if !iterator.Valid() {
		return math.MaxUint64, []byte{}, []byte{}
	}

	value, key := types.DecodeBlockAndKey(iterator.Key())
	return value, key, iterator.Value()
}

func (tstore *TimerStore) tickValue(ctx sdk.Context, which types.TimerType, tickValue uint64) {
	nextValue := tstore.getNextTimeout(ctx, which)
	if tickValue < nextValue {
		return
	}

	// iterate over the timers and collect those that expire: can not use the
	// KVStore iterator because it cannot handle callbacks added/removed during
	// its operation. instead, repeatedly get the front (earliest) entry.

	for {
		value, key, data := tstore.getFrontTimer(ctx, which)

		if value > tickValue {
			// stop at first not-expired timer (update next timeout)
			tstore.setNextTimeout(ctx, which, value)
			break
		}

		tstore.delTimer(ctx, which, value, key)
		tstore.callbacks[which](ctx, key, data)
	}
}

// Tick advances the timer by a block. It should be called at the beginning of each block.
func (tstore *TimerStore) Tick(ctx sdk.Context) {
	block := uint64(ctx.BlockHeight())
	tstore.tickValue(ctx, types.BlockHeight, block)

	timestamp := uint64(ctx.BlockTime().UTC().Unix())
	tstore.tickValue(ctx, types.BlockTime, timestamp)
}
