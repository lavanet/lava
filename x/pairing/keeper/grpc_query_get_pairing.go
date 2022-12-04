package keeper

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/tendermint/tendermint/rpc/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TODO: if I stop the chain, and then start it after a few seconds, the average block time considers the time that passed between activations. Bug?
// Gets a client's provider list in a specific chain. Also returns the start block of the current epoch, time (in seconds) until there's a new pairing, the block that the chain in the request's spec was changed
func (k Keeper) GetPairing(goCtx context.Context, req *types.QueryGetPairingRequest) (*types.QueryGetPairingResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// check client address
	clientAddr, err := sdk.AccAddressFromBech32(req.Client)
	if err != nil {
		return nil, fmt.Errorf("invalid creator address %s error: %s", req.Client, err)
	}

	// Make sure the chain ID exists and the chain's functional
	foundAndActive, _ := k.specKeeper.IsSpecFoundAndActive(ctx, req.ChainID)
	if !foundAndActive {
		return nil, errors.New("spec not found or not enabled")
	}

	// If req.BlockHeight == -1, use current block. Else, use the blockHeight from the request (should be positive if not -1)
	blockHeight := uint64(0)
	if req.BlockHeight == -1 {
		blockHeight = uint64(ctx.BlockHeight())
	} else if req.BlockHeight < 0 && req.BlockHeight != -1 {
		return nil, fmt.Errorf("invalid block height: %d", req.BlockHeight)
	} else {
		blockHeight = uint64(req.BlockHeight)
	}

	// Get pairing in a specific block height
	providers, err := k.GetPairingForClient(ctx, req.ChainID, clientAddr, blockHeight)
	if err != nil {
		return nil, fmt.Errorf("could not get pairing for chainID: %s, client addr: %s, blockHeight: %d, err: %s", req.ChainID, clientAddr, blockHeight, err)
	}

	// Check if the chain in on its first epoch
	chainOnFirstEpoch := false
	if k.epochStorageKeeper.GetEpochStart(ctx) == 0 {
		chainOnFirstEpoch = true
	}

	// Get the past reference for the average time calculation. Should be the start block of the previous epoch or block 0 if the chain's on its first epoch
	pastRef := uint64(0)
	if !chainOnFirstEpoch {
		pastRef, err = k.epochStorageKeeper.GetPreviousEpochStartForBlock(ctx, uint64(ctx.BlockHeight()))
		if err != nil {
			return nil, fmt.Errorf("could not get previous epoch start, err: %s", err)
		}
	}

	// Get the timestamp of the past reference's block
	pastRefInt64 := int64(pastRef)
	pastRefStartBlock, err := core.Block(nil, &pastRefInt64)
	if err != nil {
		return nil, fmt.Errorf("could not get past reference block, err: %s", err)
	}
	pastRefStartBlockTime := pastRefStartBlock.Block.Header.Time.UTC()

	// Get the present reference for the average time calculation. Should be the start block of the current epoch or the current block if the chain's on its first epoch
	presentRef := uint64(ctx.BlockHeight())
	if !chainOnFirstEpoch {
		presentRef = k.epochStorageKeeper.GetEpochStart(ctx)
	}

	// Get the timestamp of the present reference's block
	presentRefInt64 := int64(presentRef)
	presentRefStartBlock, err := core.Block(nil, &presentRefInt64)
	if err != nil {
		return nil, fmt.Errorf("could not get present reference block, err: %s", err)
	}
	presentRefStartBlockTime := presentRefStartBlock.Block.Header.Time.UTC()

	// Get the number of blocks from the past reference to the present reference
	if presentRef < pastRef {
		return nil, fmt.Errorf("previous epoch's start block height is larger than the current epoch's start block height")
	}
	pastRefToCurrentRefBlocksNum := presentRef - pastRef

	// Calculate average block time in seconds (averaged on one epoch or blocks past until current if the chain is on its first epoch)
	averageBlockTime := presentRefStartBlockTime.Sub(pastRefStartBlockTime).Seconds() / float64(pastRefToCurrentRefBlocksNum)

	// Get the next epoch from the present reference
	nextEpochStart, err := k.epochStorageKeeper.GetNextEpoch(ctx, presentRef)
	if err != nil {
		return nil, fmt.Errorf("could not get next epoch start, err: %s", err)
	}

	// Get the defined as overlap blocks
	overlapBlocks := k.EpochBlocksOverlap(ctx)

	// Get number of blocks from the current block to the next epoch
	blocksUntilNewEpoch := nextEpochStart + overlapBlocks - uint64(ctx.BlockHeight())

	// Calculate the time left for the next pairing in seconds (blocks left * avg block time)
	timeLeftToNextPairing := blocksUntilNewEpoch * uint64(averageBlockTime)

	// Get current epoch start block
	currentEpoch := k.epochStorageKeeper.GetEpochStart(ctx)

	// Get the block in which there was the latest change for the current spec
	spec, found := k.specKeeper.GetSpec(ctx, req.GetChainID())
	if !found {
		return nil, errors.New("spec not found or not enabled")
	}
	specLastUpdatedBlock := spec.BlockLastUpdated

	return &types.QueryGetPairingResponse{Providers: providers, CurrentEpoch: currentEpoch, TimeLeftToNextPairing: timeLeftToNextPairing, SpecLastUpdatedBlock: specLastUpdatedBlock}, nil
}
