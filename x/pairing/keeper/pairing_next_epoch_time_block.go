package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	EPOCH_BLOCK_DIVIDER uint64 = 5 // determines how many blocks from the previous epoch will be included in the average block time calculation
	MIN_SAMPLE_STEP     uint64 = 1 // the minimal sample step when calculating the average block time
)

// Function to calculate how much time (in seconds) is left until the next epoch
func (k Keeper) calculateNextEpochTimeAndBlock(ctx sdk.Context) (uint64, uint64, error) {
	// Get current epoch
	currentEpoch := k.epochStorageKeeper.GetEpochStart(ctx)

	// Calculate the average block time (i.e., how much time it takes to create a new block, in average)
	averageBlockTime := k.downtimeKeeper.GetParams(ctx).DowntimeDuration

	// Get the next epoch
	nextEpochStart, err := k.epochStorageKeeper.GetNextEpoch(ctx, currentEpoch)
	if err != nil {
		return 0, 0, fmt.Errorf("could not get next epoch start, err: %s", err)
	}

	// Get epochBlocksOverlap
	overlapBlocks := k.EpochBlocksOverlap(ctx)

	// calculate the block in which the next pairing will happen (+overlap)
	nextPairingBlock := nextEpochStart + overlapBlocks

	// Get number of blocks from the current block to the next epoch
	blocksUntilNewEpoch := nextPairingBlock - uint64(ctx.BlockHeight())

	// Calculate the time left for the next pairing in seconds (blocks left * avg block time)
	timeLeftToNextEpoch := blocksUntilNewEpoch * uint64(averageBlockTime.Seconds())

	return timeLeftToNextEpoch, nextPairingBlock, nil
}
