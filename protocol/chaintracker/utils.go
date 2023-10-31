package chaintracker

import "time"

func exponentialBackoff(baseTime time.Duration, fails uint64) time.Duration {
	if fails > maxFails {
		fails = maxFails
	}
	maxIncrease := BACKOFF_MAX_TIME
	backoff := baseTime * (1 << fails)
	if backoff > maxIncrease {
		backoff = maxIncrease
	}
	return backoff
}

func FindRequestedBlockHash(requestedHashes []*BlockStore, requestBlock, toBlock, fromBlock int64, finalizedBlockHashes map[int64]interface{}) (requestedBlockHash []byte, finalizedBlockHashesMapRet map[int64]interface{}) {
	for _, block := range requestedHashes {
		if block.Block == requestBlock {
			requestedBlockHash = []byte(block.Hash)
			if int64(len(requestedHashes)) == (toBlock - fromBlock + 1) {
				finalizedBlockHashes[block.Block] = block.Hash
			}
		} else {
			finalizedBlockHashes[block.Block] = block.Hash
		}
	}
	return requestedBlockHash, finalizedBlockHashes
}

func BuildProofFromBlocks(requestedHashes []*BlockStore) map[int64]interface{} {
	finalizedBlockHashes := map[int64]interface{}{}
	for _, block := range requestedHashes {
		finalizedBlockHashes[block.Block] = block.Hash
	}
	return finalizedBlockHashes
}
