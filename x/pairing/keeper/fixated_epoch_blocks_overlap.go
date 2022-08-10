package keeper

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/pairing/types"
)

// SetFixatedEpochBlocksOverlap set a specific fixatedEpochBlocksOverlap in the store from its index
func (k Keeper) SetFixatedEpochBlocksOverlap(ctx sdk.Context, fixatedEpochBlocksOverlap types.FixatedEpochBlocksOverlap) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.FixatedEpochBlocksOverlapKeyPrefix))
	b := k.cdc.MustMarshal(&fixatedEpochBlocksOverlap)
	store.Set(types.FixatedEpochBlocksOverlapKey(
		fixatedEpochBlocksOverlap.Index,
	), b)
}

// GetFixatedEpochBlocksOverlap returns a fixatedEpochBlocksOverlap from its index
func (k Keeper) GetFixatedEpochBlocksOverlap(
	ctx sdk.Context,
	index string,

) (val types.FixatedEpochBlocksOverlap, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.FixatedEpochBlocksOverlapKeyPrefix))

	b := store.Get(types.FixatedEpochBlocksOverlapKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveFixatedEpochBlocksOverlap removes a fixatedEpochBlocksOverlap from the store
func (k Keeper) RemoveFixatedEpochBlocksOverlap(
	ctx sdk.Context,
	index string,

) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.FixatedEpochBlocksOverlapKeyPrefix))
	store.Delete(types.FixatedEpochBlocksOverlapKey(
		index,
	))
}

// GetAllFixatedEpochBlocksOverlap returns all fixatedEpochBlocksOverlap
func (k Keeper) GetAllFixatedEpochBlocksOverlap(ctx sdk.Context) (list []types.FixatedEpochBlocksOverlap) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.FixatedEpochBlocksOverlapKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.FixatedEpochBlocksOverlap
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) FixatedEpochBlocksOverlapKey(index uint64) string {
	switch index {
	default:
		return FixatedKey + strconv.FormatUint(index, 10)
	}
}

func (k Keeper) LatestFixatedEpochBlocksOverlap(ctx sdk.Context) (fixation types.FixatedEpochBlocksOverlap, found bool) {
	return k.GetFixatedEpochBlocksOverlap(ctx, k.FixatedEpochBlocksOverlapKey(0))
}

func (k Keeper) PushFixatedEpochBlocksOverlap(ctx sdk.Context, block uint64, limit uint64) {
	currentEpochBlocksOverlap := k.GetParams(ctx).EpochBlocksOverlap
	FixatedEpochBlocksOverlapToPush := types.FixatedEpochBlocksOverlap{EpochBlocksOverlap: currentEpochBlocksOverlap, FixationBlock: block}
	found := true
	var olderParams types.FixatedEpochBlocksOverlap
	var idx uint64
	for idx = 0; found; idx++ { //we limit to 100 but never expect to get there
		thisIdxKey := k.FixatedEpochBlocksOverlapKey(idx)
		olderParams, found = k.GetFixatedEpochBlocksOverlap(ctx, thisIdxKey)
		FixatedEpochBlocksOverlapToPush.Index = thisIdxKey
		k.SetFixatedEpochBlocksOverlap(ctx, FixatedEpochBlocksOverlapToPush)
		//check if what we just set is enough to keep all the memory
		if FixatedEpochBlocksOverlapToPush.FixationBlock < limit {
			// if the FixatedEpochBlocksOverlap we have in the list are too old, we dont need to store them any more
			k.CleanOlderFixatedEpochBlocksOverlap(ctx, idx+1)
			break
		}
		FixatedEpochBlocksOverlapToPush = olderParams
	}
	utils.LogLavaEvent(ctx, k.Logger(ctx), "fixated_epochblocksoverlap_after_change", map[string]string{"moduleName": types.ModuleName, "block": strconv.FormatUint(block, 10), "fixatedParametersListLen": strconv.FormatUint(idx, 10)}, "params fixated after a change")
}

func (k Keeper) CleanOlderFixatedEpochBlocksOverlap(ctx sdk.Context, startIdx uint64) {
	var idx uint64
	for idx = uint64(startIdx); true; idx++ {
		thisIdxKey := k.FixatedEpochBlocksOverlapKey(idx)
		_, found := k.GetFixatedEpochBlocksOverlap(ctx, thisIdxKey)
		if !found {
			break
		}
		k.RemoveFixatedEpochBlocksOverlap(ctx, thisIdxKey)
	}
	utils.LogLavaEvent(ctx, k.Logger(ctx), "clean_fixated_epochblocksoverlap", map[string]string{"moduleName": types.ModuleName, "fixatedParametersListLen": strconv.FormatUint(idx, 10)}, "fixation cleaned")
}

func (k Keeper) GetFixatedEpochBlocksOverlapForBlock(ctx sdk.Context, block uint64) (fixated types.FixatedEpochBlocksOverlap, err error) {
	for idx := uint64(0); true; idx++ {
		thisIdxKey := k.FixatedEpochBlocksOverlapKey(idx)
		FixatedEpochBlocksOverlap, found := k.GetFixatedEpochBlocksOverlap(ctx, thisIdxKey)
		if !found {
			earliestEpochStart := k.epochStorageKeeper.GetEarliestEpochStart(ctx)
			if block < earliestEpochStart {
				err = utils.LavaError(ctx, k.Logger(ctx), "fixated_epochblocksoverlap_too_early", map[string]string{"error": "tried to read for block that is earlier than earliest_epoch_start", "block": strconv.FormatUint(block, 10), "earliest": strconv.FormatUint(earliestEpochStart, 10)}, "invalid block requested, that is lower than earliest block in memory")
			} else {
				err = utils.LavaError(ctx, k.Logger(ctx), "fixated_epochblocksoverlap_for_block_empty", map[string]string{"error": "tried to read index: " + thisIdxKey + " but wasn't found", "block": strconv.FormatUint(block, 10)}, "invalid block requested, that is lower than saved fixation memory")
			}
			break
		}
		if FixatedEpochBlocksOverlap.FixationBlock <= block {
			// this means that the requested block is newer than the fixation, so we dont need to check older fixations
			return FixatedEpochBlocksOverlap, nil
		}
	}
	//handle case of error with current params
	return types.FixatedEpochBlocksOverlap{EpochBlocksOverlap: k.GetParams(ctx).EpochBlocksOverlap, FixationBlock: block}, err
}
