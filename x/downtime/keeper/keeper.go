package keeper

import (
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	gogowellknown "github.com/gogo/protobuf/types"
	"github.com/lavanet/lava/x/downtime/types"
	v1 "github.com/lavanet/lava/x/downtime/v1"
)

func NewKeeper(cdc codec.BinaryCodec, sk sdk.StoreKey, ps paramtypes.Subspace) Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(v1.ParamKeyTable())
	}
	return Keeper{
		storeKey:   sk,
		cdc:        cdc,
		paramstore: ps,
	}
}

type Keeper struct {
	storeKey   sdk.StoreKey
	cdc        codec.BinaryCodec
	paramstore paramtypes.Subspace
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
	return new(v1.GenesisState), nil
}

func (k Keeper) ImportGenesis(ctx sdk.Context, gs *v1.GenesisState) error {
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

func (k Keeper) SetLastBlockTime(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	protoTime, err := gogowellknown.TimestampProto(ctx.BlockTime())
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
	// if iter is not valid then there are no downtimes between the provided heights.
	if !iter.Valid() {
		return false, time.Duration(0)
	}
	// we had downtimes.
	totalDowntime := time.Duration(0)
	for ; iter.Valid(); iter.Next() {
		totalDowntime += k.unmarshalDuration(iter.Value())
	}
	return true, totalDowntime
}

func (k Keeper) SetDowntimeGarbageCollection(ctx sdk.Context, height uint64, gcTime time.Time) {
	ctx.KVStore(k.storeKey).
		Set(
			types.GetDowntimeGarbageKey(gcTime),
			sdk.Uint64ToBigEndian(height),
		)
}

// ------ STATE END -------

// RecordDowntime will record a downtime for the current block
func (k Keeper) RecordDowntime(ctx sdk.Context, duration time.Duration) {
	k.SetDowntime(ctx, uint64(ctx.BlockHeight()), duration)
	k.SetDowntimeGarbageCollection(ctx, uint64(ctx.BlockHeight()), ctx.BlockTime().Add(k.GetParams(ctx).GarbageCollectionDuration))
}

// GarbageCollectDowntimes will garbage collect downtimes.
func (k Keeper) GarbageCollectDowntimes(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	iter := store.Iterator(
		types.GetDowntimeGarbageKey(time.Unix(0, 0)),
		types.GetDowntimeGarbageKey(ctx.BlockTime()))

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
	k.GarbageCollectDowntimes(ctx)
	// we fetch the last block time
	lastBlockTime, ok := k.GetLastBlockTime(ctx)
	// if no last block time is known then it means that
	// this is the first time we're recording a block time,
	// so we just store the current block time and exit.
	if !ok {
		k.SetLastBlockTime(ctx)
		return
	}

	// if this is not the first time we're recording a block time
	// then we need to compare the current block time with the last
	// using the max downtime duration parameter.
	params := k.GetParams(ctx)
	maxDowntimeDuration := params.DowntimeDuration
	elapsedDuration := ctx.BlockTime().Sub(lastBlockTime)
	if elapsedDuration < maxDowntimeDuration {
		// if the current block time is less than the max downtime duration
		// then we just store the current block time and exit.
		k.SetLastBlockTime(ctx)
		return
	}
	// if the current block time is greater than the max downtime duration
	// it means that we had a downtime period, so we need to record it.
	k.RecordDowntime(ctx, elapsedDuration)
}
