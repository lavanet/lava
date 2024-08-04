package keeper

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/epochstorage/types"
)

// SetFixatedParams set a specific fixatedParams in the store from its index
func (k Keeper) SetFixatedParams(ctx sdk.Context, fixatedParams types.FixatedParams) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.FixatedParamsKeyPrefix))
	b := k.cdc.MustMarshal(&fixatedParams)
	store.Set(types.FixatedParamsKey(
		fixatedParams.Index,
	), b)
}

// GetFixatedParams returns a fixatedParams from its index
func (k Keeper) GetFixatedParams(
	ctx sdk.Context,
	index string,
) (val types.FixatedParams, found bool) {
	if k.storeKey == nil {
		return val, false
	}
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.FixatedParamsKeyPrefix))

	b := store.Get(types.FixatedParamsKey(
		index,
	))
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveFixatedParams removes a fixatedParams from the store
func (k Keeper) RemoveFixatedParams(
	ctx sdk.Context,
	index string,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.FixatedParamsKeyPrefix))
	store.Delete(types.FixatedParamsKey(
		index,
	))
}

// GetAllFixatedParams returns all fixatedParams
func (k Keeper) GetAllFixatedParams(ctx sdk.Context) (list []types.FixatedParams) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.FixatedParamsKeyPrefix))
	iterator := sdk.KVStorePrefixIterator(store, []byte{})

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var val types.FixatedParams
		k.cdc.MustUnmarshal(iterator.Value(), &val)
		list = append(list, val)
	}

	return
}

func (k Keeper) fixatedParamsKey(fixatedKey string, index uint64) string {
	return fixatedKey + strconv.FormatUint(index, 10)
}

func (k Keeper) LatestFixatedParams(ctx sdk.Context, fixationKey string) (fixation types.FixatedParams, found bool) {
	return k.GetFixatedParams(ctx, k.fixatedParamsKey(fixationKey, 0))
}

func (k Keeper) FixateParams(ctx sdk.Context, block uint64) {
	latestParamChange := k.LatestParamChange(ctx)
	if latestParamChange == 0 { // no change
		return
	}
	if latestParamChange > block {
		utils.LavaFormatError("latest param change cant be in the future", fmt.Errorf("latestParamChange > block"),
			utils.Attribute{Key: "latestParamChange", Value: latestParamChange},
			utils.Attribute{Key: "block", Value: block},
		)
		return
	}
	earliestEpochStart := k.GetEarliestEpochStart(ctx) // this is the previous epoch start, before we update it to the current block
	if latestParamChange < earliestEpochStart {
		// latest param change is older than memory, so remove it
		k.paramstore.Set(ctx, types.KeyLatestParamChange, uint64(0))
		// clean up older fixated params, they no longer matter
		k.CleanAllOlderFixatedParams(ctx, 1) // everything after 0 is too old since there wasn't a param change in a while
		return
	}
	// we have a param change, is it in the last epoch?
	prevEpochStart, err := k.GetPreviousEpochStartForBlock(ctx, block)
	if err != nil {
		utils.LavaFormatError("can't get previous epoch start block", err,
			utils.Attribute{Key: "block", Value: block},
		)
	} else if latestParamChange >= prevEpochStart {
		// this is a recent change so we need to move the current fixation backwards
		k.PushFixatedParams(ctx, block, earliestEpochStart)
	}
}

func (k Keeper) PushFixatedParams(ctx sdk.Context, block, limit uint64) {
	for fixationKey, fixationGetParam := range k.fixationRegistries {
		currentParam := utils.Serialize(fixationGetParam(ctx))                // get the current param with the pointer function and serialize, TODO: usually getparam gets from the param store so we unmarshal than serialize, maybe we cam skip save one cast here
		currentFixatedParam, found := k.LatestFixatedParams(ctx, fixationKey) // get the fixater param and compare
		if found && bytes.Equal(currentParam, currentFixatedParam.Parameter) {
			continue
		}

		fixatedParamsToPush := types.FixatedParams{Parameter: currentParam, FixationBlock: block}
		found = true
		var olderParams types.FixatedParams
		var idx uint64
		var thisIdxKey string
		for idx = 0; found; idx++ { // we limit to 100 but never expect to get there
			thisIdxKey = k.fixatedParamsKey(fixationKey, idx)
			olderParams, found = k.GetFixatedParams(ctx, thisIdxKey)
			fixatedParamsToPush.Index = thisIdxKey
			k.SetFixatedParams(ctx, fixatedParamsToPush)
			// check if what we just set is enough to keep all the memory
			if fixatedParamsToPush.FixationBlock < limit {
				// if the fixatedParams we have in the list are too old, we dont need to store them any more
				k.CleanOlderFixatedParams(ctx, fixationKey, idx+1)
				break
			}
			fixatedParamsToPush = olderParams
		}
		utils.LogLavaEvent(ctx, k.Logger(ctx), types.FixatedParamChangeEventName, map[string]string{"moduleName": types.ModuleName, "block": strconv.FormatUint(block, 10), "fixatedParametersListLen": strconv.FormatUint(idx, 10), "fixationKey": fixationKey}, "params fixated after a change")
	}
}

func (k Keeper) CleanAllOlderFixatedParams(ctx sdk.Context, startIdx uint64) {
	for fixationKey := range k.fixationRegistries {
		k.CleanOlderFixatedParams(ctx, fixationKey, startIdx)
	}
}

func (k Keeper) CleanOlderFixatedParams(ctx sdk.Context, fixationKey string, startIdx uint64) {
	var idx uint64
	var thisIdxKey string
	for idx = startIdx; true; idx++ {
		thisIdxKey = k.fixatedParamsKey(fixationKey, idx)
		_, found := k.GetFixatedParams(ctx, thisIdxKey)
		if !found {
			break
		}
		k.RemoveFixatedParams(ctx, thisIdxKey)
	}
	utils.LogLavaEvent(ctx, k.Logger(ctx), types.FixatedParamCleanedEventName, map[string]string{"moduleName": types.ModuleName, "fixatedParametersListLen": thisIdxKey}, "fixation cleaned")
}

func (k Keeper) GetFixatedParamsForBlock(ctx sdk.Context, fixationKey string, block uint64) (fixated types.FixatedParams, err error) {
	fixationGetParam, ok := k.fixationRegistries[fixationKey]
	if !ok {
		return types.FixatedParams{}, fmt.Errorf("fixation not found for fixation key %s in fixation registers", fixationKey)
	}
	for idx := uint64(0); true; idx++ {
		thisIdxKey := k.fixatedParamsKey(fixationKey, idx)
		fixatedParams, found := k.GetFixatedParams(ctx, thisIdxKey)
		if !found {
			earliestEpochStart := k.GetEarliestEpochStart(ctx)
			if block < earliestEpochStart {
				err = utils.LavaFormatError("invalid block requested, that is lower than earliest block in memory", fmt.Errorf("tried to read for block that is earlier than earliest_epoch_start"),
					utils.Attribute{Key: "block", Value: block},
					utils.Attribute{Key: "earliest", Value: earliestEpochStart},
				)
			} else {
				err = utils.LavaFormatError("block not found", fmt.Errorf("tried to read index: "+thisIdxKey+" but wasn't found"),
					utils.Attribute{Key: "block", Value: block},
				)
			}
			break
		}
		if fixatedParams.FixationBlock <= block {
			// this means that the requested block is newer than the fixation, so we dont need to check older fixations
			return fixatedParams, nil
		}
	}
	// handle case of error with current params
	return types.FixatedParams{Parameter: utils.Serialize(fixationGetParam(ctx)), FixationBlock: block}, err
}

func (k Keeper) GetParamForBlock(ctx sdk.Context, fixationKey string, block uint64, param any) error {
	fixation, err := k.GetFixatedParamsForBlock(ctx, fixationKey, block)
	utils.Deserialize(fixation.Parameter, param)
	return err
}
