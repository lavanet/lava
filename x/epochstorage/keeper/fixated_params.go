package keeper

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/epochstorage/types"
)

const (
	FixatedKey = "current"
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

func (k Keeper) fixatedParamsKey(index uint64) string {
	switch index {
	default:
		return FixatedKey + strconv.FormatUint(index, 10)
	}
}

func (k Keeper) PushFixatedParams(ctx sdk.Context, block uint64, limit uint64) {
	currentParams := k.GetParams(ctx)
	fixatedParamsToPush := types.FixatedParams{Parameters: currentParams, FixationBlock: block}
	found := true
	var olderParams types.FixatedParams
	var idx uint64
	for idx = 0; found; idx++ { //we limit to 100 but never expect to get there
		thisIdxKey := k.fixatedParamsKey(idx)
		olderParams, found = k.GetFixatedParams(ctx, thisIdxKey)
		fixatedParamsToPush.Index = thisIdxKey
		k.SetFixatedParams(ctx, fixatedParamsToPush)
		fixatedParamsToPush = olderParams
		if olderParams.FixationBlock < limit {
			// if the fixatedParams we have in the list are too old, we dont need to store them any more
			k.CleanOlderFixatedParams(ctx, idx+1)
			break
		}
	}
	utils.LogLavaEvent(ctx, k.Logger(ctx), "fixated_params_after_change", map[string]string{"moduleName": types.ModuleName, "block": strconv.FormatUint(block, 10), "fixatedParametersListLen": strconv.FormatUint(idx, 10)}, "params fixated after a change")
}

func (k Keeper) CleanOlderFixatedParams(ctx sdk.Context, startIdx uint64) {
	for idx := uint64(startIdx); true; idx++ { //we limit to 100 but never expect to get there
		thisIdxKey := k.fixatedParamsKey(idx)
		_, found := k.GetFixatedParams(ctx, thisIdxKey)
		if !found {
			break
		}
		k.RemoveFixatedParams(ctx, thisIdxKey)
	}
}

func (k Keeper) GetFixatedParamsForBlock(ctx sdk.Context, block uint64) types.Params {
	for idx := uint64(0); true; idx++ {
		thisIdxKey := k.fixatedParamsKey(idx)
		fixatedParams, found := k.GetFixatedParams(ctx, thisIdxKey)
		if fixatedParams.FixationBlock <= block {
			// this means that the requested block is newer than the fixation, so we dont need to check older fixations
			return fixatedParams.Parameters
		}
		if !found {
			utils.LavaError(ctx, k.Logger(ctx), "fixated_params_for_block_empty", map[string]string{"error": "tried to read index: " + thisIdxKey + " but wasn't found", "block": strconv.FormatUint(block, 10)}, "invalid block requested, that is lower than saved fixation memory")
			break
		}
	}
	return k.GetParams(ctx) //this is very bad it returns the latest for osmehting that is older, but thats the best we can do
}
