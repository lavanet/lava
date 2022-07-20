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
		k.EpochBlocksTmp(ctx),
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
func (k Keeper) EpochBlocks(ctx sdk.Context, block uint64) (res uint64) {
	res = k.GetFixatedParamsForBlock(ctx, block).EpochBlocks
	return
}

// EpochBlocks returns the EpochBlocks param
func (k Keeper) EpochBlocksTmp(ctx sdk.Context) (res uint64) {
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

// return if this block is an epoch start
func (k Keeper) IsEpochStart(ctx sdk.Context) (res bool) {
	currentBlock := uint64(ctx.BlockHeight())
	return k.BlockInEpoch(ctx, currentBlock) == 0
}

func (k Keeper) BlocksToSave(ctx sdk.Context) (res uint64) {
	blocksToSave := k.EpochsToSave(ctx) * k.EpochBlocksTmp(ctx)
	return blocksToSave
}

func (k Keeper) BlockInEpoch(ctx sdk.Context, block uint64) (res uint64) {
	blocksCycle := k.EpochBlocks(ctx, block)

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
	earliestEpochStart := k.GetEarliestEpochStart(ctx)
	if latestParamChange < earliestEpochStart {
		//latest param change is older than memory, so remove it
		k.paramstore.Set(ctx, types.KeyLatestParamChange, uint64(0))
		//clean up older fixated params, they no longer matter
		k.CleanOlderFixatedParams(ctx, 1) //everything after 0 is too old since there wasn't a param change in a while
		return
	}
	// we have a param change, is it in the last epoch?
	if latestParamChange > k.GetPreviousEpochStartForBlock(ctx, block) {
		// this is a recent change so we need to move the current fixation backwards
		k.PushFixatedParams(ctx, block, earliestEpochStart)
	}
}
