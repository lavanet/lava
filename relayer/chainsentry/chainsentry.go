package chainsentry

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/lavanet/lava/relayer/chainproxy"
)

const (
	DEFAULT_NUM_FINAL_BLOCKS = 3
	DEFAULT_NUM_SAVED_BLOCKS = 7
)

type ChainSentry struct {
	chainProxy             chainproxy.ChainProxy
	finalizedBlockDistance int   // the distance from the latest block to the latest finalized block - in eth its 7
	numFinalBlocks         int   // how many finalized blocks to keep
	latestBlockNum         int64 // uint?
	ChainID                string

	// Spec blockQueueMu (rw mutex)
	blockQueueMu sync.RWMutex
	blocksQueue  []string
}

type jsonError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type jsonrpcMessage struct {
	Version string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  []interface{}   `json:"params,omitempty"`
	Error   *jsonError      `json:"error,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
}

func (cs *ChainSentry) GetLatestBlockNum() int64 {
	return atomic.LoadInt64(&cs.latestBlockNum)
}

func (cs *ChainSentry) SetLatestBlockNum(value int64) {
	cs.blockQueueMu.Lock()
	defer cs.blockQueueMu.Unlock()
	atomic.StoreInt64(&cs.latestBlockNum, value)
}

func (cs *ChainSentry) GetLatestBlockData() (int64, map[int64]interface{}, error) {
	cs.blockQueueMu.Lock()
	defer cs.blockQueueMu.Unlock()

	latestBlockNum := cs.GetLatestBlockNum()
	var hashes = make(map[int64]interface{}, len(cs.blocksQueue))

	for i := 0; i < cs.numFinalBlocks; i++ {
		blockNum := latestBlockNum - int64(cs.finalizedBlockDistance) - int64(cs.numFinalBlocks) + int64(i)
		hashes[blockNum] = cs.blocksQueue[i]
	}
	return latestBlockNum, hashes, nil
}

func (cs *ChainSentry) fetchLatestBlockNum(ctx context.Context) (int64, error) {
	return cs.chainProxy.FetchLatestBlockNum(ctx)
}

func (cs *ChainSentry) fetchBlockHashByNum(ctx context.Context, blockNum int64) (string, error) {
	return cs.chainProxy.FetchBlockHashByNum(ctx, blockNum)

}

func (cs *ChainSentry) Init(ctx context.Context) error {
	latestBlock, err := cs.fetchLatestBlockNum(ctx)
	// TODO:: chekc if we have at least x blocknums before forloop
	if err != nil {
		log.Fatalln("error: Start", err)
		return err
	}

	cs.SetLatestBlockNum(latestBlock)
	log.Printf("latest %v block %v", cs.ChainID, latestBlock)
	cs.blockQueueMu.Lock()
	// TODO:: move cs.numSavedBlocks+cs.numFinalBlocks to get finalizationBlockDistance
	// TODO:: get finalized blocks only - not latest ( 100 -> 83-93)
	for i := latestBlock - int64(cs.finalizedBlockDistance+cs.numFinalBlocks); i <= latestBlock-int64(cs.finalizedBlockDistance)-1; i++ {
		result, err := cs.fetchBlockHashByNum(ctx, i)
		if err != nil {
			log.Fatalln("error: Start", err)
			return err
		}

		log.Printf("Block number: %d, block hash: %s", i, result)
		cs.blocksQueue = append(cs.blocksQueue, result) // save entire block data for now
	}
	cs.blockQueueMu.Unlock()

	return nil
}

func (cs *ChainSentry) catchupOnFinalizedBlocks(ctx context.Context) error {
	latestBlock, err := cs.fetchLatestBlockNum(ctx) // get actual latest from chain
	if err != nil {
		log.Printf("error: chainSentry block fetcher", err)
	}

	// TODO:: dont lock for this entire process. create a temp list and replace blocksqueue. like in sentry service api
	if cs.latestBlockNum != latestBlock {
		cs.blockQueueMu.Lock()

		prevLatestBlock := cs.GetLatestBlockNum()
		// Get all missing blocks
		//max x missing blocks from (latestbloc - how many we need )
		for i := prevLatestBlock + 1; i <= latestBlock; i++ {
			blockData, err := cs.fetchBlockHashByNum(ctx, i)
			if err != nil {
				log.Fatalln("error: Start", err)
				return err
			}

			cs.blocksQueue = append(cs.blocksQueue, blockData) //save entire block data for now
		}

		// remove the first entries from the queue (oldest ones)
		if len(cs.blocksQueue) > cs.numFinalBlocks {
			cs.blocksQueue = cs.blocksQueue[(len(cs.blocksQueue) - cs.numFinalBlocks):]
		}
		log.Printf("chainSentry blocks list updated. latest block %d", latestBlock)
		atomic.StoreInt64(&cs.latestBlockNum, latestBlock)
		cs.blockQueueMu.Unlock()
	}
	return nil
}

func (cs *ChainSentry) Start(ctx context.Context) error {
	ticker := time.NewTicker(time.Second * 5)
	quit := make(chan struct{})

	// Polls blocks and keeps a queue of them
	go func() {
		for {
			select {
			case <-ticker.C:
				cs.catchupOnFinalizedBlocks(ctx)
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	return nil
}

func NewChainSentry(
	clientCtx client.Context,
	cp chainproxy.ChainProxy,
	chainID string,
) *ChainSentry {
	return &ChainSentry{
		chainProxy:             cp,
		ChainID:                chainID,
		numFinalBlocks:         int(cp.GetSentry().GetSpecSavedBlocks()),
		finalizedBlockDistance: int(cp.GetSentry().GetSpecFinalizationCriteria()),
	}
}
