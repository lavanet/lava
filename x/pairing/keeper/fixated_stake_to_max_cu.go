package keeper

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/pairing/types"
)

// SetFixatedStakeToMaxCu set a specific fixatedStakeToMaxCu in the store from its index
func (k Keeper) SetFixatedStakeToMaxCu(ctx sdk.Context, fixatedStakeToMaxCu types.FixatedStakeToMaxCu) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.FixatedStakeToMaxCuKeyPrefix))
	b := k.cdc.MustMarshal(&fixatedStakeToMaxCu)
	store.Set(types.FixatedStakeToMaxCuKey(
		fixatedStakeToMaxCu.Index,
	), b)
}

// GetFixatedStakeToMaxCu returns a fixatedStakeToMaxCu from its index
func (k Keeper) GetFixatedStakeToMaxCu(
	ctx sdk.Context,
	index string,

) (val types.FixatedStakeToMaxCu, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.FixatedStakeToMaxCuKeyPrefix))

	b := store.Get(types.FixatedStakeToMaxCuKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveFixatedStakeToMaxCu removes a fixatedStakeToMaxCu from the store
func (k Keeper) RemoveFixatedStakeToMaxCu(
	ctx sdk.Context,
	index string,

) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.FixatedStakeToMaxCuKeyPrefix))
	store.Delete(types.FixatedStakeToMaxCuKey(
		index,
	))
}

// GetAllFixatedStakeToMaxCu returns all fixatedStakeToMaxCu
func (k Keeper) GetAllFixatedStakeToMaxCu(ctx sdk.Context) (list []types.FixatedStakeToMaxCu) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.FixatedStakeToMaxCuKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.FixatedStakeToMaxCu
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) FixatedStakeToMaxCuKey(index uint64) string {
	switch index {
	default:
		return FixatedKey + strconv.FormatUint(index, 10)
	}
}

func (k Keeper) LatestFixatedStakeToMaxCu(ctx sdk.Context) (fixation types.FixatedStakeToMaxCu, found bool) {
	return k.GetFixatedStakeToMaxCu(ctx, k.FixatedStakeToMaxCuKey(0))
}

func (k Keeper) PushFixatedStakeToMaxCu(ctx sdk.Context, block uint64, limit uint64) {
	currentStakeToMaxCu := k.GetParams(ctx).StakeToMaxCUList
	FixatedStakeToMaxCuToPush := types.FixatedStakeToMaxCu{StakeToMaxCUList: currentStakeToMaxCu, FixationBlock: block}
	found := true
	var olderParams types.FixatedStakeToMaxCu
	var idx uint64
	for idx = 0; found; idx++ { //we limit to 100 but never expect to get there
		thisIdxKey := k.FixatedStakeToMaxCuKey(idx)
		olderParams, found = k.GetFixatedStakeToMaxCu(ctx, thisIdxKey)
		FixatedStakeToMaxCuToPush.Index = thisIdxKey
		k.SetFixatedStakeToMaxCu(ctx, FixatedStakeToMaxCuToPush)
		//check if what we just set is enough to keep all the memory
		if FixatedStakeToMaxCuToPush.FixationBlock < limit {
			// if the FixatedStakeToMaxCu we have in the list are too old, we dont need to store them any more
			k.CleanOlderFixatedStakeToMaxCu(ctx, idx+1)
			break
		}
		FixatedStakeToMaxCuToPush = olderParams
	}
	utils.LogLavaEvent(ctx, k.Logger(ctx), "fixated_staketomaxcu_after_change", map[string]string{"moduleName": types.ModuleName, "block": strconv.FormatUint(block, 10), "fixatedParametersListLen": strconv.FormatUint(idx, 10)}, "params fixated after a change")
}

func (k Keeper) CleanOlderFixatedStakeToMaxCu(ctx sdk.Context, startIdx uint64) {
	var idx uint64
	for idx = uint64(startIdx); true; idx++ {
		thisIdxKey := k.FixatedStakeToMaxCuKey(idx)
		_, found := k.GetFixatedStakeToMaxCu(ctx, thisIdxKey)
		if !found {
			break
		}
		k.RemoveFixatedStakeToMaxCu(ctx, thisIdxKey)
	}
	utils.LogLavaEvent(ctx, k.Logger(ctx), "clean_fixated_staketomaxcu", map[string]string{"moduleName": types.ModuleName, "fixatedstaketomaxcuListLen": strconv.FormatUint(idx, 10)}, "fixation cleaned")
}

func (k Keeper) GetFixatedStakeToMaxCuForBlock(ctx sdk.Context, block uint64) (fixated types.FixatedStakeToMaxCu, err error) {
	for idx := uint64(0); true; idx++ {
		thisIdxKey := k.FixatedStakeToMaxCuKey(idx)
		FixatedStakeToMaxCu, found := k.GetFixatedStakeToMaxCu(ctx, thisIdxKey)
		if !found {
			earliestEpochStart := k.epochStorageKeeper.GetEarliestEpochStart(ctx)
			if block < earliestEpochStart {
				err = utils.LavaError(ctx, k.Logger(ctx), "fixated_staketomaxcu_too_early", map[string]string{"error": "tried to read for block that is earlier than earliest_epoch_start", "block": strconv.FormatUint(block, 10), "earliest": strconv.FormatUint(earliestEpochStart, 10)}, "invalid block requested, that is lower than earliest block in memory")
			} else {
				err = utils.LavaError(ctx, k.Logger(ctx), "fixated_staketomaxcu_for_block_empty", map[string]string{"error": "tried to read index: " + thisIdxKey + " but wasn't found", "block": strconv.FormatUint(block, 10)}, "invalid block requested, that is lower than saved fixation memory")
			}
			break
		}
		if FixatedStakeToMaxCu.FixationBlock <= block {
			// this means that the requested block is newer than the fixation, so we dont need to check older fixations
			return FixatedStakeToMaxCu, nil
		}
	}
	//handle case of error with current params
	return types.FixatedStakeToMaxCu{StakeToMaxCUList: k.GetParams(ctx).StakeToMaxCUList, FixationBlock: block}, err
}
