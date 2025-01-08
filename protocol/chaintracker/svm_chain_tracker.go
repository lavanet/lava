package chaintracker

import (
	"context"
	"encoding/json"
	fmt "fmt"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/lavanet/lava/v4/utils"
)

const (
	CacheMaxCost       = 100000 // each item cost would be 1
	CacheNumCounters   = 100000 // expect 100000 items
	latestBlockRequest = "{\"jsonrpc\":\"2.0\",\"method\":\"getLatestBlockhash\",\"params\":[{\"commitment\":\"finalized\"}],\"id\":1}"
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
	cache        *ristretto.Cache[int64, int64] // cache for block to slot. (a few slots can point the same block, but we don't really care about that so overwrite is ok)
}

type LatestBlockResponse struct {
	Result struct {
		Context struct {
			Slot int64 `json:"slot"`
		} `json:"context"`
		Value struct {
			LastValidBlockHeight int64 `json:"lastValidBlockHeight"`
		} `json:"value"`
	} `json:"result"`
}

func (cs *SVMChainTracker) fetchLatestBlockNumInner(ctx context.Context) (int64, error) {
	latestBlockResponse, err := cs.chainFetcher.CustomMessage(ctx, "", []byte(latestBlockRequest), "POST", "getLatestBlockhash")
	if err != nil {
		return 0, err
	}
	var response LatestBlockResponse
	if err := json.Unmarshal(latestBlockResponse, &response); err != nil {
		return 0, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	blockNum := response.Result.Value.LastValidBlockHeight
	slot := response.Result.Context.Slot
	cs.cache.SetWithTTL(blockNum, slot, 1, time.Hour*24)
	utils.LavaFormatDebug("[SVMChainTracker] fetching latest block num", utils.LogAttr("slot", slot), utils.LogAttr("block_num", blockNum))
	return blockNum, nil
}

func (cs *SVMChainTracker) FetchLatestBlockNum(ctx context.Context) (int64, error) {
	latestBlockNum, err := cs.fetchLatestBlockNumInner(ctx)
	if err != nil {
		utils.LavaFormatWarning("[SVMChainTracker] failed to get latest block num, getting from chain fetcher", err)
		return cs.chainFetcher.FetchLatestBlockNum(ctx)
	}
	utils.LavaFormatDebug("[SVMChainTracker] fetched latest block num", utils.LogAttr("block_num", latestBlockNum))
	return latestBlockNum, nil
}

func (cs *SVMChainTracker) FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error) {
	if blockNum < cs.dataFetcher.GetAtomicLatestBlockNum()-int64(cs.dataFetcher.GetServerBlockMemory()) {
		return "", ErrorFailedToFetchTooEarlyBlock.Wrapf("requested Block: %d, latest block: %d, server memory %d", blockNum, cs.dataFetcher.GetAtomicLatestBlockNum(), cs.dataFetcher.GetServerBlockMemory())
	}
	// In SVM, the block hash is fetched by slot instead of block.
	// We need to get the slot which is related to this block number.
	slot, ok := cs.cache.Get(blockNum)
	if !ok {
		utils.LavaFormatError("slot not found in cache, falling back to direct block fetch - This error can happen on bootstrap and should resolve by itself, if persists please let the dev team know", ErrorFailedToFetchTooEarlyBlock, utils.LogAttr("block", blockNum), utils.LogAttr("latest_block", cs.dataFetcher.GetAtomicLatestBlockNum()), utils.LogAttr("server_memory", cs.dataFetcher.GetServerBlockMemory()))
		return cs.chainFetcher.FetchBlockHashByNum(ctx, blockNum)
	}
	utils.LavaFormatDebug("[SVMChainTracker] FetchBlockHashByNum found slot in cache", utils.LogAttr("block_num", blockNum), utils.LogAttr("slot", slot))
	hash, err := cs.chainFetcher.FetchBlockHashByNum(ctx, slot)
	if err == nil {
		utils.LavaFormatDebug("[SVMChainTracker] FetchBlockHashByNum succeeded", utils.LogAttr("block_num", blockNum), utils.LogAttr("hash", hash), utils.LogAttr("slot", slot))
	}
	return hash, err
}
