package keeper

import (
	"math"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	gogowellknown "github.com/gogo/protobuf/types"
	"github.com/lavanet/lava/x/downtime/types"
	v1 "github.com/lavanet/lava/x/downtime/v1"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

type EpochStorageKeeper interface {
	GetParams(ctx sdk.Context) (params epochstoragetypes.Params)
}

func NewKeeper(cdc codec.BinaryCodec, sk sdk.StoreKey, ps paramtypes.Subspace, esk EpochStorageKeeper) Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(v1.ParamKeyTable())
	}
	return Keeper{
		storeKey:           sk,
		cdc:                cdc,
		paramstore:         ps,
		epochStorageKeeper: esk,
	}
}

type Keeper struct {
	storeKey   sdk.StoreKey
	cdc        codec.BinaryCodec
	paramstore paramtypes.Subspace

	epochStorageKeeper EpochStorageKeeper
}

// ------ STATE -------

func (k Keeper) GetParams(ctx sdk.Context) (params v1.Params) {
	k.paramstore.GetParamSet(ctx, &params)
	return
}

func (k Keeper) SetParams(ctx sdk.Context, params v1.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

func (k Keeper) ExportGenesis(ctx sdk.Context) (*v1.GenesisState, error) {
	gs := new(v1.GenesisState)
	// get params
	gs.Params = k.GetParams(ctx)
	// get downtimes, the end is max uint64 -1 because we don't want to overflow uint64.
	k.IterateDowntimes(ctx, 0, math.MaxUint64-1, func(height uint64, downtime time.Duration) (stop bool) {
		gs.Downtimes = append(gs.Downtimes, &v1.Downtime{Block: height, Duration: downtime})
		return false
	})
	// get garbage collection
	k.IterateGarbageCollections(ctx, func(height uint64, gcBlock uint64) (stop bool) {
		gs.DowntimesGarbageCollection = append(gs.DowntimesGarbageCollection, &v1.DowntimeGarbageCollection{Block: height, GcBlock: gcBlock})
		return false
	})

	// get last block time, we report it only if it's set
	lastBlockTime, ok := k.GetLastBlockTime(ctx)
	if ok {
		gs.LastBlockTime = &lastBlockTime
	}

	return gs, nil
}

func (k Keeper) ImportGenesis(ctx sdk.Context, gs *v1.GenesisState) error {
	// set params
	k.SetParams(ctx, gs.Params)
	// set downtimes
	for _, downtime := range gs.Downtimes {
		k.SetDowntime(ctx, downtime.Block, downtime.Duration)
	}
	// set garbage collection
	for _, gc := range gs.DowntimesGarbageCollection {
		k.SetDowntimeGarbageCollection(ctx, gc.Block, gc.GcBlock)
	}
	// set last block time, only if it's present in genesis
	// otherwise it means we don't care about it. This can
	// happen when we are creating a new chain from scratch
	// for testing for example.
	if gs.LastBlockTime != nil {
		k.SetLastBlockTime(ctx, *gs.LastBlockTime)
	}
	return nil
}

func (k Keeper) GetLastBlockTime(ctx sdk.Context) (time.Time, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.LastBlockTimeKey)
	if bz == nil {
		return time.Time{}, false
	}
	protoTime := &gogowellknown.Timestamp{}
	k.cdc.MustUnmarshal(bz, protoTime)
	stdTime, err := gogowellknown.TimestampFromProto(protoTime)
	if err != nil {
		panic(err)
	}
	return stdTime, true
}

func (k Keeper) SetLastBlockTime(ctx sdk.Context, t time.Time) {
	store := ctx.KVStore(k.storeKey)
	protoTime, err := gogowellknown.TimestampProto(t)
	if err != nil {
		panic(err)
	}
	bz := k.cdc.MustMarshal(protoTime)
	store.Set(types.LastBlockTimeKey, bz)
}

func (k Keeper) SetDowntime(ctx sdk.Context, height uint64, duration time.Duration) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetDowntimeKey(height)
	value := gogowellknown.DurationProto(duration)
	bz := k.cdc.MustMarshal(value)
	store.Set(key, bz)
}

func (k Keeper) GetDowntime(ctx sdk.Context, height uint64) (time.Duration, bool) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetDowntimeKey(height)
	bz := store.Get(key)
	if bz == nil {
		return 0, false
	}
	return k.unmarshalDuration(bz), true
}

func (k Keeper) DeleteDowntime(ctx sdk.Context, height uint64) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetDowntimeKey(height)
	store.Delete(key)
}

func (k Keeper) unmarshalDuration(bz []byte) time.Duration {
	protoDuration := &gogowellknown.Duration{}
	k.cdc.MustUnmarshal(bz, protoDuration)
	stdDuration, err := gogowellknown.DurationFromProto(protoDuration)
	if err != nil {
		panic(err)
	}
	return stdDuration
}

// HadDowntimeBetween will return true if we had downtimes between the provided heights.
// The query is inclusive on both ends. The duration returned is the total downtime duration.
func (k Keeper) HadDowntimeBetween(ctx sdk.Context, startHeight, endHeight uint64) (bool, time.Duration) {
	var totalDuration time.Duration
	var hadDowntime bool
	k.IterateDowntimes(ctx, startHeight, endHeight, func(height uint64, duration time.Duration) bool {
		hadDowntime = true
		totalDuration += duration
		return false
	})
	return hadDowntime, totalDuration
}

// IterateDowntimes will iterate over all downtimes between the provided heights, it is inclusive on both ends.
// Will stop iteration when the callback returns true.
func (k Keeper) IterateDowntimes(ctx sdk.Context, startHeight, endHeight uint64, onResult func(height uint64, duration time.Duration) (stop bool)) {
	if startHeight > endHeight {
		panic("start height must be less than or equal to end height")
	}

	endHeightForQuery := endHeight + 1 // NOTE: KVStore Iterator end is exclusive, so we need to add 1.
	iter := ctx.KVStore(k.storeKey).
		Iterator(
			types.GetDowntimeKey(startHeight),
			types.GetDowntimeKey(endHeightForQuery),
		)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		height := types.ParseDowntimeKey(iter.Key())
		duration := k.unmarshalDuration(iter.Value())
		if onResult(height, duration) {
			break
		}
	}
}

func (k Keeper) SetDowntimeGarbageCollection(ctx sdk.Context, height uint64, gcBlock uint64) {
	ctx.KVStore(k.storeKey).
		Set(
			types.GetDowntimeGarbageKey(gcBlock),
			sdk.Uint64ToBigEndian(height),
		)
}

func (k Keeper) IterateGarbageCollections(ctx sdk.Context, onResult func(height uint64, gcBlock uint64) (stop bool)) {
	store := ctx.KVStore(k.storeKey)
	iter := store.Iterator(types.DowntimeHeightGarbageKey, sdk.PrefixEndBytes(types.DowntimeHeightGarbageKey))

	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		height := sdk.BigEndianToUint64(iter.Value())
		gcBlock := types.ParseDowntimeGarbageKey(iter.Key())
		if onResult(height, gcBlock) {
			break
		}
	}
}

// ------ STATE END -------

// GCBlocks returns the number of blocks a downtime should live.
func (k Keeper) GCBlocks(ctx sdk.Context) uint64 {
	p := k.epochStorageKeeper.GetParams(ctx)
	return p.EpochBlocks * p.EpochsToSave
}

// RecordDowntime will record a downtime for the current block
func (k Keeper) RecordDowntime(ctx sdk.Context, duration time.Duration) {
	k.SetDowntime(ctx, uint64(ctx.BlockHeight()), duration)
	k.SetDowntimeGarbageCollection(ctx, uint64(ctx.BlockHeight()), uint64(ctx.BlockHeight())+k.GCBlocks(ctx))
}

// GarbageCollectDowntimes will garbage collect downtimes.
func (k Keeper) GarbageCollectDowntimes(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	iter := store.Iterator(
		types.GetDowntimeGarbageKey(0),
		types.GetDowntimeGarbageKey(uint64(ctx.BlockHeight())+1), // NOTE: +1 because prefix end is exclusive.
	)

	defer iter.Close()
	keysToDelete := make([][]byte, 0) // this guarantees that we don't delete keys while iterating.
	for ; iter.Valid(); iter.Next() {
		height := sdk.BigEndianToUint64(iter.Value())
		k.DeleteDowntime(ctx, height) // clear downtime
		keysToDelete = append(keysToDelete, iter.Key())
	}
	for _, key := range keysToDelete {
		store.Delete(key)
	}
}

func (k Keeper) BeginBlock(ctx sdk.Context) {
	// this ensures that no matter the outcome, we will
	// reset the last block time to the current block time.
	defer func() {
		k.SetLastBlockTime(ctx, ctx.BlockTime())
	}()

	k.GarbageCollectDowntimes(ctx)
	// we fetch the last block time
	lastBlockTime, ok := k.GetLastBlockTime(ctx)
	// if no last block time is known then it means that
	// this is the first time we're recording a block time,
	// so we just store the current block time and exit.
	if !ok {
		return
	}

	// if this is not the first time we're recording a block time
	// then we need to compare the current block time with the last
	// using the max downtime duration parameter.
	params := k.GetParams(ctx)
	maxDowntimeDuration := params.DowntimeDuration
	elapsedDuration := ctx.BlockTime().Sub(lastBlockTime)
	// we didn't find any downtimes. So we exit.
	if elapsedDuration < maxDowntimeDuration {
		return
	}
	// if the current block time is greater than the max downtime duration
	// it means that we had a downtime period, so we need to record it.
	k.RecordDowntime(ctx, elapsedDuration)
}
