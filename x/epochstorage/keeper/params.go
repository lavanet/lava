package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/epochstorage/types"
)

// GetParams get all parameters as types.Params
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	return types.NewParams(
		k.UnstakeHoldBlocks(ctx),
		k.EpochBlocks(ctx),
		k.EpochsToSave(ctx),
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
	k.paramstore.Get(ctx, types.KeyEpochsToSave, &res)
	return
}

// EpochBlocks returns the EpochBlocks param
func (k Keeper) GetEpochBlocks(ctx sdk.Context, block uint64) (res uint64) {
	//TODO: make sure they param wasn't changed from the current value and that block
	k.paramstore.Get(ctx, types.KeyEpochBlocks, &res)
	return
}

// return the next epoch start
func (k Keeper) IsEpochStart(ctx sdk.Context) (res bool) {
	blocksCycle := k.EpochBlocks(ctx)
	currentBlock := uint64(ctx.BlockHeight())
	//current block modulu blocks cycle returns how many block in the current epoch we are, if its 0 we are at epoch start
	return (currentBlock % blocksCycle) == 0
}

func (k Keeper) BlocksToSave(ctx sdk.Context) (res uint64) {
	blocksToSave := k.EpochsToSave(ctx) * k.EpochBlocks(ctx)
	return blocksToSave
}
