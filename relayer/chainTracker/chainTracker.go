package chaintracker

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/lavanet/lava/relayer/chainproxy"
	"github.com/lavanet/lava/utils"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	grpc "google.golang.org/grpc"
)

type ChainTracker struct {
	chainProxy        chainproxy.ChainProxy //used to communicate with the node
	blocksToSave      uint64                // how many finalized blocks to keep
	latestBlockNum    int64
	blockQueueMu      sync.RWMutex
	blocksQueue       []BlockStore // holds all past hashes up until latest block
	forkCallback      func()       //a function to be called when a fork is detected
	newLatestCallback func()       //a function to be called when a new block is detected
}

// returns hashes [fromBlock - toBlock) non inclusive, its sorted from smallest to highest. to get relative to latest it accepts spectypes.LATEST_BLOCK-distance
// to get only the latest use specific block
func (cs *ChainTracker) GetLatestBlockData(fromBlock int64, toBlock int64, specificBlock int64) (latestBlock int64, requestedHashes []*BlockStore, err error) {
	cs.blockQueueMu.RLock()
	defer cs.blockQueueMu.RUnlock()

	latestBlock = cs.GetLatestBlockNum()
	if len(cs.blocksQueue) == 0 {
		return latestBlock, nil, utils.LavaFormatError("ChainTracker GetLatestBlockData had no blocks", nil, &map[string]string{"latestBlock": strconv.FormatInt(latestBlock, 10)})
	}

	fromBlock = formatRequestedBlock(fromBlock, latestBlock)
	toBlock = formatRequestedBlock(toBlock, latestBlock)
	specificBlock = formatRequestedBlock(specificBlock, latestBlock)

	earliestBlock := cs.blocksQueue[0].Block
	earliestRes := earliestBlock
	if fromBlock > 0 && fromBlock < specificBlock {
		earliestRes = fromBlock
	} else if specificBlock > 0 && specificBlock < fromBlock {
		earliestRes = specificBlock
	} else {
		//both not applicable, this is wrong
		return latestBlock, nil, utils.LavaFormatError("ChainTracker GetLatestBlockData invalid requested blocks, both specific and from are NOT_APPLICABLE", nil, &map[string]string{"fromBlock": strconv.FormatInt(fromBlock, 10), "specificBlock": strconv.FormatInt(specificBlock, 10)})
	}
	var queueIdx int64 = 0
	if earliestRes <= earliestBlock {
		queueIdx = 0
	} else {
		queueIdx = earliestRes - earliestBlock
	}

	for queueIdx < int64(len(cs.blocksQueue)) {
		blockStore := cs.blocksQueue[queueIdx]
		from_block_cond := fromBlock <= blockStore.Block
		to_block_cond := toBlock == spectypes.NOT_APPLICABLE || blockStore.Block < toBlock
		specific_block_cond := blockStore.Block == specificBlock
		if specific_block_cond || (from_block_cond && to_block_cond) {
			requestedHashes = append(requestedHashes, &blockStore)
		}
	}
	return
}

func (cs *ChainTracker) GetLatestBlockNum() int64 {
	return atomic.LoadInt64(&cs.latestBlockNum)
}

func formatRequestedBlock(requested int64, latestBlock int64) int64 {
	//if later we want to support other things like earliest, pending, finalized or safe this needs to change
	if requested >= latestBlock {
		return latestBlock
	}
	if requested >= spectypes.NOT_APPLICABLE {
		return spectypes.NOT_APPLICABLE
	}
	requestedRes := latestBlock + requested - spectypes.LATEST_BLOCK
	if requestedRes < 0 {
		return spectypes.NOT_APPLICABLE
	}
	return requestedRes
}

func (cs *ChainTracker) setLatestBlockNum(value int64) {
	atomic.StoreInt64(&cs.latestBlockNum, value)
}

func (cs *ChainTracker) fetchLatestBlockNum(ctx context.Context) (int64, error) {
	return cs.chainProxy.FetchLatestBlockNum(ctx)
}

func (cs *ChainTracker) fetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error) {
	return cs.chainProxy.FetchBlockHashByNum(ctx, blockNum)
}

func (cs *ChainTracker) fetchAllPreviousBlocks(ctx context.Context, latestBlock int64) (err error) {
	newBlocksQueue := []BlockStore{}
	for blockNumToFetch := latestBlock - int64(cs.blocksToSave) + 1; blockNumToFetch <= latestBlock; blockNumToFetch++ { // save all blocks from the past up until latest block
		result, err := cs.fetchBlockHashByNum(ctx, blockNumToFetch)
		if err != nil {
			return utils.LavaFormatError("could not get block data in ChainTracker", err, &map[string]string{"block": strconv.FormatInt(blockNumToFetch, 10)})
		}

		utils.LavaFormatDebug("ChainTracker read a block", &map[string]string{"block": strconv.FormatInt(blockNumToFetch, 10), "result": result})
		newBlocksQueue = append(newBlocksQueue, BlockStore{Block: blockNumToFetch, Hash: result}) // save entire block data for now
	}
	cs.blockQueueMu.Lock()
	cs.setLatestBlockNum(latestBlock)
	cs.blocksQueue = newBlocksQueue
	blocksQueueLen := int64(len(cs.blocksQueue))
	latestHash := cs.blocksQueue[blocksQueueLen-1].Hash
	cs.blockQueueMu.Unlock()
	utils.LavaFormatInfo("ChainSentry Updated latest block", &map[string]string{"block": strconv.FormatInt(latestBlock, 10), "latestHash": latestHash, "blocksQueueLen": strconv.FormatInt(blocksQueueLen, 10)})
	return nil
}

func (cs *ChainTracker) forkChanged(ctx context.Context, newLatestBlock int64) (forked bool, err error) {
	hash, err := cs.fetchBlockHashByNum(ctx, newLatestBlock)
	if err != nil {
		return
	}
	cs.blockQueueMu.RLock()
	defer cs.blockQueueMu.RUnlock()
	latestHash := cs.blocksQueue[len(cs.blocksQueue)-1].Hash
	return latestHash == hash, nil
}

func (cs *ChainTracker) gotNewBlock(ctx context.Context, newLatestBlock int64) (gotNewBlock bool) {
	return newLatestBlock > cs.GetLatestBlockNum()
}

func (cs *ChainTracker) fetchAllPreviousBlocksIfNecessary(ctx context.Context) (err error) {
	newLatestBlock, err := cs.fetchLatestBlockNum(ctx)
	if err != nil {
		return utils.LavaFormatError("could not fetchLatestBlockNum in ChainTracker", err, nil)
	}
	forked, err := cs.forkChanged(ctx, newLatestBlock)
	if err != nil {
		return utils.LavaFormatError("could not fetchLatestBlock Hash in ChainTracker", err, &map[string]string{"block": strconv.FormatInt(newLatestBlock, 10)})
	}
	gotNewBlock := cs.gotNewBlock(ctx, newLatestBlock)
	if gotNewBlock || forked {
		cs.fetchAllPreviousBlocks(ctx, newLatestBlock)
		if gotNewBlock {
			cs.newLatestCallback()
		}
		if forked {
			cs.forkCallback()
		}
	}
	return
}

func (cs *ChainTracker) start(ctx context.Context, averageBlockTime time.Duration) error {
	// how often to query latest block.
	//TODO: subscribe instead of repeatedly fetching
	ticker := time.NewTicker(averageBlockTime / 3) //sample it three times as much

	// Polls blocks and keeps a queue of them
	go func() {
		for {
			select {
			case <-ticker.C:
				err := cs.fetchAllPreviousBlocksIfNecessary(ctx)
				if err != nil {
					utils.LavaFormatError("failed to fetch all previous blocks and was necessary", err, nil)
				}
			}
		}
	}()

	return nil
}

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
		utils.LavaFormatFatal("ChainTracker failure setting up listener", err, &map[string]string{"listenAddr": listenAddr})
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
			utils.LavaFormatInfo("chainTracker Server ctx.Done", nil)
		case <-signalChan:
			utils.LavaFormatInfo("chainTracker Server signalChan", nil)
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

func New(ctx context.Context, cp chainproxy.ChainProxy, config ChainTrackerConfig) (chainTracker *ChainTracker, err error) {
	config.validate()
	chainTracker = &ChainTracker{forkCallback: config.ForkCallback, newLatestCallback: config.NewLatestCallback, blocksToSave: config.blocksToSave, chainProxy: cp}
	chainTracker.start(ctx, config.averageBlockTime)
	chainTracker.serve(ctx, config.ServerAddress)
	return
}
