package chaintracker

import (
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var ( // Consumer Side Errors
	InvalidConfigErrorBlocksToSave = sdkerrors.New("Invalid blocks to save", 1, "blocks to save wasn't defined in config")
	InvalidConfigBlockTime         = sdkerrors.New("Invalid average block time", 2, "average block time wasn't defined in config")
	InvalidLatestBlockNumValue     = sdkerrors.New("Invalid value for latestBlockNum", 3, "returned latest block num should be greater than 0, but it's not")
	InvalidReturnedHashes          = sdkerrors.New("Invalid value for requestedHashes length", 4, "returned requestedHashes key count should be greater than 0, but it's not")
	ErrorFailedToFetchLatestBlock  = sdkerrors.New("Error FailedToFetchLatestBlock", 5, "Failed to fetch latest block from node")
)
