package chainsentry

import (
	"context"
	"log"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/lavanet/lava/relayer/chainproxy"
	"github.com/lavanet/lava/relayer/sentry"
	"github.com/lavanet/lava/utils"
)

type ChainSentry struct {
	chainProxy             chainproxy.ChainProxy
	finalizedBlockDistance int   // the distance from the latest block to the latest finalized block - in eth its 7
	numFinalBlocks         int   // how many finalized blocks to keep
	latestBlockNum         int64 // uint?
	ChainID                string

	quit chan bool
	// Spec blockQueueMu (rw mutex)
	blockQueueMu utils.LavaMutex
	blocksQueue  []string // holds all past hashes up until latest block
}

func (cs *ChainSentry) GetLatestBlockNum() int64 {
	return atomic.LoadInt64(&cs.latestBlockNum)
}

func (cs *ChainSentry) SetLatestBlockNum(value int64) {
	atomic.StoreInt64(&cs.latestBlockNum, value)
}

func (cs *ChainSentry) GetLatestBlockData(requestedBlock int64) (latestBlock int64, hashesRes map[int64]interface{}, requestedBlockHash string, err error) {
	cs.blockQueueMu.Lock()
	defer cs.blockQueueMu.Unlock()

	latestBlockNum := cs.GetLatestBlockNum()
	if len(cs.blocksQueue) == 0 {
		return latestBlockNum, nil, "", utils.LavaFormatError("chainSentry GetLatestBlockData had no blocks", nil, &map[string]string{"latestBlock": strconv.FormatInt(latestBlockNum, 10)})
	}
	if len(cs.blocksQueue) < cs.numFinalBlocks {
		return latestBlockNum, nil, "", utils.LavaFormatError("chainSentry GetLatestBlockData had too little blocks in queue", nil, &map[string]string{"numFinalBlocks": strconv.FormatInt(int64(cs.numFinalBlocks), 10), "blocksQueueLen": strconv.FormatInt(int64(len(cs.blocksQueue)), 10)})
	}
	if requestedBlock < 0 {
		requestedBlock = sentry.ReplaceRequestedBlock(requestedBlock, latestBlockNum)
	}
	hashes := make(map[int64]interface{}, len(cs.blocksQueue))

	for indexInQueue := 0; indexInQueue < cs.numFinalBlocks; indexInQueue++ {
		blockNum := latestBlockNum - int64(cs.finalizedBlockDistance) - int64(cs.numFinalBlocks) + int64(indexInQueue+1)
		if blockNum < 0 {
			continue
		}
		if indexInQueue < cs.numFinalBlocks {
			// only return numFinalBlocks in the finalization guarantee
			hashes[blockNum] = cs.blocksQueue[indexInQueue]
		}
		// keep iterating on the others to find a match for the request
		if blockNum == requestedBlock {
			requestedBlockHash = cs.blocksQueue[indexInQueue]
		}
	}
	return latestBlockNum, hashes, requestedBlockHash, nil
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
		return ErrorFailedToFetchLatestBlock.Wrapf("Chain Sentry Init failed, additional info: " + err.Error())
	}

	err = cs.fetchAllPreviousBlocks(ctx, latestBlock)
	if err != nil {
		return ErrorFailedToFetchLatestBlock.Wrapf("Chain Sentry Init failed to fetch all blocks, additional info: " + err.Error())
	}
	return nil
}

func (cs *ChainSentry) fetchAllPreviousBlocks(ctx context.Context, latestBlock int64) error {
	tmpArr := []string{}
	for i := latestBlock - int64(cs.finalizedBlockDistance+cs.numFinalBlocks) + 1; i <= latestBlock; i++ { // save all blocks from the past up until latest block
		result, err := cs.fetchBlockHashByNum(ctx, i)
		if err != nil {
			utils.LavaFormatError("could not get block data in chainSentry", err, &map[string]string{"block": strconv.FormatInt(i, 10)})
			return err
		}

		utils.LavaFormatDebug("ChainSentry read a block", &map[string]string{"block": strconv.FormatInt(i, 10), "result": result})
		tmpArr = append(tmpArr, result) // save entire block data for now
	}
	cs.blockQueueMu.Lock()
	cs.SetLatestBlockNum(latestBlock)
	cs.blocksQueue = tmpArr
	blocksQueueLen := int64(len(cs.blocksQueue))
	cs.blockQueueMu.Unlock()
	utils.LavaFormatInfo("ChainSentry Updated latest block", &map[string]string{"block": strconv.FormatInt(latestBlock, 10), "latestHash": cs.GetLatestBlockHash(), "blocksQueueLen": strconv.FormatInt(blocksQueueLen, 10)})
	return nil
}

func (cs *ChainSentry) forkChangedOrGotNewBlock(ctx context.Context, latestBlock int64) (bool, error) {
	if cs.latestBlockNum != latestBlock {
		return true, nil
	}
	blockHash, err := cs.fetchBlockHashByNum(ctx, latestBlock)
	if err != nil {
		return true, utils.LavaFormatError("ChainSentry fetchBlockHashByNum failed", err, &map[string]string{"block": strconv.FormatInt(latestBlock, 10)})
	}
	return cs.GetLatestBlockHash() != blockHash, nil
}

func (cs *ChainSentry) catchupOnFinalizedBlocks(ctx context.Context) error {
	latestBlock, err := cs.fetchLatestBlockNum(ctx) // get actual latest from chain
	if err != nil {
		return utils.LavaFormatError("error getting latestBlockNum on catchup", err, nil)
	}
	shouldFetchAgain, err := cs.forkChangedOrGotNewBlock(ctx, latestBlock)
	if shouldFetchAgain || err != nil {
		err := cs.fetchAllPreviousBlocks(ctx, latestBlock)
		if err != nil {
			return utils.LavaFormatError("error getting all previous blocks on catchup", err, nil)
		}
	} else {
		utils.LavaFormatDebug("chainSentry skipped reading blocks because its up to date", &map[string]string{"latestHash": cs.GetLatestBlockHash()})
	}
	return nil
}

func (cs *ChainSentry) GetLatestBlockHash() string {
	cs.blockQueueMu.Lock()
	latestHash := cs.blocksQueue[len(cs.blocksQueue)-1]
	cs.blockQueueMu.Unlock()
	return latestHash
}

func (cs *ChainSentry) Start(ctx context.Context) error {
	// how often to query latest block.
	ticker := time.NewTicker(
		time.Millisecond * time.Duration(int64(cs.chainProxy.GetSentry().GetAverageBlockTime())))

	// Polls blocks and keeps a queue of them
	go func() {
		for {
			select {
			case <-ticker.C:
				err := cs.catchupOnFinalizedBlocks(ctx)
				if err != nil {
					log.Println(err)
				}
			case <-cs.quit:
				ticker.Stop()
				return
			}
		}
	}()

	return nil
}

func (cs *ChainSentry) quitSentry(ctx context.Context) {
	cs.quit <- true
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
		quit:                   make(chan bool),
	}
}
