package keeper

import (
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/epochstorage/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(
		k.UnstakeHoldBlocksRaw(ctx),
		k.EpochBlocksRaw(ctx),
		k.EpochsToSaveRaw(ctx),
		k.LatestParamChange(ctx),
	)
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

func (k Keeper) UnstakeHoldBlocks(ctx sdk.Context, block uint64) (res uint64) {
	//Unstake Hold Blocks is always used for the latest, but we want to use the fixated
	fixation, _ := k.GetFixatedParamsForBlock(ctx, block) //we ignore the error here because this must succeed, and we are logging
	res = fixation.Parameters.UnstakeHoldBlocks
	return
}

// UnstakeHoldBlocksRaw returns the UnstakeHoldBlocks param
func (k Keeper) UnstakeHoldBlocksRaw(ctx sdk.Context) (res uint64) {
	//Unstake Hold Blocks is always used for the latest, but we want to use the fixated
	k.paramstore.Get(ctx, types.KeyUnstakeHoldBlocks, &res)
	return
}

// EpochBlocks returns the EpochBlocks fixated param
func (k Keeper) EpochBlocks(ctx sdk.Context, block uint64) (res uint64, err error) {
	fixation, err := k.GetFixatedParamsForBlock(ctx, block)
	return fixation.Parameters.EpochBlocks, err //in case of error we still have default fixation object
}

// EpochBlocks returns the EpochBlocks param
func (k Keeper) EpochBlocksRaw(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyEpochBlocks, &res)
	return
}

// EpochsToSave returns the EpochsToSave fixated param
func (k Keeper) EpochsToSave(ctx sdk.Context, block uint64) (res uint64, err error) {
	fixation, err := k.GetFixatedParamsForBlock(ctx, block)
	return fixation.Parameters.EpochsToSave, err
}

// EpochsToSaveRaw returns the EpochsToSave param
func (k Keeper) EpochsToSaveRaw(ctx sdk.Context) (res uint64) {
	//TODO: change to support param change
	k.paramstore.Get(ctx, types.KeyEpochsToSave, &res)
	return
}

// return if this block is an epoch start
func (k Keeper) IsEpochStart(ctx sdk.Context) (res bool) {
	currentBlock := uint64(ctx.BlockHeight())
	blockInEpoch, err := k.BlockInEpoch(ctx, currentBlock)
	if err != nil {
		utils.LavaError(ctx, k.Logger(ctx), "IsEpochStart", map[string]string{"error": err.Error()}, "can't get block in epoch")
		return false
	}
	return blockInEpoch == 0
}

func (k Keeper) BlocksToSave(ctx sdk.Context, block uint64) (res uint64, erro error) {
	epochsToSave, err := k.EpochsToSave(ctx, block)
	epochBlocks, err2 := k.EpochBlocks(ctx, block)
	blocksToSave := epochsToSave * epochBlocks
	if err != nil || err2 != nil {
		erro = fmt.Errorf("BlocksToSave param read errors %s, %s", err.Error(), err2.Error())
	}
	return blocksToSave, erro
}

func (k Keeper) BlockInEpoch(ctx sdk.Context, block uint64) (res uint64, err error) {
	//get epochBlocks directly because we also need an epoch start on the current grid and when fixation was saved is an epoch start
	fixtedParams, err := k.GetFixatedParamsForBlock(ctx, block)
	blocksCycle := fixtedParams.Parameters.EpochBlocks
	epochStartInGrid := fixtedParams.FixationBlock //fixation block is always <= block
	blockRelativeToGrid := block - epochStartInGrid
	return blockRelativeToGrid % blocksCycle, err
}

func (k Keeper) LatestParamChange(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyLatestParamChange, &res)
	return
}

// GetEpochStartForBlock gets a session start supports one param change
func (k Keeper) GetEpochStartForBlock(ctx sdk.Context, block uint64) (epochStart uint64, blockInEpoch uint64, err error) {
	blockInTargetEpoch, err := k.BlockInEpoch(ctx, block)
	targetEpochStart := block - blockInTargetEpoch
	return targetEpochStart, blockInTargetEpoch, err
}

func (k Keeper) GetNextEpoch(ctx sdk.Context, block uint64) (nextEpoch uint64, erro error) {
	epochBlocks, err := k.EpochBlocks(ctx, block)
	epochStart, _, err2 := k.GetEpochStartForBlock(ctx, block)
	nextEpoch = epochStart + epochBlocks
	if err != nil {
		erro = err
	} else if err2 != nil {
		erro = err2
	}
	return nextEpoch, erro
}

func (k Keeper) GetPreviousEpochStartForBlock(ctx sdk.Context, block uint64) (previousEpochStart uint64, erro error) {
	epochStart, _, err := k.GetEpochStartForBlock(ctx, block)
	previousEpochStart, _, err2 := k.GetEpochStartForBlock(ctx, epochStart-1) //we take one block before the target epoch so it belongs to the previous epoch
	if err != nil {
		erro = err
	} else if err2 != nil {
		erro = err2
	}
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
	earliestEpochStart := k.GetEarliestEpochStart(ctx) //this is the previous epoch start, before we update it to the current block
	if latestParamChange < earliestEpochStart {
		//latest param change is older than memory, so remove it
		k.paramstore.Set(ctx, types.KeyLatestParamChange, uint64(0))
		//clean up older fixated params, they no longer matter
		k.CleanOlderFixatedParams(ctx, 1) //everything after 0 is too old since there wasn't a param change in a while
		return
	}
	// we have a param change, is it in the last epoch?
	prevEpochStart, err := k.GetPreviousEpochStartForBlock(ctx, block)
	if err != nil {
		utils.LavaError(ctx, k.Logger(ctx), "GetPreviousEpochStartForBlock_pushFixation", map[string]string{"error": err.Error(), "block": strconv.FormatUint(block, 10)}, "can't get block in epoch")
	} else if latestParamChange > prevEpochStart {
		// this is a recent change so we need to move the current fixation backwards
		k.PushFixatedParams(ctx, block, earliestEpochStart)
	}
}
