package keeper

import (
	"fmt"
	"math"
	"time"

	"github.com/cometbft/cometbft/rpc/core"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
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
	averageBlockTime, err := k.calculateAverageBlockTime(ctx, currentEpoch)
	if err != nil {
		return 0, 0, fmt.Errorf("could not calculate average block time, err: %s", err)
	}

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
	timeLeftToNextEpoch := blocksUntilNewEpoch * averageBlockTime

	return timeLeftToNextEpoch, nextPairingBlock, nil
}

// Function to calculate the average block time (i.e., how much time it takes to create a new block, in average)
func (k Keeper) calculateAverageBlockTime(ctx sdk.Context, epoch uint64) (uint64, error) {
	// Get epochBlocks (the number of blocks in an epoch)
	epochBlocks, err := k.epochStorageKeeper.EpochBlocks(ctx, epoch)
	if err != nil {
		return 0, fmt.Errorf("could not get epochBlocks, err: %s", err)
	}

	// Define sample step. Determines which timestamps will be taken in the average block time calculation.
	//    if epochBlock < EPOCH_BLOCK_DIVIDER -> sampleStep = MIN_SAMPLE_STEP.
	//    else sampleStep will be epochBlocks/EPOCH_BLOCK_DIVIDER
	if MIN_SAMPLE_STEP > epochBlocks {
		return 0, fmt.Errorf("invalid MIN_SAMPLE_STEP value since it's larger than epochBlocks. MIN_SAMPLE_STEP: %v, epochBlocks: %v", MIN_SAMPLE_STEP, epochBlocks)
	}
	sampleStep := MIN_SAMPLE_STEP
	if epochBlocks > EPOCH_BLOCK_DIVIDER {
		sampleStep = epochBlocks / EPOCH_BLOCK_DIVIDER
	}

	// Get a list of the previous epoch's blocks timestamp and height
	prevEpochTimestampAndHeightList, err := k.getPreviousEpochTimestampsByHeight(ctx, epoch, sampleStep)
	if pairingtypes.NoPreviousEpochForAverageBlockTimeCalculationError.Is(err) || pairingtypes.PreviousEpochStartIsBlockZeroError.Is(err) {
		// if the errors indicate that we're on the first epoch / after a fork or previous epoch start is 0, we return averageBlockTime=0 without an error
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("couldn't get prevEpochTimestampAndHeightList. err: %v", err)
	}

	// Calculate the average block time from prevEpochTimestampAndHeightList
	averageBlockTime, err := calculateAverageBlockTimeFromList(prevEpochTimestampAndHeightList, sampleStep)
	if pairingtypes.NotEnoughBlocksToCalculateAverageBlockTimeError.Is(err) || pairingtypes.AverageBlockTimeIsLessOrEqualToZeroError.Is(err) {
		// we shouldn't fail the get-pairing query because the average block time calculation failed (to indicate the fail, we return 0)
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("couldn't calculate average block time. err: %v", err)
	}

	return averageBlockTime, nil
}

type blockHeightAndTime struct {
	blockHeight uint64
	blockTime   time.Time
}

// Function to get a list of the timestamps of the blocks in the previous epoch of the input (so it'll be deterministic)
func (k Keeper) getPreviousEpochTimestampsByHeight(ctx sdk.Context, epoch, sampleStep uint64) ([]blockHeightAndTime, error) {
	// Check for special cases:
	// 1. no previous epoch - we're on the first epoch / after a fork. Since there is no previous epoch to calculate average time on, return an empty slice and no error
	// 2. start of previous epoch is block 0 - we're on the second epoch. To get the block's header using the "core" module, the block height can't be zero (causes panic). In this case, we also return an empty slice and no error
	prevEpoch, err := k.epochStorageKeeper.GetPreviousEpochStartForBlock(ctx, epoch)
	if err != nil {
		return nil, pairingtypes.NoPreviousEpochForAverageBlockTimeCalculationError
	} else if prevEpoch == 0 {
		return nil, pairingtypes.PreviousEpochStartIsBlockZeroError
	}

	// Get previous epoch timestamps, in sampleStep steps
	prevEpochTimestampAndHeightList := []blockHeightAndTime{}
	for block := prevEpoch; block <= epoch; block += sampleStep {
		// Get current block's height and timestamp
		blockInt64 := int64(block)
		blockCore, err := core.Block(nil, &blockInt64)
		if err != nil {
			return nil, fmt.Errorf("could not get current block header, block: %v, err: %s", blockInt64, err)
		}
		currentBlockTimestamp := blockCore.Block.Header.Time.UTC()
		blockHeightAndTimeStruct := blockHeightAndTime{blockHeight: block, blockTime: currentBlockTimestamp}

		// Append the timestamp to the timestamp list
		prevEpochTimestampAndHeightList = append(prevEpochTimestampAndHeightList, blockHeightAndTimeStruct)
	}

	return prevEpochTimestampAndHeightList, nil
}

func calculateAverageBlockTimeFromList(blockHeightAndTimeList []blockHeightAndTime, sampleStep uint64) (uint64, error) {
	if len(blockHeightAndTimeList) <= 1 {
		return 0, utils.LavaFormatError("There isn't enough blockHeight structs in the previous epoch to calculate average block time", pairingtypes.NotEnoughBlocksToCalculateAverageBlockTimeError)
	}

	averageBlockTime := time.Duration(math.MaxInt64)
	for i := 1; i < len(blockHeightAndTimeList); i++ {
		// Calculate the average block time creation over sampleStep blocks
		currentAverageBlockTime := blockHeightAndTimeList[i].blockTime.Sub(blockHeightAndTimeList[i-1].blockTime) / time.Duration(sampleStep)
		if currentAverageBlockTime <= 0 {
			return 0, utils.LavaFormatError("calculated average block time is less than or equal to zero", pairingtypes.AverageBlockTimeIsLessOrEqualToZeroError, []utils.Attribute{{Key: "block", Value: blockHeightAndTimeList[i].blockHeight}, {Key: "block timestamp", Value: blockHeightAndTimeList[i].blockTime}, {Key: "prevBlock", Value: blockHeightAndTimeList[i-1].blockHeight}, {Key: "prevBlock timestamp", Value: blockHeightAndTimeList[i-1].blockTime}}...)
		}
		// save the minimal average block time
		if averageBlockTime > currentAverageBlockTime {
			averageBlockTime = currentAverageBlockTime
		}
	}

	return uint64(averageBlockTime.Seconds()), nil
}
