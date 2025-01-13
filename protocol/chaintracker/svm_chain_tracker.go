package chaintracker

import (
	"context"
	"encoding/json"
	fmt "fmt"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/lavanet/lava/v4/utils"
)

const (
	CacheMaxCost                  = 100000 // each item cost would be 1
	CacheNumCounters              = 100000 // expect 100000 items
	latestBlockRequest            = "{\"jsonrpc\":\"2.0\",\"method\":\"getLatestBlockhash\",\"params\":[{\"commitment\":\"finalized\"}],\"id\":1}"
	slotCacheTTL                  = time.Hour * 4
	hashCacheTTL                  = time.Hour * 1
	getSlotFromCacheMaxRetries    = 5
	getSlotFromCacheSleepDuration = time.Millisecond * 50
)

type IChainFetcherWrapper interface {
	FetchLatestBlockNum(ctx context.Context) (int64, error)
	FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error)
}

type IChainTrackerDataFetcher interface {
	GetAtomicLatestBlockNum() int64
	GetServerBlockMemory() uint64
}

type SVMChainTracker struct {
	dataFetcher  IChainTrackerDataFetcher
	chainFetcher ChainFetcher
	slotCache    *ristretto.Cache[int64, int64]  // cache for block to slot. (a few slots can point the same block, but we don't really care about that so overwrite is ok)
	hashCache    *ristretto.Cache[int64, string] // cache for block to hash.
	seenBlock    int64
}

type SVMLatestBlockResponse struct {
	Result struct {
		Context struct {
			Slot int64 `json:"slot"`
		} `json:"context"`
		Value struct {
			LastValidBlockHeight int64  `json:"lastValidBlockHeight"`
			BlockHash            string `json:"blockhash"`
		} `json:"value"`
	} `json:"result"`
}

func (cs *SVMChainTracker) fetchLatestBlockNumInner(ctx context.Context) (int64, error) {
	latestBlockResponse, err := cs.chainFetcher.CustomMessage(ctx, "", []byte(latestBlockRequest), "POST", "getLatestBlockhash")
	if err != nil {
		return 0, err
	}

	var response SVMLatestBlockResponse
	if err := json.Unmarshal(latestBlockResponse, &response); err != nil {
		return 0, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	blockNum := response.Result.Value.LastValidBlockHeight
	slot := response.Result.Context.Slot
	blockHash := response.Result.Value.BlockHash

	atomic.StoreInt64(&cs.seenBlock, blockNum)
	cs.slotCache.SetWithTTL(blockNum, slot, 1, slotCacheTTL)
	cs.hashCache.SetWithTTL(blockNum, blockHash, 1, hashCacheTTL)

	utils.LavaFormatTrace("[SVMChainTracker] fetching latest block num",
		utils.LogAttr("slot", slot),
		utils.LogAttr("block_num", blockNum),
		utils.LogAttr("block_hash", blockHash),
	)

	return blockNum, nil
}

func (cs *SVMChainTracker) FetchLatestBlockNum(ctx context.Context) (int64, error) {
	latestBlockNum, err := cs.fetchLatestBlockNumInner(ctx)
	if err != nil {
		return 0, utils.LavaFormatWarning("[SVMChainTracker] failed to get latest block num, getting from chain fetcher", err,
			utils.LogAttr("block_num", latestBlockNum),
			utils.LogAttr("latest_block", cs.dataFetcher.GetAtomicLatestBlockNum()),
			utils.LogAttr("server_memory", cs.dataFetcher.GetServerBlockMemory()))
	}
	utils.LavaFormatTrace("[SVMChainTracker] fetched latest block num", utils.LogAttr("block_num", latestBlockNum))
	return latestBlockNum, nil
}

func (cs *SVMChainTracker) FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error) {
	if blockNum < cs.dataFetcher.GetAtomicLatestBlockNum()-int64(cs.dataFetcher.GetServerBlockMemory()) {
		return "", ErrorFailedToFetchTooEarlyBlock.Wrapf("requested Block: %d, latest block: %d, server memory %d", blockNum, cs.dataFetcher.GetAtomicLatestBlockNum(), cs.dataFetcher.GetServerBlockMemory())
	}
	blockHash, ok := cs.hashCache.Get(blockNum)
	if ok {
		utils.LavaFormatTrace("[SVMChainTracker] FetchBlockHashByNum found block hash in cache", utils.LogAttr("block_num", blockNum), utils.LogAttr("hash", blockHash))
		return blockHash, nil
	}

	// In SVM, the block hash is fetched by slot instead of block.
	// We need to get the slot which is related to this block number.
	slot, err := cs.tryGetSlotFromCache(blockNum)
	if err != nil {
		return "", err
	}

	utils.LavaFormatTrace("[SVMChainTracker] FetchBlockHashByNum found slot in cache", utils.LogAttr("block_num", blockNum), utils.LogAttr("slot", slot))
	hash, err := cs.chainFetcher.FetchBlockHashByNum(ctx, slot)
	if err == nil {
		utils.LavaFormatTrace("[SVMChainTracker] FetchBlockHashByNum succeeded", utils.LogAttr("block_num", blockNum), utils.LogAttr("hash", hash), utils.LogAttr("slot", slot))
	}
	return hash, err
}

func (cs *SVMChainTracker) tryGetSlotFromCache(blockNum int64) (int64, error) {
	if blockNum <= atomic.LoadInt64(&cs.seenBlock) {
		for i := 0; i < getSlotFromCacheMaxRetries; i++ {
			slot, ok := cs.slotCache.Get(blockNum)
			if ok {
				return slot, nil
			}
			time.Sleep(getSlotFromCacheSleepDuration)
		}
	}

	return 0, fmt.Errorf("slot not found in cache. This error can happen on bootstrap and should resolve by itself, if persists please let the dev team know. "+
		"block: %d, latest_block: %d, server_memory: %d", blockNum, cs.dataFetcher.GetAtomicLatestBlockNum(), cs.dataFetcher.GetServerBlockMemory())
}
