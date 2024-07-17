package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/epochstorage/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(
		k.UnstakeHoldBlocksRaw(ctx),
		k.EpochBlocksRaw(ctx),
		k.EpochsToSaveRaw(ctx),
		k.LatestParamChange(ctx),
		k.UnstakeHoldBlocksStaticRaw(ctx),
	)
}

// SetParams set the params
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramstore.SetParamSet(ctx, &params)
}

func (k Keeper) UnstakeHoldBlocks(ctx sdk.Context, block uint64) (res uint64) {
	// Unstake Hold Blocks is always used for the latest, but we want to use the fixated
	k.GetParamForBlock(ctx, string(types.KeyUnstakeHoldBlocks), block, &res)
	return
}

// UnstakeHoldBlocksRaw returns the UnstakeHoldBlocks param
func (k Keeper) UnstakeHoldBlocksRaw(ctx sdk.Context) (res uint64) {
	// Unstake Hold Blocks is always used for the latest, but we want to use the fixated
	k.paramstore.Get(ctx, types.KeyUnstakeHoldBlocks, &res)
	return
}

func (k Keeper) UnstakeHoldBlocksStatic(ctx sdk.Context, block uint64) (res uint64) {
	// Unstake Hold Blocks is always used for the latest, but we want to use the fixated
	k.GetParamForBlock(ctx, string(types.KeyUnstakeHoldBlocksStatic), block, &res)
	return
}

// UnstakeHoldBlocksRaw returns the UnstakeHoldBlocks param
func (k Keeper) UnstakeHoldBlocksStaticRaw(ctx sdk.Context) (res uint64) {
	// Unstake Hold Blocks is always used for the latest, but we want to use the fixated
	k.paramstore.Get(ctx, types.KeyUnstakeHoldBlocksStatic, &res)
	return
}

// UnstakeHoldBlocksRaw sets the UnstakeHoldBlocks param
func (k Keeper) SetUnstakeHoldBlocksStaticRaw(ctx sdk.Context, unstakeHoldBlocksStatic uint64) {
	k.paramstore.Set(ctx, types.KeyUnstakeHoldBlocksStatic, unstakeHoldBlocksStatic)
}

// EpochBlocks returns the EpochBlocks fixated param
func (k Keeper) EpochBlocks(ctx sdk.Context, block uint64) (res uint64, err error) {
	err = k.GetParamForBlock(ctx, string(types.KeyEpochBlocks), block, &res)
	return
}

// EpochBlocks returns the EpochBlocks param
func (k Keeper) EpochBlocksRaw(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyEpochBlocks, &res)
	return
}

// EpochsToSave returns the EpochsToSave fixated param
func (k Keeper) EpochsToSave(ctx sdk.Context, block uint64) (res uint64, err error) {
	err = k.GetParamForBlock(ctx, string(types.KeyEpochsToSave), block, &res)
	return
}

// EpochsToSaveRaw returns the EpochsToSave param
func (k Keeper) EpochsToSaveRaw(ctx sdk.Context) (res uint64) {
	// TODO: change to support param change
	k.paramstore.Get(ctx, types.KeyEpochsToSave, &res)
	return
}

// return if this block is an epoch start
func (k Keeper) IsEpochStart(ctx sdk.Context) (res bool) {
	currentBlock := uint64(ctx.BlockHeight())
	blockInEpoch, err := k.BlockInEpoch(ctx, currentBlock)
	if err != nil {
		utils.LavaFormatError("can't get block in epoch", err)
		return false
	}
	return blockInEpoch == 0
}

func (k Keeper) BlocksToSaveRaw(ctx sdk.Context) (res uint64) {
	return k.EpochsToSaveRaw(ctx) * k.EpochBlocksRaw(ctx)
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
	// get epochBlocks directly because we also need an epoch start on the current grid and when fixation was saved is an epoch start
	fixatedParams, err := k.GetFixatedParamsForBlock(ctx, string(types.KeyEpochBlocks), block)
	if err != nil {
		return 0, err
	}

	var epochBlocks uint64
	utils.Deserialize(fixatedParams.Parameter, &epochBlocks)

	if epochBlocks == 0 {
		return 0, utils.LavaFormatError("blocksCycle is 0", fmt.Errorf("critical: Attempt to divide by zero"),
			utils.LogAttr("fixatedParams", fixatedParams),
		)
	}

	epochStartInGrid := fixatedParams.FixationBlock // fixation block is always <= block
	blockRelativeToGrid := block - epochStartInGrid
	return blockRelativeToGrid % epochBlocks, err
}

func (k Keeper) LatestParamChange(ctx sdk.Context) (res uint64) {
	k.paramstore.Get(ctx, types.KeyLatestParamChange, &res)
	return
}

// GetEpochStartForBlock gets a session start supports one param change
func (k Keeper) GetEpochStartForBlock(ctx sdk.Context, block uint64) (epochStart, blockInEpoch uint64, err error) {
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

func (k Keeper) GetCurrentNextEpoch(ctx sdk.Context) (nextEpoch uint64) {
	epochBlocks := k.EpochBlocksRaw(ctx)
	details, found := k.GetEpochDetails(ctx)
	if details.EarliestStart == details.StartBlock {
		nextEpoch = details.StartBlock + epochBlocks
		if !found {
			utils.LavaFormatPanic("blabla", nil)
		}
	} else {
		var err error
		nextEpoch, err = k.GetNextEpoch(ctx, uint64(ctx.BlockHeight()))
		if err != nil {
			utils.LavaFormatPanic("blabla", nil)
		}
	}

	return nextEpoch
}

func (k Keeper) GetPreviousEpochStartForBlock(ctx sdk.Context, block uint64) (previousEpochStart uint64, erro error) {
	epochStart, _, err := k.GetEpochStartForBlock(ctx, block)
	if epochStart <= 0 {
		return 0, fmt.Errorf("GetPreviousEpochStartForBlock tried to fetch epoch beyond zero")
	}
	previousEpochStart, _, err2 := k.GetEpochStartForBlock(ctx, epochStart-1) // we take one block before the target epoch so it belongs to the previous epoch
	if err != nil {
		erro = err
	} else if err2 != nil {
		erro = err2
	}
	return
}
