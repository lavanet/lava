package chainsentry

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/lavanet/lava/relayer/chainproxy"
)

const (
	DEFAULT_NUM_FINAL_BLOCKS = 3
	DEFAULT_NUM_SAVED_BLOCKS = 7
)

type ChainSentry struct {
	chainProxy     chainproxy.ChainProxy
	numSavedBlocks int
	numFinalBlocks int
	latestBlockNum int64 // uint?
	ChainID        string

	// Spec blockQueueMu (rw mutex)
	blockQueueMu sync.RWMutex
	blocksQueue  []map[string]interface{}
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
	return cs.latestBlockNum // Lock mutex?
}

func (cs *ChainSentry) GetLatestBlockHashes() []map[string]interface{} {
	cs.blockQueueMu.Lock()
	defer cs.blockQueueMu.Unlock()

	var hashes = make([]map[string]interface{}, len(cs.blocksQueue))
	for i := 0; i < len(cs.blocksQueue); i++ {
		item := make(map[string]interface{})
		item["blockNum"] = cs.latestBlockNum - int64(len(cs.blocksQueue)+i)
		item["blockHash"] = cs.blocksQueue[i]["hash"]
		hashes[i] = item
	}
	return hashes
}

func (cs *ChainSentry) fetchLatestBlockNum(ctx context.Context) (int64, error) {
	blockNumMsg, err := cs.chainProxy.ParseMsg("", []byte(`{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`))
	if err != nil {
		return -1, err
	}
	blockNumReply, err := blockNumMsg.Send(ctx)
	if err != nil {
		return -1, err
	}

	var msg jsonrpcMessage
	err = json.Unmarshal(blockNumReply.GetData(), &msg)
	if err != nil {
		return -1, err
	}

	latestBlockstr, err := strconv.Unquote(string(msg.Result))
	latestBlockstr = strings.Replace(latestBlockstr, "0x", "", -1)

	// latestBlock := new(big.Int) // should we use bigint for this?
	// latestBlock.SetString(latestBlockstr, 16)
	latestBlock, err := strconv.ParseInt(latestBlockstr, 16, 64)
	// cs.LatestBlock = latestBlock
	return latestBlock, nil
}

func (cs *ChainSentry) getBlockByNum(ctx context.Context, blockNum int64) (map[string]interface{}, error) {
	messageTemplate := `{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x%x", true],"id":1}` // move to parser
	message := fmt.Sprintf(messageTemplate, blockNum)

	blockMsg, err := cs.chainProxy.ParseMsg("", []byte(message))
	if err != nil {
		log.Fatalln("error: Start", err)
		return nil, err
	}
	blockMsgReply, err := blockMsg.Send(ctx)
	if err != nil {
		log.Fatalln("error: Start", err)
		return nil, err
	}

	var msg jsonrpcMessage
	err = json.Unmarshal(blockMsgReply.GetData(), &msg)
	var result map[string]interface{}
	err = json.Unmarshal(msg.Result, &result)
	if err != nil {
		log.Fatalln("error: Start", err)
		return nil, err
	}
	return result, nil
}

func (cs *ChainSentry) Start(ctx context.Context) error {

	latestBlock, err := cs.fetchLatestBlockNum(ctx)
	if err != nil {
		log.Fatalln("error: Start", err)
		return err
	}

	cs.latestBlockNum = latestBlock
	log.Printf("latest %v block %v", cs.ChainID, latestBlock)
	// log.Printf("latest %v block %x", cs.ChainID, *latestBlock)
	// cs.blocksQueue = make([]map[string]interface{}, cs.numSavedBlocks+cs.numFinalBlocks)

	for i := cs.latestBlockNum - int64(cs.numSavedBlocks+cs.numFinalBlocks-1); i <= cs.latestBlockNum; i++ {
		result, err := cs.getBlockByNum(ctx, i)
		if err != nil {
			log.Fatalln("error: Start", err)
			return err
		}

		log.Printf("Block number: %v, block hash: %s", i, result["hash"])
		cs.blocksQueue = append(cs.blocksQueue, result) //save entire block data for now
	}
	log.Printf("%s", cs.GetLatestBlockHashes())

	ticker := time.NewTicker(time.Second * 5)
	quit := make(chan struct{})

	// Polls blocks and keeps a queue of them
	go func() {
		for {
			select {
			case <-ticker.C:
				latestBlock, err := cs.fetchLatestBlockNum(ctx)
				if err != nil {
					log.Printf("error: chainSentry block fetcher", err)
				}

				if cs.latestBlockNum != latestBlock {
					// Get all missing blocks
					for i := cs.latestBlockNum + 1; i <= latestBlock; i++ {
						blockData, err := cs.getBlockByNum(ctx, i)
						if err != nil {
							log.Fatalln("error: Start", err)
							return
						}

						cs.blockQueueMu.Lock()
						defer cs.blockQueueMu.Unlock()
						cs.blocksQueue = append(cs.blocksQueue, blockData) //save entire block data for now
						cs.latestBlockNum = latestBlock
					}
					cs.blocksQueue = cs.blocksQueue[(latestBlock - cs.latestBlockNum):]
					log.Printf("chainSentry blocks list updated %v", cs.blocksQueue)

				}
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
		chainProxy:     cp,
		ChainID:        chainID,
		numFinalBlocks: DEFAULT_NUM_FINAL_BLOCKS,
		numSavedBlocks: DEFAULT_NUM_SAVED_BLOCKS,
	}
}
