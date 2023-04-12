package common

import (
	"math"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common/types"
)

// TimerStore manages timers to efficiently support future timeouts. Timeouts
// can be based on either block-height or block-timestamp. When a timeout occurs,
// a designated callback function is called with the details (ctx and data).
//
// Once instantiated with NewTimerStore(), it offers the following methods:
//    - WithCallbackByBlockHeight(callback): sets the callback for block-height timers
//    - WithCallbackByBlockTime(callback): sets the callback for block-time timers
//    - AddTimerByBlockHeight(ctx, block, data): add a timer to expire at block height
//    - AddTimerByBlockTime(ctx, timestamp, data): add timer to expire at block timestamp
//    - Tick(ctx): advance the timer to the ctx's block (height and timestamp)
//
// How does it work? The explanation below illustrates how the data is stored, assuming
// the user is the module "package":
//
// 1. When instantiated, TimerStore gets a `prefix string` - used as a namespace to
// separate between instances of TimerStore. For instance, module "package" would
// use its module name for prefix.
//
// 2. TimerStore keeps the timers with a special prefix; it uses the timeout value
// (block height/timestamp) as the key, so that the standard iterator would yield them
// in the desired chronological order. TimeSStore also keeps the next-timeout values
// (of block height/timestamp) to efficiently determine if the iterator is needed.
// For instance, module "packages" may have two (block height) timeouts with their data
// set to "first" and "second" respectively:
//
//     prefix: package_Timer_Next_       key: BlockHeight      value: 150
//     prefix: package_Timer_Next_       key: BLockTimer       value: MaxInt64
//     prefix: package_Timer_Value_          key: 150          data: "first"
//     prefix: package_Entry_Value_          key: 180          data: "second"
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
//     tstore.AddTimerByBlockHeight(ctx, futureBlock1, "reason1")
//     tstore.AddTimerByBlockHeight(ctx, futureBlock2, "reason2")
//     ...
//
//     // usually called from a module's BeginBlock() callback
//     tstore.Tick(ctx)

type TimerCallback func(ctx sdk.Context, data string)

type TimerStore struct {
	storeKey  sdk.StoreKey
	cdc       codec.BinaryCodec
	prefix    string
	callbacks [2]TimerCallback // as per TimerType
}

func TimerVersion() uint64 {
	return 1
}

// NewTimerStore returns a new TimerStore object
func NewTimerStore(storeKey sdk.StoreKey, cdc codec.BinaryCodec, prefix string) *TimerStore {
	tstore := TimerStore{storeKey: storeKey, cdc: cdc, prefix: prefix}
	return &tstore
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

func (tstore *TimerStore) WithCallbackByBlockHeight(callback func(ctx sdk.Context, data string)) *TimerStore {
	tstoreNew := tstore
	tstoreNew.callbacks[types.BlockHeight] = callback
	return tstoreNew
}

func (tstore *TimerStore) WithCallbackByBlockTime(callback func(ctx sdk.Context, data string)) *TimerStore {
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
		return math.MaxInt64
	}
	return types.DecodeKey(b)
}

func (tstore *TimerStore) GetNextTimeoutBlockHeight(ctx sdk.Context) uint64 {
	return tstore.getNextTimeout(ctx, types.BlockHeight)
}

func (tstore *TimerStore) GetNextTimeoutBlockTime(ctx sdk.Context) uint64 {
	return tstore.getNextTimeout(ctx, types.BlockTime)
}

func (tstore *TimerStore) setNextTimeout(ctx sdk.Context, which types.TimerType, value uint64) {
	store := tstore.getStore(ctx, types.NextTimerPrefix)
	b := types.EncodeKey(value)
	store.Set([]byte(types.NextTimerKey[which]), b)
}

func (tstore *TimerStore) addTimer(ctx sdk.Context, which types.TimerType, value uint64, data string) {
	store := tstore.getStoreTimer(ctx, which)
	store.Set(types.EncodeKey(value), []byte(data))

	nextValue := tstore.getNextTimeout(ctx, which)
	if value < nextValue {
		tstore.setNextTimeout(ctx, which, value)
	}
}

func (tstore *TimerStore) delTimer(ctx sdk.Context, which types.TimerType, value uint64) {
	store := tstore.getStoreTimer(ctx, which)
	store.Delete(types.EncodeKey(value))
}

func (tstore *TimerStore) AddTimerByBlockHeight(ctx sdk.Context, block uint64, data string) {
	tstore.addTimer(ctx, types.BlockHeight, block, data)
}

func (tstore *TimerStore) AddTimerByBlockTime(ctx sdk.Context, timestamp uint64, data string) {
	tstore.addTimer(ctx, types.BlockTime, timestamp, data)
}

type timerTuple struct {
	value uint64
	data  string
}

func (tstore *TimerStore) tickValue(ctx sdk.Context, which types.TimerType, tickValue uint64) {
	nextValue := tstore.getNextTimeout(ctx, which)
	if tickValue < nextValue {
		return
	}

	store := tstore.getStoreTimer(ctx, which)

	iterator := sdk.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	var removals []timerTuple

	// iterate over the timers and collect those that expire: because the
	// key is block height/timestamp, the iterator yields entries ordered
	// by height/timestamp.

	for ; iterator.Valid(); iterator.Next() {
		value := types.DecodeKey(iterator.Key())
		if value > tickValue {
			// stop at first not-expired timer (update next timeout)
			tstore.setNextTimeout(ctx, which, value)
			break
		}
		tuple := timerTuple{value, string(iterator.Value())}
		removals = append(removals, tuple)
	}

	// if no more pending timers - then set next timeout to infinity
	if !iterator.Valid() {
		tstore.setNextTimeout(ctx, which, math.MaxInt64)
	}

	// iterates over expired timers: remote and invoke callback
	for _, tuple := range removals {
		tstore.delTimer(ctx, which, tuple.value)
		tstore.callbacks[which](ctx, tuple.data)
	}
}

func (tstore *TimerStore) Tick(ctx sdk.Context) {
	block := uint64(ctx.BlockHeight())
	tstore.tickValue(ctx, types.BlockHeight, block)

	timestamp := uint64(ctx.BlockTime().UTC().Unix())
	tstore.tickValue(ctx, types.BlockTime, timestamp)
}
