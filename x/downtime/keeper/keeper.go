package keeper

import (
	"math"
	"time"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	gogowellknown "github.com/gogo/protobuf/types"
	"github.com/lavanet/lava/v2/x/downtime/types"
	v1 "github.com/lavanet/lava/v2/x/downtime/v1"
)

type EpochStorageKeeper interface {
	IsEpochStart(ctx sdk.Context) bool
	GetEpochStart(ctx sdk.Context) (epochStartBlock uint64)
	GetDeletedEpochs(ctx sdk.Context) []uint64
	GetEpochStartForBlock(ctx sdk.Context, block uint64) (epochStart uint64, blockInEpoch uint64, err error)
}

func NewKeeper(cdc codec.BinaryCodec, sk storetypes.StoreKey, ps paramtypes.Subspace, esk EpochStorageKeeper) Keeper {
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
	storeKey   storetypes.StoreKey
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

func (k Keeper) SetDowntime(ctx sdk.Context, epochStartBlock uint64, duration time.Duration) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetDowntimeKey(epochStartBlock)
	value := gogowellknown.DurationProto(duration)
	bz := k.cdc.MustMarshal(value)
	store.Set(key, bz)
}

func (k Keeper) GetDowntime(ctx sdk.Context, epochStartBlock uint64) (time.Duration, bool) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetDowntimeKey(epochStartBlock)
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

// ------ STATE END -------

// RecordDowntime will record a downtime for the current block
func (k Keeper) RecordDowntime(ctx sdk.Context, duration time.Duration) {
	epochStartBlock := k.epochStorageKeeper.GetEpochStart(ctx)
	// get epoch identifier
	cumulativeEpochDowntime, _ := k.GetDowntime(ctx, epochStartBlock)
	k.SetDowntime(ctx, epochStartBlock, duration+cumulativeEpochDowntime)
}

// GarbageCollectDowntimes will garbage collect downtimes.
func (k Keeper) GarbageCollectDowntimes(ctx sdk.Context) {
	for _, epoch := range k.epochStorageKeeper.GetDeletedEpochs(ctx) {
		k.DeleteDowntime(ctx, epoch)
	}
}

func (k Keeper) GetDowntimeFactor(ctx sdk.Context, epoch uint64) uint64 {
	duration, exists := k.GetDowntime(ctx, epoch)
	if !exists {
		return 1
	}
	epochDuration := k.GetParams(ctx).EpochDuration
	return uint64(duration/epochDuration) + 1
}

func (k Keeper) BeginBlock(ctx sdk.Context) {
	// this ensures that no matter the outcome, we will
	// reset the last block time to the current block time.
	defer func() {
		k.SetLastBlockTime(ctx, ctx.BlockTime())
	}()

	if k.epochStorageKeeper.IsEpochStart(ctx) {
		k.GarbageCollectDowntimes(ctx)
	}
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
