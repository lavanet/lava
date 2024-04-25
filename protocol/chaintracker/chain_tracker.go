package chaintracker

import (
	"context"
	"errors"
	fmt "fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	rand "github.com/lavanet/lava/utils/rand"

	sdkerrors "cosmossdk.io/errors"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/lavanet/lava/protocol/common"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/metrics"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/lavaslices"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	grpc "google.golang.org/grpc"
)

const (
	initRetriesCount              = 4
	BACKOFF_MAX_TIME              = 10 * time.Minute
	maxFails                      = 10
	GoodStabilityThreshold        = 0.3
	PollingUpdateLength           = 10
	MostFrequentPollingMultiplier = 16
	PollingMultiplierFlagName     = "polling-multiplier"
)

var PollingMultiplier = uint64(1)

type ChainFetcher interface {
	FetchLatestBlockNum(ctx context.Context) (int64, error)
	FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error)
	FetchEndpoint() lavasession.RPCProviderEndpoint
}

type blockTimeUpdatable interface {
	UpdateBlockTime(time.Duration)
}

type ChainTracker struct {
	chainFetcher            ChainFetcher // used to communicate with the node
	blocksToSave            uint64       // how many finalized blocks to keep
	latestBlockNum          int64
	blockQueueMu            sync.RWMutex
	blocksQueue             []BlockStore                    // holds all past hashes up until latest block
	forkCallback            func(int64)                     // a function to be called when a fork is detected
	newLatestCallback       func(int64, int64, string)      // a function to be called when a new block is detected, from what block to what block including gaps
	oldBlockCallback        func(latestBlockTime time.Time) // a function to be called when an old block is detected
	consistencyCallback     func(oldBlock int64, block int64)
	serverBlockMemory       uint64
	endpoint                lavasession.RPCProviderEndpoint
	blockCheckpointDistance uint64 // used to do something every X blocks
	blockCheckpoint         uint64 // last time checkpoint was met
	timer                   *time.Timer
	latestChangeTime        time.Time
	startupTime             time.Time
	blockEventsGap          []time.Duration
	blockTimeUpdatables     map[blockTimeUpdatable]struct{}
	pmetrics                *metrics.ProviderMetricsManager
}

// this function returns block hashes of the blocks: [from block - to block] inclusive. an additional specific block hash can be provided. order is sorted ascending
// it supports requests for [spectypes.LATEST_BLOCK-distance1, spectypes.LATEST_BLOCK-distance2)
// spectypes.NOT_APPLICABLE in fromBlock or toBlock results in only returning specific block.
// if specific block is spectypes.NOT_APPLICABLE it is ignored
func (cs *ChainTracker) GetLatestBlockData(fromBlock, toBlock, specificBlock int64) (latestBlock int64, requestedHashes []*BlockStore, changeTime time.Time, err error) {
	cs.blockQueueMu.RLock()
	defer cs.blockQueueMu.RUnlock()

	latestBlock = cs.GetAtomicLatestBlockNum()
	if len(cs.blocksQueue) == 0 {
		return latestBlock, nil, time.Time{}, utils.LavaFormatError("ChainTracker GetLatestBlockData had no blocks", nil, utils.Attribute{Key: "latestBlock", Value: latestBlock})
	}
	earliestBlockSaved := cs.getEarliestBlockUnsafe().Block
	wantedBlocksData := WantedBlocksData{}
	err = wantedBlocksData.New(fromBlock, toBlock, specificBlock, latestBlock, earliestBlockSaved)
	if err != nil {
		return latestBlock, nil, time.Time{}, sdkerrors.Wrap(err, fmt.Sprintf("invalid input for GetLatestBlockData %v", &map[string]string{
			"fromBlock": strconv.FormatInt(fromBlock, 10), "toBlock": strconv.FormatInt(toBlock, 10), "specificBlock": strconv.FormatInt(specificBlock, 10),
			"latestBlock": strconv.FormatInt(latestBlock, 10), "earliestBlockSaved": strconv.FormatInt(earliestBlockSaved, 10),
		}))
	}

	for _, blocksQueueIdx := range wantedBlocksData.IterationIndexes() {
		blockStore := cs.blocksQueue[blocksQueueIdx]
		if !wantedBlocksData.IsWanted(blockStore.Block) {
			return latestBlock, nil, time.Time{}, utils.LavaFormatError("invalid wantedBlocksData Iteration", err, utils.Attribute{Key: "blocksQueueIdx", Value: blocksQueueIdx}, utils.Attribute{Key: "blockStore", Value: blockStore},
				utils.Attribute{Key: "wantedBlocksData", Value: wantedBlocksData})
		}
		requestedHashes = append(requestedHashes, &blockStore)
	}
	changeTime = cs.latestChangeTime
	return
}

func (cs *ChainTracker) RegisterForBlockTimeUpdates(updatable blockTimeUpdatable) {
	cs.blockQueueMu.Lock()
	defer cs.blockQueueMu.Unlock()
	cs.blockTimeUpdatables[updatable] = struct{}{}
}

func (cs *ChainTracker) updateAverageBlockTimeForRegistrations(averageBlockTime time.Duration) {
	cs.blockQueueMu.RLock()
	defer cs.blockQueueMu.RUnlock()
	for updatable := range cs.blockTimeUpdatables {
		updatable.UpdateBlockTime(averageBlockTime)
	}
}

// blockQueueMu must be locked
func (cs *ChainTracker) getEarliestBlockUnsafe() BlockStore {
	return cs.blocksQueue[0]
}

// blockQueueMu must be locked
func (cs *ChainTracker) getLatestBlockUnsafe() BlockStore {
	if len(cs.blocksQueue) == 0 {
		return BlockStore{Hash: "BAD-HASH"}
	}
	return cs.blocksQueue[len(cs.blocksQueue)-1]
}

func (cs *ChainTracker) GetLatestBlockNum() (int64, time.Time) {
	cs.blockQueueMu.RLock()
	defer cs.blockQueueMu.RUnlock()
	return atomic.LoadInt64(&cs.latestBlockNum), cs.latestChangeTime
}

func (cs *ChainTracker) GetAtomicLatestBlockNum() int64 {
	return atomic.LoadInt64(&cs.latestBlockNum)
}

func (cs *ChainTracker) setLatestBlockNum(value int64) {
	atomic.StoreInt64(&cs.latestBlockNum, value)
}

func (cs *ChainTracker) fetchLatestBlockNum(ctx context.Context) (int64, error) {
	return cs.chainFetcher.FetchLatestBlockNum(ctx)
}

func (cs *ChainTracker) fetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error) {
	if blockNum < cs.GetAtomicLatestBlockNum()-int64(cs.serverBlockMemory) {
		return "", ErrorFailedToFetchTooEarlyBlock.Wrapf("requested Block: %d, latest block: %d, server memory %d", blockNum, cs.GetAtomicLatestBlockNum(), cs.serverBlockMemory)
	}
	return cs.chainFetcher.FetchBlockHashByNum(ctx, blockNum)
}

// this function fetches all previous blocks from the node starting at the latest provided going backwards blocksToSave blocks
// if it reaches a hash that it already has it stops reading
func (cs *ChainTracker) fetchAllPreviousBlocks(ctx context.Context, latestBlock int64) (hashLatest string, err error) {
	newBlocksQueue := make([]BlockStore, int64(cs.blocksToSave))
	currentLatestBlock := cs.GetAtomicLatestBlockNum()
	if latestBlock < currentLatestBlock {
		return "", utils.LavaFormatError("invalid latestBlock provided to fetch, it is older than the current state latest block", err, utils.Attribute{Key: "latestBlock", Value: latestBlock}, utils.Attribute{Key: "currentLatestBlock", Value: currentLatestBlock})
	}
	readIndexDiff := latestBlock - currentLatestBlock
	blocksQueueStartIndex, blocksQueueEndIndex, newQueueStartIndex := int64(0), int64(0), int64(0)
	blocksQueueStartIndex, blocksQueueEndIndex, newQueueStartIndex, err = cs.readHashes(latestBlock, ctx, blocksQueueStartIndex, blocksQueueEndIndex, newQueueStartIndex, readIndexDiff, newBlocksQueue)
	if err != nil {
		return "", err
	}
	blocksCopied := int64(cs.blocksToSave)
	blocksCopied, blocksQueueLen, latestHash := cs.replaceBlocksQueue(latestBlock, newQueueStartIndex, blocksQueueStartIndex, blocksQueueEndIndex, newBlocksQueue, blocksCopied)
	if blocksQueueLen < cs.blocksToSave {
		return "", utils.LavaFormatError("fetchAllPreviousBlocks didn't save enough blocks in Chain Tracker", nil, utils.Attribute{Key: "blocksQueueLen", Value: blocksQueueLen})
	}
	// only print logs if there is something interesting or we reached the checkpoint
	if readIndexDiff > 1 || cs.blockCheckpoint+cs.blockCheckpointDistance < uint64(latestBlock) {
		cs.blockCheckpoint = uint64(latestBlock)
		utils.LavaFormatDebug("Chain Tracker Updated block hashes", utils.Attribute{Key: "latest_block", Value: latestBlock}, utils.Attribute{Key: "latestHash", Value: latestHash}, utils.Attribute{Key: "blocksQueueLen", Value: blocksQueueLen}, utils.Attribute{Key: "blocksQueried", Value: int64(cs.blocksToSave) - blocksCopied}, utils.Attribute{Key: "blocksKept", Value: blocksCopied}, utils.Attribute{Key: "ChainID", Value: cs.endpoint.ChainID}, utils.Attribute{Key: "ApiInterface", Value: cs.endpoint.ApiInterface}, utils.Attribute{Key: "nextBlocksUpdate", Value: cs.blockCheckpoint + cs.blockCheckpointDistance})
	}
	return latestHash, nil
}

func (cs *ChainTracker) replaceBlocksQueue(latestBlock, newQueueStartIndex, blocksQueueStartIndex, blocksQueueEndIndex int64, newBlocksQueue []BlockStore, blocksCopied int64) (int64, uint64, string) {
	cs.blockQueueMu.Lock()
	defer cs.blockQueueMu.Unlock()
	cs.setLatestBlockNum(latestBlock)
	if newQueueStartIndex > 0 {
		// means we copy previous blocks
		cs.blocksQueue = append(cs.blocksQueue[blocksQueueStartIndex:blocksQueueEndIndex], newBlocksQueue[newQueueStartIndex:]...)
		blocksCopied = blocksQueueEndIndex - blocksQueueStartIndex
	} else {
		// this should only happens if we lost connection for a really long time and readIndexDiff is big, or there was a bigger fork than memory
		cs.blocksQueue = newBlocksQueue
	}
	blocksQueueLen := uint64(len(cs.blocksQueue))
	latestHash := cs.getLatestBlockUnsafe().Hash
	return blocksCopied, blocksQueueLen, latestHash
}

func (cs *ChainTracker) readHashes(latestBlock int64, ctx context.Context, blocksQueueStartIndex, blocksQueueEndIndex, newQueueStartIndex, readIndexDiff int64, newBlocksQueue []BlockStore) (int64, int64, int64, error) {
	cs.blockQueueMu.RLock()
	defer cs.blockQueueMu.RUnlock()
	// loop through our block queue and compare new hashes to previous ones to find when to stop reading
	for idx := int64(0); idx < int64(cs.blocksToSave); idx++ {
		// reading the blocks from the newest to oldest
		blockNumToFetch := latestBlock - idx
		newHashForBlock, err := cs.fetchBlockHashByNum(ctx, blockNumToFetch)
		if err != nil {
			return 0, 0, 0, utils.LavaFormatWarning("could not get block data in Chain Tracker", err, utils.Attribute{Key: "block", Value: blockNumToFetch}, utils.Attribute{Key: "ChainID", Value: cs.endpoint.ChainID}, utils.Attribute{Key: "ApiInterface", Value: cs.endpoint.ApiInterface})
		}
		var foundOverlap bool
		foundOverlap, blocksQueueStartIndex, blocksQueueEndIndex, newQueueStartIndex = cs.hashesOverlapIndexes(readIndexDiff, idx, blockNumToFetch, newHashForBlock)
		if foundOverlap {
			utils.LavaFormatDebug("Chain Tracker read a block Hash, and it existed, stopping fetch", utils.Attribute{Key: "block", Value: blockNumToFetch}, utils.Attribute{Key: "hash", Value: newHashForBlock}, utils.Attribute{Key: "KeptBlocks", Value: blocksQueueEndIndex - blocksQueueStartIndex}, utils.Attribute{Key: "ChainID", Value: cs.endpoint.ChainID}, utils.Attribute{Key: "ApiInterface", Value: cs.endpoint.ApiInterface})
			break
		}
		// there is no existing hash for this block
		newBlocksQueue[int64(cs.blocksToSave)-1-idx] = BlockStore{Block: blockNumToFetch, Hash: newHashForBlock}
	}
	return blocksQueueStartIndex, blocksQueueEndIndex, newQueueStartIndex, nil
}

// this function finds if there is an existing block data by hash at the existing data, this allows us to stop querying for further data backwards since when there is a match all former blocks are the same
// it goes over the list backwards looking for a match. when one is found it returns how many blocks are needed from the memory in order to get the required length of queue
func (cs *ChainTracker) hashesOverlapIndexes(readIndexDiff, newQueueIdx, fetchedBlockNum int64, newHashForBlock string) (foundOverlap bool, blocksQueueStartIndex, blocksQueueEndIndex, newQueueStartIndex int64) {
	savedBlocks := int64(len(cs.blocksQueue))
	if readIndexDiff >= savedBlocks {
		// we are too far ahead, there is no overlap for sure
		return false, 0, 0, 0
	}
	blocksQueueEnd := savedBlocks - 1 + readIndexDiff // this is not the real end of the queue, its incremented by readIndexDiff so we traverse it together with newBlockQueue
	blocksQueueIdx := blocksQueueEnd - newQueueIdx
	if blocksQueueIdx > 0 && blocksQueueIdx <= savedBlocks-1 {
		existingBlockStore := cs.blocksQueue[blocksQueueIdx]
		if existingBlockStore.Block != fetchedBlockNum { // sanity
			utils.LavaFormatError("mismatching blocksQueue Index and fetch index, blockStore isn't the right block", nil, utils.Attribute{
				Key: "block", Value: fetchedBlockNum,
			}, utils.Attribute{Key: "existingBlockStore", Value: existingBlockStore},
				utils.Attribute{Key: "blocksQueueIdx", Value: blocksQueueEnd}, utils.Attribute{Key: "newQueueIdx", Value: newQueueIdx}, utils.Attribute{Key: "readIndexDiff", Value: readIndexDiff})
			return false, 0, 0, 0
		}
		if existingBlockStore.Hash == newHashForBlock { // means we already have that hash, since its a blockchain, this means all previous hashes are the same too
			overwriteElements := blocksQueueIdx + 1
			if overwriteElements < int64(cs.blocksToSave)-1-newQueueIdx || readIndexDiff > overwriteElements { // make sure that in the tail we updated and the existing block we have at least cs.blocksToSave
				utils.LavaFormatError("mismatching blocksQueue Index and fetch index, there aren't enough blocks", nil, utils.Attribute{Key: "block", Value: fetchedBlockNum},
					utils.Attribute{Key: "existingBlockStore", Value: existingBlockStore},
					utils.Attribute{Key: "overwriteElements", Value: overwriteElements}, utils.Attribute{Key: "newQueueIdx", Value: newQueueIdx}, utils.Attribute{Key: "readIndexDiff", Value: readIndexDiff})
				return false, 0, 0, 0
			} else {
				return true, readIndexDiff, overwriteElements, overwriteElements - readIndexDiff
			}
		}
	}
	return false, 0, 0, 0
}

// this function reads the hash of the latest block and finds wether there was a fork, if it identifies a newer block arrived it goes backwards to the block in memory and reads again
func (cs *ChainTracker) forkChanged(ctx context.Context, newLatestBlock int64) (forked bool, err error) {
	if newLatestBlock == cs.GetAtomicLatestBlockNum() {
		// no new block arrived, compare the last hash
		hash, err := cs.fetchBlockHashByNum(ctx, newLatestBlock)
		if err != nil {
			return false, err
		}
		cs.blockQueueMu.RLock()
		defer cs.blockQueueMu.RUnlock()
		latestBlockSaved := cs.getLatestBlockUnsafe()
		return latestBlockSaved.Hash != hash, nil
	}
	// a new block was received, we need to compare a previous hash
	cs.blockQueueMu.RLock()
	latestBlockSaved := cs.getLatestBlockUnsafe()
	cs.blockQueueMu.RUnlock() // not with defer because we are going to call an external function here
	prevHash, err := cs.fetchBlockHashByNum(ctx, latestBlockSaved.Block)
	if err != nil {
		return false, err
	}
	return latestBlockSaved.Hash != prevHash, nil
}

func (cs *ChainTracker) gotNewBlock(ctx context.Context, newLatestBlock int64) (gotNewBlock bool) {
	return newLatestBlock > cs.GetAtomicLatestBlockNum()
}

// this function is periodically called, it checks if there is a new block or a fork and fetches all necessary previous data in order to fill gaps if any.
// if a new block or fork is not found, check the emergency mode
func (cs *ChainTracker) fetchAllPreviousBlocksIfNecessary(ctx context.Context) (err error) {
	newLatestBlock, err := cs.fetchLatestBlockNum(ctx)
	if err != nil {
		type wrappedError interface {
			Unwrap() error
		}
		if wrapErr, ok := err.(wrappedError); ok {
			// Check if the unwrapped error is a *net.OpError
			if _, ok := wrapErr.Unwrap().(net.Error); ok {
				cs.notUpdated()
			}
		}
		cs.pmetrics.SetLatestBlockFetchError(cs.endpoint.ChainID)
		return err
	}
	cs.pmetrics.SetLatestBlockFetchSuccess(cs.endpoint.ChainID)
	gotNewBlock := cs.gotNewBlock(ctx, newLatestBlock)
	forked, err := cs.forkChanged(ctx, newLatestBlock)
	if err != nil {
		cs.pmetrics.SetSpecificBlockFetchError(cs.endpoint.ChainID)
		return utils.LavaFormatDebug("could not fetchLatestBlock Hash in ChainTracker", utils.Attribute{Key: "error", Value: err}, utils.Attribute{Key: "block", Value: newLatestBlock}, utils.Attribute{Key: "endpoint", Value: cs.endpoint})
	}
	prev_latest := cs.GetAtomicLatestBlockNum()
	cs.pmetrics.SetSpecificBlockFetchSuccess(cs.endpoint.ChainID)
	if gotNewBlock || forked {
		latestHash, err := cs.fetchAllPreviousBlocks(ctx, newLatestBlock)
		if err != nil {
			return err
		}
		if gotNewBlock {
			if cs.newLatestCallback != nil {
				cs.newLatestCallback(prev_latest, newLatestBlock, latestHash) // TODO: this is calling the latest hash only repeatedly, this is not precise, currently not used anywhere except for prints
			}
			blocksUpdated := uint64(newLatestBlock - prev_latest)
			// update our timer resolution
			if !cs.latestChangeTime.IsZero() {
				cs.AddBlockGap(time.Since(cs.latestChangeTime), blocksUpdated)
			}
			cs.latestChangeTime = time.Now()
		}
		if forked {
			if cs.forkCallback != nil {
				cs.forkCallback(newLatestBlock)
			}
		}
	} else if prev_latest > newLatestBlock {
		if cs.consistencyCallback != nil {
			cs.consistencyCallback(prev_latest, newLatestBlock)
		}
	} else if cs.oldBlockCallback != nil {
		// if new block is not found we should check emergency mode
		cs.notUpdated()
	}

	return err
}

func (cs *ChainTracker) notUpdated() {
	latestBlockTime := cs.latestChangeTime
	if cs.latestChangeTime.IsZero() {
		latestBlockTime = cs.startupTime
	}
	cs.oldBlockCallback(latestBlockTime)
}

// this function starts the fetching timer periodically checking by polling if updates are necessary
func (cs *ChainTracker) start(ctx context.Context, pollingTime time.Duration) error {
	// how often to query latest block.
	// chainTracker polls blocks in the following strategy:
	// start polling every averageBlockTime/4, then averageBlockTime/8 after passing middle, then averageBlockTime/16 after passing averageBlockTime*3/4
	// so polling at averageBlockTime/4,averageBlockTime/2,averageBlockTime*5/8,averageBlockTime*3/4,averageBlockTime*13/16,,averageBlockTime*14/16,,averageBlockTime*15/16,averageBlockTime*16/16,averageBlockTime*17/16
	// initial polling = averageBlockTime/16
	initialPollingTime := pollingTime / MostFrequentPollingMultiplier // on boot we need to query often to catch changes
	cs.latestChangeTime = time.Time{}                                 // we will discard the first change time, so this is uninitialized
	cs.timer = time.NewTimer(initialPollingTime)
	err := cs.fetchInitDataWithRetry(ctx)
	if err != nil {
		return err
	}
	blockGapTicker := time.NewTicker(pollingTime) // initially every block we check for a polling time
	// Polls blocks and keeps a queue of them
	go func() {
		fetchFails := uint64(0)
		for {
			select {
			case <-cs.timer.C:
				utils.LavaFormatTrace("chain tracker fetch triggered", utils.LogAttr("currTime", time.Now()))
				fetchCtx, cancel := context.WithTimeout(ctx, 3*time.Second) // protect this flow from hanging code
				err := cs.fetchAllPreviousBlocksIfNecessary(fetchCtx)
				cancel()
				if err != nil {
					fetchFails += 1
					cs.updateTimer(pollingTime, fetchFails)
					if fetchFails > maxFails {
						utils.LavaFormatError("failed to fetch all previous blocks and was necessary", err, utils.Attribute{Key: "fetchFails", Value: fetchFails}, utils.Attribute{Key: "endpoint", Value: cs.endpoint.String()})
					} else {
						utils.LavaFormatDebug("failed to fetch all previous blocks", utils.Attribute{Key: "error", Value: err}, utils.Attribute{Key: "fetchFails", Value: fetchFails}, utils.Attribute{Key: "endpoint", Value: cs.endpoint.String()})
					}
				} else {
					cs.updateTimer(pollingTime, 0)
					fetchFails = 0
				}
			case <-blockGapTicker.C:
				var enoughSamples bool
				pollingTime, enoughSamples = cs.updatePollingTimeBasedOnBlockGap(pollingTime)
				if enoughSamples {
					blockGapTicker.Reset(pollingTime * 10)
				}
			case <-ctx.Done():
				cs.timer.Stop()
				return
			}
		}
	}()
	return nil
}

func (cs *ChainTracker) updateTimer(tickerBaseTime time.Duration, fetchFails uint64) {
	blockGap := cs.smallestBlockGap()
	timeSinceLastUpdate := time.Since(cs.latestChangeTime)
	var newPollingTime time.Duration
	if timeSinceLastUpdate <= tickerBaseTime/2 && blockGap > tickerBaseTime/4 {
		newPollingTime = tickerBaseTime / (MostFrequentPollingMultiplier / 4)
	} else if timeSinceLastUpdate <= (tickerBaseTime*3)/4 && blockGap > tickerBaseTime/4 {
		newPollingTime = tickerBaseTime / (MostFrequentPollingMultiplier / 2)
	} else {
		newPollingTime = tickerBaseTime / MostFrequentPollingMultiplier
	}
	newTickerDuration := exponentialBackoff(newPollingTime, fetchFails)
	if PollingMultiplier > 1 {
		newTickerDuration /= time.Duration(PollingMultiplier)
	}

	utils.LavaFormatTrace("state tracker ticker set",
		utils.LogAttr("timeSinceLastUpdate", timeSinceLastUpdate),
		utils.LogAttr("time", time.Now()),
		utils.LogAttr("newTickerDuration", newTickerDuration),
	)

	cs.timer = time.NewTimer(newTickerDuration)
}

func (cs *ChainTracker) fetchInitDataWithRetry(ctx context.Context) (err error) {
	var newLatestBlock int64
	for idx := 0; idx < initRetriesCount+1; idx++ {
		newLatestBlock, err = cs.fetchLatestBlockNum(ctx)
		if err == nil {
			break
		}
		utils.LavaFormatDebug("failed fetching block num data on chain tracker init, retry", utils.Attribute{Key: "retry Num", Value: idx}, utils.Attribute{Key: "endpoint", Value: cs.endpoint})
	}
	if err != nil {
		// Add suggestion if error is due to context deadline exceeded
		if common.ContextDeadlineExceededError.Is(err) {
			utils.LavaFormatError("suggestion -- If you encounter a 'context deadline exceeded' error, consider increasing the timeout configuration in the 'node-url' config option. Sometimes, the initial HTTPS/WSS communication takes a long time to establish a connection.", nil)
		}
		return utils.LavaFormatError("critical -- failed fetching data from the node, chain tracker creation error", err, utils.Attribute{Key: "endpoint", Value: cs.endpoint})
	}
	for idx := 0; idx < initRetriesCount; idx++ {
		_, err = cs.fetchAllPreviousBlocks(ctx, newLatestBlock)
		if err == nil {
			break
		}
		utils.LavaFormatDebug("failed fetching data on chain tracker init, retry", utils.Attribute{Key: "retry Num", Value: idx}, utils.Attribute{Key: "endpoint", Value: cs.endpoint.String()})
	}
	if err != nil {
		// Add suggestion if error is due to context deadline exceeded
		if common.ContextDeadlineExceededError.Is(err) {
			utils.LavaFormatError("suggestion -- If you encounter a 'context deadline exceeded' error, consider increasing the timeout configuration in the 'node-url' config option. Sometimes, the initial HTTPS/WSS communication takes a long time to establish a connection.", nil)
		}
		return utils.LavaFormatError("critical -- failed fetching data from the node, chain tracker creation error", err, utils.Attribute{Key: "endpoint", Value: cs.endpoint})
	}
	return nil
}

func (ct *ChainTracker) updatePollingTimeBasedOnBlockGap(pollingTime time.Duration) (pollTime time.Duration, enoughSampled bool) {
	blockGapsLen := len(ct.blockEventsGap)
	if blockGapsLen > PollingUpdateLength { // check we have enough samples
		// smaller times give more resolution to indentify changes, and also make block arrival predictions more optimistic
		// so we take a 0.33 percentile because we want to be on the safe side by have a smaller time than expected
		percentileTime := lavaslices.Percentile(ct.blockEventsGap, 0.33)
		stability := lavaslices.Stability(ct.blockEventsGap, percentileTime)
		utils.LavaFormatTrace("block gaps",
			utils.LogAttr("block gaps", ct.blockEventsGap),
			utils.LogAttr("specID", ct.endpoint.ChainID),
		)

		if blockGapsLen > int(ct.serverBlockMemory)-2 || stability < GoodStabilityThreshold {
			// only update if there is a 10% difference or more
			if percentileTime < (pollingTime*9/10) || percentileTime > (pollingTime*11/10) {
				utils.LavaFormatInfo("updated chain tracker polling time", utils.Attribute{Key: "blocks measured", Value: blockGapsLen}, utils.Attribute{Key: "median new polling time", Value: percentileTime}, utils.Attribute{Key: "original polling time", Value: pollingTime}, utils.Attribute{Key: "chainID", Value: ct.endpoint.ChainID}, utils.Attribute{Key: "stability", Value: stability})
				if percentileTime > pollingTime*2 {
					utils.LavaFormatWarning("[-] substantial polling time increase for chain detected", nil, utils.Attribute{Key: "median new polling time", Value: percentileTime}, utils.Attribute{Key: "original polling time", Value: pollingTime}, utils.Attribute{Key: "chainID", Value: ct.endpoint.ChainID}, utils.Attribute{Key: "stability", Value: stability})
				}
				go ct.updateAverageBlockTimeForRegistrations(percentileTime)
				return percentileTime, true
			}
			return pollingTime, true
		} else {
			utils.LavaFormatTrace("current stability measurement",
				utils.LogAttr("chainID", ct.endpoint.ChainID),
				utils.LogAttr("stability", stability),
			)
		}
	}
	return pollingTime, false
}

func (ct *ChainTracker) AddBlockGap(newData time.Duration, blocks uint64) {
	averageBlockTimeForOneBlock := newData / time.Duration(blocks)
	if uint64(len(ct.blockEventsGap)) < ct.serverBlockMemory {
		ct.blockEventsGap = append(ct.blockEventsGap, averageBlockTimeForOneBlock)
	} else {
		// we need to discard an index at random because this list is sorted by values and not by insertion time
		randomIndex := rand.Intn(len(ct.blockEventsGap)) // it's not inclusive so len is fine
		ct.blockEventsGap[randomIndex] = averageBlockTimeForOneBlock
	}
}

func (ct *ChainTracker) smallestBlockGap() time.Duration {
	length := len(ct.blockEventsGap)
	if length < PollingUpdateLength {
		return 0
	}
	return ct.blockEventsGap[0] // this list is sorted
}

// this function serves a grpc server if configuration for it was provided, the goal is to enable stateTracker to serve several processes and minimize node queries
func (ct *ChainTracker) serve(ctx context.Context, listenAddr string) error {
	if listenAddr == "" {
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	defer func() {
		signal.Stop(signalChan)
		cancel()
	}()
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		utils.LavaFormatFatal("Chain Tracker failure setting up listener", err, utils.Attribute{Key: "listenAddr", Value: listenAddr})
	}
	s := grpc.NewServer()

	wrappedServer := grpcweb.WrapServer(s)
	handler := func(resp http.ResponseWriter, req *http.Request) {
		// Set CORS headers
		resp.Header().Set("Access-Control-Allow-Origin", "*")
		resp.Header().Set("Access-Control-Allow-Headers", "Content-Type,x-grpc-web")

		wrappedServer.ServeHTTP(resp, req)
	}

	httpServer := http.Server{
		Handler: h2c.NewHandler(http.HandlerFunc(handler), &http2.Server{}),
	}

	go func() {
		select {
		case <-ctx.Done():
			utils.LavaFormatInfo("Chain Tracker Server ctx.Done")
		case <-signalChan:
			utils.LavaFormatInfo("Chain Tracker Server signalChan")
		}

		shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownRelease()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			utils.LavaFormatFatal("chainTracker failed to shutdown", err)
		}
	}()

	server := &ChainTrackerService{ChainTracker: ct}

	RegisterChainTrackerServiceServer(s, server)

	utils.LavaFormatInfo("Chain Tracker Listening", utils.Attribute{Key: "Address", Value: lis.Addr().String()})
	if err := httpServer.Serve(lis); !errors.Is(err, http.ErrServerClosed) {
		utils.LavaFormatFatal("Chain Tracker failed to serve", err, utils.Attribute{Key: "Address", Value: lis.Addr()})
	}
	return nil
}

func NewChainTracker(ctx context.Context, chainFetcher ChainFetcher, config ChainTrackerConfig) (chainTracker *ChainTracker, err error) {
	if !rand.Initialized() {
		utils.LavaFormatFatal("can't start chainTracker with nil rand source", nil)
	}
	err = config.validate()
	if err != nil {
		return nil, err
	}

	chainTracker = &ChainTracker{
		consistencyCallback:     config.ConsistencyCallback,
		forkCallback:            config.ForkCallback,
		newLatestCallback:       config.NewLatestCallback,
		oldBlockCallback:        config.OldBlockCallback,
		blocksToSave:            config.BlocksToSave,
		chainFetcher:            chainFetcher,
		latestBlockNum:          0,
		serverBlockMemory:       config.ServerBlockMemory,
		blockCheckpointDistance: config.BlocksCheckpointDistance,
		blockEventsGap:          []time.Duration{},
		blockTimeUpdatables:     map[blockTimeUpdatable]struct{}{},
		startupTime:             time.Now(),
		pmetrics:                config.Pmetrics,
	}
	if chainFetcher == nil {
		return nil, utils.LavaFormatError("can't start chainTracker with nil chainFetcher argument", nil)
	}
	chainTracker.endpoint = chainFetcher.FetchEndpoint()
	err = chainTracker.start(ctx, config.AverageBlockTime)
	if err != nil {
		return nil, err
	}

	err = chainTracker.serve(ctx, config.ServerAddress)
	return chainTracker, err
}
