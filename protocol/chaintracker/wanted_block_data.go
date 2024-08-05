package chaintracker

import (
	fmt "fmt"

	spectypes "github.com/lavanet/lava/v2/x/spec/types"
)

type BlockRange struct {
	fromBlock              int64
	toBlock                int64
	startIndexFromEarliest uint64
	endIndexFromEarliest   uint64 // not inclusive
}

type WantedBlocksData struct {
	rangeBlocks   *BlockRange
	specificBlock *BlockRange
}

func (wbd *WantedBlocksData) New(fromBlock, toBlock, specificBlock, latestBlock, earliestBlockSaved int64) (err error) {
	if earliestBlockSaved > latestBlock {
		return InvalidLatestBlockNumValue
	}
	ignoreRange := fromBlock == spectypes.NOT_APPLICABLE || toBlock == spectypes.NOT_APPLICABLE
	ignoreSpecific := specificBlock == spectypes.NOT_APPLICABLE
	if ignoreRange && ignoreSpecific {
		return InvalidRequestedBlocks
	}
	// handle spectypes.LATEST_BLOCK - distance requests
	if !ignoreRange {
		fromBlock = LatestArgToBlockNum(fromBlock, latestBlock)
		toBlock = LatestArgToBlockNum(toBlock, latestBlock)
		wbd.rangeBlocks, err = NewBlockRange(fromBlock, toBlock, earliestBlockSaved, latestBlock)
		if err != nil {
			return err
		}
	} else {
		// for clarity
		wbd.rangeBlocks = nil
	}
	if !ignoreSpecific {
		fromBlockArg := LatestArgToBlockNum(specificBlock, latestBlock)
		if wbd.rangeBlocks.IsWanted(fromBlockArg) {
			// this means the specific block is within the range of [from-to] and there is no reason to create specific block range
			wbd.specificBlock = nil
		} else {
			toBlockArg := fromBlockArg // [from,to] with only one block
			wbd.specificBlock, err = NewBlockRange(fromBlockArg, toBlockArg, earliestBlockSaved, latestBlock)
			if err != nil {
				return InvalidRequestedSpecificBlock.Wrapf("specific " + err.Error())
			}
		}
	} else {
		// for clarity
		wbd.specificBlock = nil
	}
	return nil
}

func (wbd *WantedBlocksData) IterationIndexes() (returnedIdxs []int) {
	if wbd.rangeBlocks != nil {
		if wbd.specificBlock != nil {
			if wbd.rangeBlocks.startIndexFromEarliest < wbd.specificBlock.startIndexFromEarliest {
				returnedIdxs = wbd.rangeBlocks.IterationIndexes()
				returnedIdxs = append(returnedIdxs, wbd.specificBlock.IterationIndexes()...)
				return
			}
			returnedIdxs = wbd.specificBlock.IterationIndexes()
			returnedIdxs = append(returnedIdxs, wbd.rangeBlocks.IterationIndexes()...)
			return
		}
		returnedIdxs = wbd.rangeBlocks.IterationIndexes()
		return
	}
	if wbd.specificBlock != nil {
		return wbd.specificBlock.IterationIndexes()
	}
	// empty iteration indexes list
	return
}

func (wbd *WantedBlocksData) IsWanted(blockNum int64) bool {
	// expecting only positive blockNums here
	if blockNum < 0 {
		return false
	}
	// if is the specific block return true
	if wbd.specificBlock.IsWanted(blockNum) {
		return true
	}
	// if is in range [from,to) return true
	if wbd.rangeBlocks.IsWanted(blockNum) {
		return true
	}
	return false
}

func (wbd *WantedBlocksData) String() string {
	return fmt.Sprintf("range: %+v specific: %+v", wbd.rangeBlocks, wbd.specificBlock)
}

func NewBlockRange(fromBlock, toBlock, earliestBlockSaved, latestBlock int64) (br *BlockRange, err error) {
	if fromBlock < 0 || toBlock < 0 || earliestBlockSaved < 0 {
		return nil, RequestedBlocksOutOfRange.Wrapf("invalid input block range: from=%d to=%d earliest=%d latest=%d", fromBlock, toBlock, earliestBlockSaved, latestBlock)
	}
	if toBlock < fromBlock { // if we don't have a range, it should be set with NOT_APPLICABLE
		return nil, InvalidRequestedBlocks.Wrapf("invalid input block range: from=%d to=%d earliest=%d latest=%d", fromBlock, toBlock, earliestBlockSaved, latestBlock)
	}
	if fromBlock < earliestBlockSaved {
		return nil, RequestedBlocksOutOfRange.Wrapf("invalid input block fromBlock: from=%d to=%d earliest=%d latest=%d", fromBlock, toBlock, earliestBlockSaved, latestBlock)
	}
	if toBlock > latestBlock {
		return nil, RequestedBlocksOutOfRange.Wrapf("invalid input block toBlock: from=%d to=%d latest=%d", fromBlock, toBlock, latestBlock)
	}
	blockRange := &BlockRange{}
	blockRange.fromBlock = fromBlock
	blockRange.toBlock = toBlock
	blockRange.startIndexFromEarliest = uint64(blockRange.fromBlock - earliestBlockSaved)
	blockRange.endIndexFromEarliest = uint64(blockRange.toBlock - earliestBlockSaved)
	return blockRange, nil
}

func (br *BlockRange) IterationIndexes() []int {
	indexes := make([]int, br.endIndexFromEarliest-br.startIndexFromEarliest+1)
	for i := 0; i < len(indexes); i++ {
		indexes[i] = int(br.startIndexFromEarliest) + i
	}
	return indexes
}

func (br *BlockRange) IsWanted(blockNum int64) bool {
	if br == nil {
		return false
	}
	if br.fromBlock > blockNum {
		return false
	}
	if br.toBlock < blockNum {
		return false
	}
	return true
}

func LatestArgToBlockNum(request, latestBlock int64) int64 {
	// only modify latest - X requests
	if request > spectypes.LATEST_BLOCK {
		return request
	}
	// we have: spectypes.LATEST_BLOCK - X = request, we need: latest - X
	// request - spectypes.LATEST_BLOCK + latest = spectypes.LATEST_BLOCK - X - spectypes.LATEST_BLOCK + latest = latest - X
	res := request - spectypes.LATEST_BLOCK + latestBlock
	if res < 0 {
		// edge case where X is bigger than latest
		return latestBlock
	}
	return res
}
