package chaintracker

import "errors"

var ( // Consumer Side Errors
	InvalidConfigErrorBlocksToSave  = errors.New("blocks to save wasn't defined in config")
	InvalidConfigBlockTime          = errors.New("average block time wasn't defined in config")
	InvalidLatestBlockNumValue      = errors.New("returned latest block num should be greater than 0, but it's not")
	InvalidReturnedHashes           = errors.New("returned requestedHashes key count should be greater than 0, but it's not")
	ErrorFailedToFetchLatestBlock   = errors.New("Failed to fetch latest block from node")
	InvalidRequestedBlocks          = errors.New("provided requested blocks for function do not compose a valid request")
	RequestedBlocksOutOfRange       = errors.New("requested blocks are outside the supported range by the state tracker")
	ErrorFailedToFetchTooEarlyBlock = errors.New("server memory protection triggered, requested block is too early")
	InvalidRequestedSpecificBlock   = errors.New("provided requested specific blocks for function do not compose a stored entry")
)
