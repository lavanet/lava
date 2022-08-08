package keeper

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/pairing/types"
)

const (
	FixatedKey = "current"
)

// SetFixatedServicersToPair set a specific fixatedServicersToPair in the store from its index
func (k Keeper) SetFixatedServicersToPair(ctx sdk.Context, fixatedServicersToPair types.FixatedServicersToPair) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.FixatedServicersToPairKeyPrefix))
	b := k.cdc.MustMarshal(&fixatedServicersToPair)
	store.Set(types.FixatedServicersToPairKey(
		fixatedServicersToPair.Index,
	), b)
}

// GetFixatedServicersToPair returns a fixatedServicersToPair from its index
func (k Keeper) GetFixatedServicersToPair(
	ctx sdk.Context,
	index string,

) (val types.FixatedServicersToPair, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.FixatedServicersToPairKeyPrefix))

	b := store.Get(types.FixatedServicersToPairKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveFixatedServicersToPair removes a fixatedServicersToPair from the store
func (k Keeper) RemoveFixatedServicersToPair(
	ctx sdk.Context,
	index string,

) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.FixatedServicersToPairKeyPrefix))
	store.Delete(types.FixatedServicersToPairKey(
		index,
	))
}

// GetAllFixatedServicersToPair returns all fixatedServicersToPair
func (k Keeper) GetAllFixatedServicersToPair(ctx sdk.Context) (list []types.FixatedServicersToPair) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.FixatedServicersToPairKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.FixatedServicersToPair
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) FixatedServicersToPairKey(index uint64) string {
	switch index {
	default:
		return FixatedKey + strconv.FormatUint(index, 10)
	}
}

func (k Keeper) LatestFixatedServicersToPair(ctx sdk.Context) (fixation types.FixatedServicersToPair, found bool) {
	return k.GetFixatedServicersToPair(ctx, k.FixatedServicersToPairKey(0))
}

func (k Keeper) PushFixatedServicersToPair(ctx sdk.Context, block uint64, limit uint64) {
	currentServicersToPairCount := k.GetParams(ctx).ServicersToPairCount
	FixatedServicersToPairToPush := types.FixatedServicersToPair{ServicersToPairCount: currentServicersToPairCount, FixationBlock: block}
	found := true
	var olderParams types.FixatedServicersToPair
	var idx uint64
	for idx = 0; found; idx++ { //we limit to 100 but never expect to get there
		thisIdxKey := k.FixatedServicersToPairKey(idx)
		olderParams, found = k.GetFixatedServicersToPair(ctx, thisIdxKey)
		FixatedServicersToPairToPush.Index = thisIdxKey
		k.SetFixatedServicersToPair(ctx, FixatedServicersToPairToPush)
		//check if what we just set is enough to keep all the memory
		if FixatedServicersToPairToPush.FixationBlock < limit {
			// if the FixatedServicersToPair we have in the list are too old, we dont need to store them any more
			k.CleanOlderFixatedServicersToPair(ctx, idx+1)
			break
		}
		FixatedServicersToPairToPush = olderParams
	}
	utils.LogLavaEvent(ctx, k.Logger(ctx), "fixated_params_after_change", map[string]string{"moduleName": types.ModuleName, "block": strconv.FormatUint(block, 10), "fixatedParametersListLen": strconv.FormatUint(idx, 10)}, "params fixated after a change")
}

func (k Keeper) CleanOlderFixatedServicersToPair(ctx sdk.Context, startIdx uint64) {
	var idx uint64
	for idx = uint64(startIdx); true; idx++ {
		thisIdxKey := k.FixatedServicersToPairKey(idx)
		_, found := k.GetFixatedServicersToPair(ctx, thisIdxKey)
		if !found {
			break
		}
		k.RemoveFixatedServicersToPair(ctx, thisIdxKey)
	}
	utils.LogLavaEvent(ctx, k.Logger(ctx), "clean_fixated_params", map[string]string{"moduleName": types.ModuleName, "fixatedParametersListLen": strconv.FormatUint(idx, 10)}, "fixation cleaned")
}

func (k Keeper) GetFixatedServicersToPairForBlock(ctx sdk.Context, block uint64) (fixated types.FixatedServicersToPair, err error) {
	for idx := uint64(0); true; idx++ {
		thisIdxKey := k.FixatedServicersToPairKey(idx)
		FixatedServicersToPair, found := k.GetFixatedServicersToPair(ctx, thisIdxKey)
		if !found {
			earliestEpochStart := k.epochStorageKeeper.GetEarliestEpochStart(ctx)
			if block < earliestEpochStart {
				err = utils.LavaError(ctx, k.Logger(ctx), "fixated_params_too_early", map[string]string{"error": "tried to read for block that is earlier than earliest_epoch_start", "block": strconv.FormatUint(block, 10), "earliest": strconv.FormatUint(earliestEpochStart, 10)}, "invalid block requested, that is lower than earliest block in memory")
			} else {
				err = utils.LavaError(ctx, k.Logger(ctx), "fixated_params_for_block_empty", map[string]string{"error": "tried to read index: " + thisIdxKey + " but wasn't found", "block": strconv.FormatUint(block, 10)}, "invalid block requested, that is lower than saved fixation memory")
			}
			break
		}
		if FixatedServicersToPair.FixationBlock <= block {
			// this means that the requested block is newer than the fixation, so we dont need to check older fixations
			return FixatedServicersToPair, nil
		}
	}
	//handle case of error with current params
	return types.FixatedServicersToPair{ServicersToPairCount: k.GetParams(ctx).ServicersToPairCount, FixationBlock: block}, err
}
