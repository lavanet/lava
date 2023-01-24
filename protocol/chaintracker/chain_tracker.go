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

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/lavanet/lava/utils"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	grpc "google.golang.org/grpc"
)

type ChainFetcher interface {
	FetchLatestBlockNum(ctx context.Context) (int64, error)
	FetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error)
}

type ChainTracker struct {
	chainFetcher      ChainFetcher // used to communicate with the node
	blocksToSave      uint64       // how many finalized blocks to keep
	latestBlockNum    int64
	blockQueueMu      sync.RWMutex
	blocksQueue       []BlockStore // holds all past hashes up until latest block
	forkCallback      func(int64)  // a function to be called when a fork is detected
	newLatestCallback func(int64)  // a function to be called when a new block is detected
	serverBlockMemory uint64
	quit              chan bool
}

// this function returns block hashes of the blocks: [from block - to block) non inclusive. an additional specific block hash can be provided. order is sorted ascending
// it supports requests for [spectypes.LATEST_BLOCK-distance1, spectypes.LATEST_BLOCK-distance2)
// spectypes.NOT_APPLICABLE in fromBlock or toBlock results in only returning specific block.
// if specific block is spectypes.NOT_APPLICABLE it is ignored
func (cs *ChainTracker) GetLatestBlockData(fromBlock int64, toBlock int64, specificBlock int64) (latestBlock int64, requestedHashes []*BlockStore, err error) {
	cs.blockQueueMu.RLock()
	defer cs.blockQueueMu.RUnlock()

	latestBlock = cs.GetLatestBlockNum()
	if len(cs.blocksQueue) == 0 {
		return latestBlock, nil, utils.LavaFormatError("ChainTracker GetLatestBlockData had no blocks", nil, &map[string]string{"latestBlock": strconv.FormatInt(latestBlock, 10)})
	}
	earliestBlockSaved := cs.getEarliestBlockUnsafe().Block
	wantedBlocksData := WantedBlocksData{}
	err = wantedBlocksData.New(fromBlock, toBlock, specificBlock, latestBlock, earliestBlockSaved)
	if err != nil {
		return latestBlock, nil, utils.LavaFormatError("invalid input for GetLatestBlockData", err, &map[string]string{
			"fromBlock": strconv.FormatInt(fromBlock, 10), "toBlock": strconv.FormatInt(toBlock, 10), "specificBlock": strconv.FormatInt(specificBlock, 10),
			"latestBlock": strconv.FormatInt(latestBlock, 10), "earliestBlockSaved": strconv.FormatInt(earliestBlockSaved, 10),
		})
	}

	for _, blocksQueueIdx := range wantedBlocksData.IterationIndexes() {
		blockStore := cs.blocksQueue[blocksQueueIdx]
		if !wantedBlocksData.IsWanted(blockStore.Block) {
			return latestBlock, nil, utils.LavaFormatError("invalid wantedBlocksData Iteration", err, &map[string]string{
				"blocksQueueIdx": strconv.FormatInt(int64(blocksQueueIdx), 10), "blockStore": fmt.Sprintf("%+v", blockStore), "wantedBlocksData": wantedBlocksData.String(),
			})
		}
		requestedHashes = append(requestedHashes, &blockStore)
	}
	return
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

func (cs *ChainTracker) GetLatestBlockNum() int64 {
	return atomic.LoadInt64(&cs.latestBlockNum)
}

func (cs *ChainTracker) setLatestBlockNum(value int64) {
	atomic.StoreInt64(&cs.latestBlockNum, value)
}

func (cs *ChainTracker) fetchLatestBlockNum(ctx context.Context) (int64, error) {
	return cs.chainFetcher.FetchLatestBlockNum(ctx)
}

func (cs *ChainTracker) fetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error) {
	if blockNum < cs.GetLatestBlockNum()-int64(cs.serverBlockMemory) {
		return "", ErrorFailedToFetchTooEarlyBlock.Wrapf("requested Block: %d, latest block: %d, server memory %d", blockNum, cs.GetLatestBlockNum(), cs.serverBlockMemory)
	}
	return cs.chainFetcher.FetchBlockHashByNum(ctx, blockNum)
}

// this function fetches all previous blocks from the node starting at the latest provided going backwards blocksToSave blocks
// if it reaches a hash that it already has it stops reading
func (cs *ChainTracker) fetchAllPreviousBlocks(ctx context.Context, latestBlock int64) (err error) {
	newBlocksQueue := make([]BlockStore, int64(cs.blocksToSave))
	currentLatestBlock := cs.GetLatestBlockNum()
	if latestBlock < currentLatestBlock {
		return utils.LavaFormatError("invalid latestBlock provided to fetch, it is older than the current state latest block", err, &map[string]string{"latestBlock": strconv.FormatInt(latestBlock, 10), "currentLatestBlock": strconv.FormatInt(currentLatestBlock, 10)})
	}
	readIndexDiff := latestBlock - currentLatestBlock
	blocksQueueStartIndex, blocksQueueEndIndex, newQueueStartIndex := int64(0), int64(0), int64(0)
	cs.blockQueueMu.RLock()
	// loop through our block queue and compare new hashes to previous ones to find when to stop reading
	for idx := int64(0); idx < int64(cs.blocksToSave); idx++ {
		blockNumToFetch := latestBlock - idx // reading the blocks from the newest to oldest
		newHashForBlock, err := cs.fetchBlockHashByNum(ctx, blockNumToFetch)
		if err != nil {
			return utils.LavaFormatError("could not get block data in Chain Tracker", err, &map[string]string{"block": strconv.FormatInt(blockNumToFetch, 10)})
		}
		var foundOverlap bool
		foundOverlap, blocksQueueStartIndex, blocksQueueEndIndex, newQueueStartIndex = cs.hashesOverlapIndexes(readIndexDiff, idx, blockNumToFetch, newHashForBlock)
		if foundOverlap {
			utils.LavaFormatDebug("Chain Tracker read a block Hash, and it existed, stopping fetch", &map[string]string{"block": strconv.FormatInt(blockNumToFetch, 10), "hash": newHashForBlock, "KeptBlocks": strconv.FormatInt(blocksQueueEndIndex-blocksQueueStartIndex, 10)})
			break
		}
		// there is no existing hash for this block
		utils.LavaFormatDebug("Chain Tracker read a new block hash", &map[string]string{"block": strconv.FormatInt(blockNumToFetch, 10), "newHash": newHashForBlock})
		newBlocksQueue[int64(cs.blocksToSave)-1-idx] = BlockStore{Block: blockNumToFetch, Hash: newHashForBlock}
	}
	cs.blockQueueMu.RUnlock()
	blocksCopied := int64(cs.blocksToSave)
	cs.blockQueueMu.Lock()
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
	cs.blockQueueMu.Unlock()
	if blocksQueueLen < cs.blocksToSave {
		return utils.LavaFormatError("fetchAllPreviousBlocks didn't save enough blocks in Chain Tracker", nil, &map[string]string{"blocksQueueLen": strconv.FormatUint(blocksQueueLen, 10)})
	} else {
		utils.LavaFormatInfo("Chain Tracker Updated latest block", &map[string]string{"block": strconv.FormatInt(latestBlock, 10), "latestHash": latestHash, "blocksQueueLen": strconv.FormatUint(blocksQueueLen, 10), "blocksQueried": strconv.FormatInt(blocksCopied, 10)})
	}
	return nil
}

// this function finds if there is an existing block data by hash at the existing data, this allows us to stop querying for further data backwards since when there is a match all former blocks are the same
// it goes over the list backwards looking for a match. when one is found it returns how many blocks are needed from the memory in order to get the required length of queue
func (cs *ChainTracker) hashesOverlapIndexes(readIndexDiff int64, newQueueIdx int64, fetchedBlockNum int64, newHashForBlock string) (foundOverlap bool, blocksQueueStartIndex int64, blocksQueueEndIndex int64, newQueueStartIndex int64) {
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
			utils.LavaFormatError("mismatching blocksQueue Index and fetch index, blockStore isn't the right block", nil, &map[string]string{
				"block": strconv.FormatInt(fetchedBlockNum, 10), "existingBlockStore": fmt.Sprintf("%+v", existingBlockStore),
				"blocksQueueIdx": strconv.FormatInt(blocksQueueEnd, 10), "newQueueIdx": strconv.FormatInt(newQueueIdx, 10), "readIndexDiff": strconv.FormatInt(readIndexDiff, 10),
			})
			return false, 0, 0, 0
		}
		if existingBlockStore.Hash == newHashForBlock { // means we already have that hash, since its a blockchain, this means all previous hashes are the same too
			overwriteElements := blocksQueueIdx + 1
			if overwriteElements < int64(cs.blocksToSave)-1-newQueueIdx || readIndexDiff > overwriteElements { // make sure that in the tail we updated and the existing block we have at least cs.blocksToSave
				utils.LavaFormatError("mismatching blocksQueue Index and fetch index, there aren't enough blocks", nil, &map[string]string{
					"block": strconv.FormatInt(fetchedBlockNum, 10), "existingBlockStore": fmt.Sprintf("%+v", existingBlockStore),
					"overwriteElements": strconv.FormatInt(overwriteElements, 10), "newQueueIdx": strconv.FormatInt(newQueueIdx, 10), "readIndexDiff": strconv.FormatInt(readIndexDiff, 10),
				})
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
	if newLatestBlock == cs.GetLatestBlockNum() {
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
	return newLatestBlock > cs.GetLatestBlockNum()
}

// this function is periodically called, it checks if there is a new block or a fork and fetches all necessary previous data in order to fill gaps if any
func (cs *ChainTracker) fetchAllPreviousBlocksIfNecessary(ctx context.Context) (err error) {
	newLatestBlock, err := cs.fetchLatestBlockNum(ctx)
	if err != nil {
		return utils.LavaFormatError("could not fetchLatestBlockNum in ChainTracker", err, nil)
	}
	gotNewBlock := cs.gotNewBlock(ctx, newLatestBlock)
	forked, err := cs.forkChanged(ctx, newLatestBlock)
	if err != nil {
		return utils.LavaFormatError("could not fetchLatestBlock Hash in ChainTracker", err, &map[string]string{"block": strconv.FormatInt(newLatestBlock, 10)})
	}
	if gotNewBlock || forked {
		utils.LavaFormatDebug("ChainTracker should update state", &map[string]string{"gotNewBlock": fmt.Sprintf("%t", gotNewBlock), "forked": fmt.Sprintf("%t", forked), "newLatestBlock": strconv.FormatInt(newLatestBlock, 10), "currentBlock": strconv.FormatInt(cs.GetLatestBlockNum(), 10)})
		// TODO: if we didn't fork theres really no need to refetch
		cs.fetchAllPreviousBlocks(ctx, newLatestBlock)
		if gotNewBlock {
			if cs.newLatestCallback != nil {
				cs.newLatestCallback(newLatestBlock)
			}
		}
		if forked {
			if cs.forkCallback != nil {
				cs.forkCallback(newLatestBlock)
			}
		}
	}
	return
}

// this function starts the fetching timer periodically checking by polling if updates are necessary
func (cs *ChainTracker) start(ctx context.Context, pollingBlockTime time.Duration) error {
	// how often to query latest block.
	// TODO: subscribe instead of repeatedly fetching
	ticker := time.NewTicker(pollingBlockTime)

	newLatestBlock, err := cs.fetchLatestBlockNum(ctx)
	if err != nil {
		utils.LavaFormatFatal("could not fetchLatestBlockNum in ChainTracker", err, nil)
	}
	cs.fetchAllPreviousBlocks(ctx, newLatestBlock)
	// Polls blocks and keeps a queue of them
	go func() {
		for {
			select {
			case <-ticker.C:
				err := cs.fetchAllPreviousBlocksIfNecessary(ctx)
				if err != nil {
					utils.LavaFormatError("failed to fetch all previous blocks and was necessary", err, nil)
				}
			case <-cs.quit:
				ticker.Stop()
				return
			}
		}
	}()

	return nil
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
		utils.LavaFormatFatal("Chain Tracker failure setting up listener", err, &map[string]string{"listenAddr": listenAddr})
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
			utils.LavaFormatInfo("Chain Tracker Server ctx.Done", nil)
		case <-signalChan:
			utils.LavaFormatInfo("Chain Tracker Server signalChan", nil)
		}

		shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownRelease()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			utils.LavaFormatFatal("chainTracker failed to shutdown", err, &map[string]string{})
		}
	}()

	server := &ChainTrackerService{ChainTracker: ct}

	RegisterChainTrackerServiceServer(s, server)

	utils.LavaFormatInfo("Chain Tracker Listening", &map[string]string{"Address": lis.Addr().String()})
	if err := httpServer.Serve(lis); !errors.Is(err, http.ErrServerClosed) {
		utils.LavaFormatFatal("Chain Tracker failed to serve", err, &map[string]string{"Address": lis.Addr().String()})
	}
	return nil
}

func New(ctx context.Context, chainFetcher ChainFetcher, config ChainTrackerConfig) (chainTracker *ChainTracker, err error) {
	config.validate()
	chainTracker = &ChainTracker{forkCallback: config.ForkCallback, newLatestCallback: config.NewLatestCallback, blocksToSave: config.BlocksToSave, chainFetcher: chainFetcher, latestBlockNum: 0, serverBlockMemory: config.ServerBlockMemory}
	chainTracker.start(ctx, config.AverageBlockTime)
	chainTracker.serve(ctx, config.ServerAddress)
	return
}
