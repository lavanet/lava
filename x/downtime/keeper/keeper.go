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
	protoDuration := &gogowellknown.Duration{}
	k.cdc.MustUnmarshal(bz, protoDuration)
	stdDuration, err := gogowellknown.DurationFromProto(protoDuration)
	if err != nil {
		panic(err)
	}
	return stdDuration, true
}

// ------ STATE END -------

func (k Keeper) BeginBlock(ctx sdk.Context) {
	// we fetch the last block time
	lastBlockTime, ok := k.GetLastBlockTime(ctx)
	// if no last block time is known then it means that
	// this is the first time we're recording a block time
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
	if ctx.BlockTime().Sub(lastBlockTime) < maxDowntimeDuration {
		// if the current block time is less than the max downtime duration
		// then we just store the current block time and exit.
		k.SetLastBlockTime(ctx)
		return
	}
	// if the current block time is greater than the max downtime duration
	// it means that we had a downtime period and we need to record it.
	panic("TODO: record downtime period")
}
