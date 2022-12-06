package keeper

import (
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/epochstorage/types"
	"github.com/tendermint/tendermint/rpc/core"
)

// SetEpochDetails set epochDetails in the store
func (k Keeper) SetEpochDetails(ctx sdk.Context, epochDetails types.EpochDetails) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.EpochDetailsKey))
	b := k.cdc.MustMarshal(&epochDetails)
	store.Set([]byte{0}, b)
}

// GetEpochDetails returns epochDetails
func (k Keeper) GetEpochDetails(ctx sdk.Context) (val types.EpochDetails, found bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.EpochDetailsKey))

	b := store.Get([]byte{0})
	if b == nil {
		return val, false
	}

	k.cdc.MustUnmarshal(b, &val)
	return val, true
}

// RemoveEpochDetails removes epochDetails from the store
func (k Keeper) RemoveEpochDetails(ctx sdk.Context) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefix(types.EpochDetailsKey))
	store.Delete([]byte{0})
}
func (k Keeper) SetEpochDetailsStart(ctx sdk.Context, block uint64) {
	details, found := k.GetEpochDetails(ctx)
	if !found {
		panic("did not find EpochDetails")
	}
	details.StartBlock = block
	k.SetEpochDetails(ctx, details)
}

func (k Keeper) GetEpochStart(ctx sdk.Context) uint64 {
	details, found := k.GetEpochDetails(ctx)
	if !found {
		panic("did not find EpochDetails")
	}
	return details.StartBlock
}

func (k Keeper) GetEarliestEpochStart(ctx sdk.Context) uint64 {
	details, found := k.GetEpochDetails(ctx)
	if !found {
		panic("did not find EpochDetails")
	}
	return details.EarliestStart
}

func (k Keeper) GetDeletedEpochs(ctx sdk.Context) []uint64 {
	details, found := k.GetEpochDetails(ctx)
	if !found {
		panic("did not find EpochDetails")
	}
	return details.DeletedEpochs
}

func (k Keeper) SetEarliestEpochStart(ctx sdk.Context, block uint64, deletedEpochs []uint64) {
	details, found := k.GetEpochDetails(ctx)
	if !found {
		panic("did not find EpochDetails")
	}
	details.DeletedEpochs = deletedEpochs
	details.EarliestStart = block
	k.SetEpochDetails(ctx, details)
}

func (k Keeper) GetAverageBlockTime(ctx sdk.Context) uint64 {
	details, found := k.GetEpochDetails(ctx)
	if !found {
		panic("did not find EpochDetails")
	}
	return details.AverageBlockTime
}

func (k Keeper) CalculateAndSetAverageBlockTime(ctx sdk.Context) error {

	start := time.Now()

	details, found := k.GetEpochDetails(ctx)
	if !found {
		panic("did not find EpochDetails")
	}

	// Check if the chain in on its first epoch
	chainOnFirstEpoch := false
	if k.GetEpochStart(ctx) == 0 {
		chainOnFirstEpoch = true
	}

	// Get the past reference for the block time calculation. Should be the start block of the previous epoch or block 0 if the chain's on its first epoch
	var pastRef uint64
	var err error
	if !chainOnFirstEpoch {
		pastRef, err = k.GetPreviousEpochStartForBlock(ctx, uint64(ctx.BlockHeight()))
		if err != nil {
			return fmt.Errorf("could not get previous epoch start, err: %s", err)
		}
	} else {
		pastRef = uint64(0)
	}

	// Get the present reference for the block time calculation. Should be the start block of the current epoch or the current block if the chain's on its first epoch
	presentRef := uint64(ctx.BlockHeight())
	if !chainOnFirstEpoch {
		presentRef = k.GetEpochStart(ctx)
	}

	// Get the number of blocks from the past reference to the present reference
	if presentRef < pastRef {
		return fmt.Errorf("previous reference start block height is larger than the present reference start block height")
	}
	pastRefToPresentRefBlocksNum := int64(presentRef - pastRef)

	// Define sample step. Determines which timestamps (step = 20% of the number of blocks between past and present ref)
	sampleStep := int64(float64(pastRefToPresentRefBlocksNum) * 0.2)

	// Get the timestamps of all the blocks between the pastRef and presentRef. Then, calculate the differences between them.
	blockTimesList := make([]float64, 50)
	counter := 0
	for block := int64(pastRef); block < int64(presentRef); block = block + sampleStep {
		counter++
		// Get current block timestamp
		currentBlock, err := core.Block(nil, &block)
		if err != nil {
			return fmt.Errorf("could not get past reference block, err: %s", err)
		}
		currentBlockTimestamp := currentBlock.Block.Header.Time.UTC()

		// Get next block timestamp
		nextBlockIndex := block + 1
		nextBlock, err := core.Block(nil, &nextBlockIndex)
		if err != nil {
			return fmt.Errorf("could not get past reference block, err: %s", err)
		}
		nextBlockTimestamp := nextBlock.Block.Header.Time.UTC()

		// Calculte difference and append to list
		timeDifference := nextBlockTimestamp.Sub(currentBlockTimestamp).Seconds()
		// timeDifference := float64(1)
		blockTimesList[block-int64(pastRef)] = timeDifference
	}

	// Calculate average block time in seconds (averaged on one epoch or blocks past until current if the chain is on its first epoch)
	minBlockTime := blockTimesList[0]
	for _, blockTime := range blockTimesList {
		if blockTime < minBlockTime {
			minBlockTime = blockTime
		}
	}
	blockCreationTime := minBlockTime
	if blockCreationTime < 0 {
		return fmt.Errorf("block creation time is less then zero")
	}

	details.AverageBlockTime = uint64(blockCreationTime)
	k.SetEpochDetails(ctx, details)

	elapsed := time.Since(start)
	fmt.Printf("YOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOOO\n")
	fmt.Printf("elapsed %v\n", elapsed)
	fmt.Printf("blockCreationTime: %v\n", blockCreationTime)

	return nil
}
