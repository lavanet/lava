package chaintracker

import (
	sdkerrors "cosmossdk.io/errors"
)

var ( // Consumer Side Errors
	InvalidConfigErrorBlocksToSave  = sdkerrors.New("Invalid blocks to save", 10701, "blocks to save wasn't defined in config")
	InvalidConfigBlockTime          = sdkerrors.New("Invalid average block time", 10702, "average block time wasn't defined in config")
	InvalidLatestBlockNumValue      = sdkerrors.New("Invalid value for latestBlockNum", 10703, "returned latest block num should be greater than 0, but it's not")
	InvalidReturnedHashes           = sdkerrors.New("Invalid value for requestedHashes length", 10704, "returned requestedHashes key count should be greater than 0, but it's not")
	ErrorFailedToFetchLatestBlock   = sdkerrors.New("Error FailedToFetchLatestBlock", 10705, "Failed to fetch latest block from node")
	InvalidRequestedBlocks          = sdkerrors.New("Error InvalidRequestedBlocks", 10706, "provided requested blocks for function do not compose a valid request")
	RequestedBlocksOutOfRange       = sdkerrors.New("RequestedBlocksOutOfRange", 10707, "requested blocks are outside the supported range by the state tracker")
	ErrorFailedToFetchTooEarlyBlock = sdkerrors.New("Error ErrorFailedToFetchTooEarlyBlock", 10708, "server memory protection triggered, requested block is too early")
	InvalidRequestedSpecificBlock   = sdkerrors.New("Error InvalidRequestedSpecificBlock", 10709, "provided requested specific blocks for function do not compose a stored entry")
)
