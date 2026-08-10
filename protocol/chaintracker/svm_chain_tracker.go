package chaintracker

import (
	"context"
	"encoding/json"
	fmt "fmt"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/lavanet/lava/v5/utils"
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
	slotCache    *ristretto.Cache[int64, int64]  // marks slots the tracker has observed, so hash lookups can tell "not yet polled" from "too old".
	hashCache    *ristretto.Cache[int64, string] // cache for slot to hash.
	seenBlock    int64
	// wireLatestBlock is value.lastValidBlockHeight from the most recent poll: the
	// block height at which the returned blockhash expires. It is NOT the chain
	// position — the tracker uses context.slot for that — and is kept only because it
	// is the number providers have historically published in RelayReply.LatestBlock.
	// Consumers ratchet that value and send it back to every provider, so changing it
	// would put an un-upgraded fleet out of sync with an upgraded one. See
	// GetWireLatestBlock.
	wireLatestBlock int64
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

// GetWireLatestBlock reports the value this tracker publishes to consumers, which
// deliberately stays in the legacy block-height domain while the chain position moves
// to slots. Returns false before the first successful poll, so callers fall back to
// the chain position rather than publishing a zero.
func (cs *SVMChainTracker) GetWireLatestBlock() (int64, bool) {
	wire := atomic.LoadInt64(&cs.wireLatestBlock)
	return wire, wire > 0
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

	// Solana uses slot (not block height) as the canonical chain-position primitive:
	// the spec's GET_BLOCKNUM directive parses context.slot, GET_BLOCK_BY_NUM calls
	// getBlock(slot), and every numeric API argument the spec parses as a requested
	// block is a slot. The tracker's latest value must live in that same domain.
	//
	// value.lastValidBlockHeight is the *block height* at which the returned blockhash
	// expires (~current height + 150). Block height trails slot by the accumulated
	// skipped slots — ~22M on mainnet — so publishing it as the latest block put every
	// downstream comparison in the wrong numeric domain.
	slot := response.Result.Context.Slot
	blockHash := response.Result.Value.BlockHash

	atomic.StoreInt64(&cs.seenBlock, slot)
	cs.slotCache.SetWithTTL(slot, slot, 1, slotCacheTTL)
	cs.hashCache.SetWithTTL(slot, blockHash, 1, hashCacheTTL)
	// Kept only to preserve the published value; nothing internal compares against it.
	atomic.StoreInt64(&cs.wireLatestBlock, response.Result.Value.LastValidBlockHeight)

	utils.LavaFormatTrace("[SVMChainTracker] fetching latest slot",
		utils.LogAttr("slot", slot),
		utils.LogAttr("block_hash", blockHash),
		utils.LogAttr("wire_latest_block", response.Result.Value.LastValidBlockHeight),
	)

	return slot, nil
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

// On Solana the interface's blockNum parameter is a slot.
func (cs *SVMChainTracker) FetchBlockHashByNum(ctx context.Context, slot int64) (string, error) {
	if slot < cs.dataFetcher.GetAtomicLatestBlockNum()-int64(cs.dataFetcher.GetServerBlockMemory()) {
		return "", ErrorFailedToFetchTooEarlyBlock.Wrapf("requested slot: %d, latest slot: %d, server memory %d", slot, cs.dataFetcher.GetAtomicLatestBlockNum(), cs.dataFetcher.GetServerBlockMemory())
	}
	blockHash, ok := cs.hashCache.Get(slot)
	if ok {
		utils.LavaFormatTrace("[SVMChainTracker] FetchBlockHashByNum found hash in cache", utils.LogAttr("slot", slot), utils.LogAttr("hash", blockHash))
		return blockHash, nil
	}

	if err := cs.waitForSlotVisible(slot); err != nil {
		return "", err
	}

	hash, err := cs.chainFetcher.FetchBlockHashByNum(ctx, slot)
	if err == nil {
		utils.LavaFormatTrace("[SVMChainTracker] FetchBlockHashByNum succeeded", utils.LogAttr("slot", slot), utils.LogAttr("hash", hash))
	}
	return hash, err
}

// waitForSlotVisible blocks briefly until the tracker has observed slot at least once.
// Handles the bootstrap race where a hash lookup can arrive before the poll that records the slot.
func (cs *SVMChainTracker) waitForSlotVisible(slot int64) error {
	if slot <= atomic.LoadInt64(&cs.seenBlock) {
		for i := 0; i < getSlotFromCacheMaxRetries; i++ {
			if _, ok := cs.slotCache.Get(slot); ok {
				return nil
			}
			time.Sleep(getSlotFromCacheSleepDuration)
		}
	}

	return fmt.Errorf("slot not yet visible. This can happen on bootstrap and should resolve by itself, if persists please let the dev team know. "+
		"slot: %d, latest_slot: %d, server_memory: %d", slot, cs.dataFetcher.GetAtomicLatestBlockNum(), cs.dataFetcher.GetServerBlockMemory())
}
