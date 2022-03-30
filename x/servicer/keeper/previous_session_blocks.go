package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
)

// SetPreviousSessionBlocks set previousSessionBlocks in the store
func (k Keeper) SetPreviousSessionBlocks(ctx sdk.Context, previousSessionBlocks types.PreviousSessionBlocks) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PreviousSessionBlocksKey))
	b := k.cdc.MustMarshal(&previousSessionBlocks)
	store.Set([]byte{0}, b)
}

// GetPreviousSessionBlocks returns previousSessionBlocks
func (k Keeper) GetPreviousSessionBlocks(ctx sdk.Context) (val types.PreviousSessionBlocks, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PreviousSessionBlocksKey))

	b := store.Get([]byte{0})
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemovePreviousSessionBlocks removes previousSessionBlocks from the store
func (k Keeper) RemovePreviousSessionBlocks(ctx sdk.Context) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.PreviousSessionBlocksKey))
	store.Delete([]byte{0})
}

func (k Keeper) GetSessionBlocksAndOverlapForBlock(ctx sdk.Context, block types.BlockNum) (uint64, uint64) {
	previousSessionsBlocks, found := k.GetPreviousSessionBlocks(ctx)
	if !found {
		panic("did not find previousSessionBlocks")
	}
	blockCycleToUse := k.userKeeper.SessionBlocks(ctx)
	overlapBlocks := k.SessionBlocksOverlap(ctx)
	if previousSessionsBlocks.ChangeBlock.Num > block.Num {
		blockCycleToUse = previousSessionsBlocks.BlocksNum
		overlapBlocks = previousSessionsBlocks.OverlapBlocks
	}
	if blockCycleToUse == 0 {
		panic(fmt.Sprintf("blockCycleToUse is 0: previous session block: %d, block num:%d", previousSessionsBlocks.ChangeBlock.Num, block.Num))
	}
	return blockCycleToUse, overlapBlocks
}

// GetSessionStartForBlock gets a session start supports one param change
func (k Keeper) GetSessionStartForBlock(ctx sdk.Context, block types.BlockNum) (targetSessionStart *types.BlockNum, overlappingPreviousSession *types.BlockNum, err error) {
	blockCycleToUse, overlapBlocks := k.GetSessionBlocksAndOverlapForBlock(ctx, block)
	blocksInTargetSession := block.Num % blockCycleToUse
	targetBlockStart := block.Num - blocksInTargetSession
	overlappingPreviousSession = nil
	if overlapBlocks >= blocksInTargetSession && targetBlockStart > blockCycleToUse {
		overlappingPreviousSession = &types.BlockNum{Num: targetBlockStart - blockCycleToUse}
	}
	targetSessionStart = &types.BlockNum{Num: targetBlockStart}
	return targetSessionStart, overlappingPreviousSession, err
}

func (k Keeper) HandleStoringPreviousSessionData(ctx sdk.Context) {
	currentBlock := uint64(ctx.BlockHeight())
	currentSessionStart, found := k.GetCurrentSessionStart(ctx)
	sessionBlocks := k.userKeeper.SessionBlocks(ctx)
	if !found {
		panic("fail due to faulty GetCurrentSessionStart in keeper")
	}

	previousSessionBlocks, found := k.GetPreviousSessionBlocks(ctx)
	//update with current data now, ebcause we dont know when it will change, and i didn't want to hook the param change
	//TODO: hook the param change instead and write to this struct only when it changes
	if !found {
		panic("fail due to faulty GetPreviousSessionBlocks in keeper")
	}
	if previousSessionBlocks.BlocksNum != k.userKeeper.SessionBlocks(ctx) && previousSessionBlocks.ChangeBlock.Num+k.userKeeper.BlocksToSave(ctx) < currentBlock {
		//meaning there was enough time since the last change, and we didn't store the new value yet, so we save the new value
		previousSessionBlocks.BlocksNum = k.userKeeper.SessionBlocks(ctx)
		previousSessionBlocks.OverlapBlocks = k.SessionBlocksOverlap(ctx)
		previousSessionBlocks.ChangeBlock = types.BlockNum{Num: currentBlock}
		k.SetPreviousSessionBlocks(ctx, previousSessionBlocks)
	}
	//1.
	//this check is before we updated currentSession start to be the current block, and itsd a new sessionBlock
	// the difference between them should be sessionBlocks unless it was changed
	if currentSessionStart.Block.Num+sessionBlocks != currentBlock && currentBlock > 0 {
		//we updated sessionBlocks

		//we save the change as the current block (that is a session start), comparisons are always for strong inequality,
		// so all blocks that are smaller
		//will use the session start with the previous params
		previousSessionBlocks.ChangeBlock = types.BlockNum{Num: currentBlock}
		k.SetPreviousSessionBlocks(ctx, previousSessionBlocks)
	}
}
