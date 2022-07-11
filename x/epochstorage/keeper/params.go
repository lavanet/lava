package keeper

import (
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/epochstorage/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(
		k.UnstakeHoldBlocks(ctx),
		k.EpochBlocks(ctx),
		k.EpochsToSave(ctx),
		k.LatestParamChange(ctx),
	)
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

// UnstakeHoldBlocks returns the UnstakeHoldBlocks param
func (k Keeper) UnstakeHoldBlocks(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyUnstakeHoldBlocks, &res)
	return
}

// EpochBlocks returns the EpochBlocks param
func (k Keeper) EpochBlocks(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyEpochBlocks, &res)
	return
}

// EpochsToSave returns the EpochsToSave param
func (k Keeper) EpochsToSave(ctx sdk.Context) (res uint64) {
	//TODO: change to support param change
	k.paramstore.Get(ctx, types.KeyEpochsToSave, &res)
	return
}

// EpochBlocks returns the EpochBlocks param
func (k Keeper) GetEpochBlocks(ctx sdk.Context, block uint64) (res uint64) {
	//TODO: modify to support param change
	k.paramstore.Get(ctx, types.KeyEpochBlocks, &res)
	return
}

// return the next epoch start
func (k Keeper) IsEpochStart(ctx sdk.Context) (res bool) {
	blocksCycle := k.EpochBlocks(ctx)
	currentBlock := uint64(ctx.BlockHeight())
	//current block modulu blocks cycle returns how many block in the current epoch we are, if its 0 we are at epoch start
	if blocksCycle == 0 {
		return false
	}
	return (currentBlock % blocksCycle) == 0
}

func (k Keeper) BlocksToSave(ctx sdk.Context) (res uint64) {
	blocksToSave := k.EpochsToSave(ctx) * k.EpochBlocks(ctx)
	return blocksToSave
}

func (k Keeper) BlockInEpoch(ctx sdk.Context, block uint64) (res uint64) {
	blocksCycle := k.GetEpochBlocks(ctx, block)
	return block % blocksCycle
}

func (k Keeper) LatestParamChange(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyLatestParamChange, &res)
	return
}

// GetSessionStartForBlock gets a session start supports one param change
func (k Keeper) GetEpochStartForBlock(ctx sdk.Context, block uint64) (epochStart uint64, blockInEpoch uint64) {
	blocksInTargetSession := k.BlockInEpoch(ctx, block)
	targetBlockStart := block - blocksInTargetSession
	return targetBlockStart, blocksInTargetSession
}

func (k Keeper) GetPreviousEpochStartForBlock(ctx sdk.Context, block uint64) (previousEpochStart uint64) {
	epochStart, _ := k.GetEpochStartForBlock(ctx, block)
	previousEpochStart, _ = k.GetEpochStartForBlock(ctx, epochStart-1) //we take one block before the target epoch so it belongs to the previous epoch
	return
}

func (k Keeper) FixateParams(ctx sdk.Context, block uint64) {
	latestParamChange := k.LatestParamChange(ctx)
	if latestParamChange == 0 { // no change
		return
	}
	if latestParamChange > block {
		utils.LavaError(ctx, k.Logger(ctx), "invalid_latest_param_change", map[string]string{"error": "latestParamChange > block", "latestParamChange": strconv.FormatUint(latestParamChange, 10)}, "latest param change cant be in the future")
		return
	}
	if latestParamChange < k.GetEarliestEpochStart(ctx) {
		//latest param change is older than memory, so remove it
		k.paramstore.Set(ctx, types.KeyLatestParamChange, 0)
		//clean up older fixated params, they no longer matter
		k.CleanOlderFixatedParams(ctx)
		return
	}
	// we have a param change, is it in the last epoch?
	if latestParamChange > k.GetPreviousEpochStartForBlock(ctx, block) {
		// this is a recent change so we need to move the current fixation backwards
		k.PushFixatedParams(ctx, block)
	}
}

func (k Keeper) PushFixatedParams(ctx sdk.Context, block uint64) {

	utils.LogLavaEvent(ctx, k.Logger(ctx), "fixated_params_after_change", map[string]string{"moduleName": types.ModuleName, "block": strconv.FormatUint(block, 10)}, "params fixated after a change")
}

func (k Keeper) CleanOlderFixatedParams(ctx sdk.Context) {
	//TODO:
}
