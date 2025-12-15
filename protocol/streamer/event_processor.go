package streamer

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/lavanet/lava/v5/utils"
)

// EventProcessor processes blocks and emits stream events
type EventProcessor struct {
	config          *StreamerConfig
	rpcClient       *RPCClient
	abiDecoder      *ABIDecoder
	subscriptionMgr *SubscriptionManager
	websocketServer *WebSocketServer
	webhookSender   *WebhookSender
	messageQueue    *MessageQueueSender
	metrics         *StreamerMetrics

	currentBlock atomic.Int64
	mu           sync.RWMutex
	stopChan     chan struct{}
	wg           sync.WaitGroup
}

// NewEventProcessor creates a new event processor
func NewEventProcessor(
	config *StreamerConfig,
	rpcClient *RPCClient,
	abiDecoder *ABIDecoder,
	subMgr *SubscriptionManager,
	wsServer *WebSocketServer,
	whSender *WebhookSender,
	mqSender *MessageQueueSender,
	metrics *StreamerMetrics,
) *EventProcessor {
	return &EventProcessor{
		config:          config,
		rpcClient:       rpcClient,
		abiDecoder:      abiDecoder,
		subscriptionMgr: subMgr,
		websocketServer: wsServer,
		webhookSender:   whSender,
		messageQueue:    mqSender,
		metrics:         metrics,
		stopChan:        make(chan struct{}),
	}
}

// Start begins processing blocks
func (ep *EventProcessor) Start(ctx context.Context) error {
	// Determine starting block
	startBlock, err := ep.determineStartBlock(ctx)
	if err != nil {
		return fmt.Errorf("failed to determine start block: %w", err)
	}

	ep.currentBlock.Store(startBlock)

	utils.LavaFormatInfo("Event processor starting",
		utils.LogAttr("chainID", ep.config.ChainID),
		utils.LogAttr("startBlock", startBlock),
	)

	// Start processing
	ep.wg.Add(1)
	go ep.processBlocks(ctx)

	// Start metrics reporter
	ep.wg.Add(1)
	go ep.reportMetrics(ctx)

	return nil
}

// Stop stops the event processor
func (ep *EventProcessor) Stop() {
	close(ep.stopChan)
	ep.wg.Wait()
	utils.LavaFormatInfo("Event processor stopped")
}

// determineStartBlock determines which block to start streaming from
func (ep *EventProcessor) determineStartBlock(ctx context.Context) (int64, error) {
	if ep.config.StartBlock > 0 {
		return ep.config.StartBlock, nil
	}

	// Start from latest block
	latestBlock, err := ep.rpcClient.GetBlockNumber(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest block: %w", err)
	}

	utils.LavaFormatInfo("Starting from latest block", utils.LogAttr("block", latestBlock))
	return latestBlock, nil
}

// processBlocks continuously processes new blocks
func (ep *EventProcessor) processBlocks(ctx context.Context) {
	defer ep.wg.Done()

	ticker := time.NewTicker(ep.config.PollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ep.stopChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := ep.processNextBlock(ctx); err != nil {
				utils.LavaFormatError("Failed to process block", err)
			}
		}
	}
}

// processNextBlock processes the next block
func (ep *EventProcessor) processNextBlock(ctx context.Context) error {
	// Get latest block
	latestBlock, err := ep.rpcClient.GetBlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest block: %w", err)
	}

	currentBlock := ep.currentBlock.Load()

	// Account for confirmation blocks
	targetBlock := latestBlock - int64(ep.config.ConfirmationBlocks)
	if currentBlock > targetBlock {
		return nil // Caught up
	}

	// Process block
	if err := ep.processBlock(ctx, currentBlock); err != nil {
		return fmt.Errorf("failed to process block %d: %w", currentBlock, err)
	}

	// Move to next block
	ep.currentBlock.Add(1)
	ep.metrics.LastBlockProcessed = currentBlock
	ep.metrics.BlocksProcessed++

	return nil
}

// processBlock processes a single block
func (ep *EventProcessor) processBlock(ctx context.Context, blockNum int64) error {
	// Fetch block
	evmBlock, err := ep.rpcClient.GetBlockByNumber(ctx, blockNum, ep.config.StreamTransactions)
	if err != nil {
		return err
	}

	if evmBlock == nil {
		return fmt.Errorf("block %d not found", blockNum)
	}

	// Emit new block event
	ep.emitBlockEvent(evmBlock)

	// Process transactions if enabled
	if ep.config.StreamTransactions && len(evmBlock.Transactions) > 0 {
		if err := ep.processTransactions(ctx, evmBlock); err != nil {
			utils.LavaFormatWarning("Failed to process transactions", err)
		}
	}

	// Process logs if enabled
	if ep.config.StreamLogs {
		if err := ep.processLogs(ctx, blockNum); err != nil {
			utils.LavaFormatWarning("Failed to process logs", err)
		}
	}

	return nil
}

// emitBlockEvent emits a new block event
func (ep *EventProcessor) emitBlockEvent(evmBlock *EVMBlock) {
	blockNum, _ := parseHexToInt64(evmBlock.Number)
	timestamp, _ := parseHexToInt64(evmBlock.Timestamp)
	gasUsed, _ := parseHexToInt64(evmBlock.GasUsed)
	gasLimit, _ := parseHexToInt64(evmBlock.GasLimit)

	block := &Block{
		ChainID:          ep.config.ChainID,
		BlockNumber:      blockNum,
		BlockHash:        evmBlock.Hash,
		ParentHash:       evmBlock.ParentHash,
		Timestamp:        timestamp,
		Miner:            evmBlock.Miner,
		GasUsed:          gasUsed,
		GasLimit:         gasLimit,
		BaseFeePerGas:    evmBlock.BaseFeePerGas,
		TransactionCount: len(evmBlock.Transactions),
	}

	event := &StreamEvent{
		ID:          uuid.New().String(),
		Type:        EventTypeNewBlock,
		ChainID:     ep.config.ChainID,
		BlockNumber: blockNum,
		BlockHash:   evmBlock.Hash,
		Timestamp:   timestamp,
		Data:        map[string]interface{}{"block": block},
		EmittedAt:   time.Now(),
	}

	ep.emitEvent(event)
}

// processTransactions processes transactions in a block
func (ep *EventProcessor) processTransactions(ctx context.Context, evmBlock *EVMBlock) error {
	for _, txInterface := range evmBlock.Transactions {
		if err := ep.processTransaction(ctx, txInterface, evmBlock); err != nil {
			utils.LavaFormatWarning("Failed to process transaction", err)
		}
	}
	return nil
}

// processTransaction processes a single transaction
func (ep *EventProcessor) processTransaction(ctx context.Context, txInterface interface{}, evmBlock *EVMBlock) error {
	var txHash string

	// Extract transaction hash
	switch tx := txInterface.(type) {
	case string:
		txHash = tx
	case map[string]interface{}:
		if hash, ok := tx["hash"].(string); ok {
			txHash = hash
		}
	default:
		return nil
	}

	// Fetch receipt
	receipt, err := ep.rpcClient.GetTransactionReceipt(ctx, txHash)
	if err != nil {
		return err
	}

	if receipt == nil {
		return nil
	}

	// Convert to Transaction
	tx := ep.convertTransaction(receipt, evmBlock.Hash)

	// Emit transaction event
	event := &StreamEvent{
		ID:              uuid.New().String(),
		Type:            EventTypeTransaction,
		ChainID:         ep.config.ChainID,
		BlockNumber:     tx.BlockNumber,
		BlockHash:       tx.BlockHash,
		TransactionHash: tx.TxHash,
		Timestamp:       0,
		Data:            map[string]interface{}{"transaction": tx},
		EmittedAt:       time.Now(),
	}

	ep.emitEvent(event)

	// Emit contract deployment event if applicable
	if tx.ContractAddress != nil {
		deployEvent := &StreamEvent{
			ID:              uuid.New().String(),
			Type:            EventTypeContractDeployment,
			ChainID:         ep.config.ChainID,
			BlockNumber:     tx.BlockNumber,
			BlockHash:       tx.BlockHash,
			TransactionHash: tx.TxHash,
			Data: map[string]interface{}{
				"contract_address": *tx.ContractAddress,
				"deployer":         tx.FromAddress,
			},
			EmittedAt: time.Now(),
		}
		ep.emitEvent(deployEvent)
	}

	// Process internal transactions if enabled
	if ep.config.StreamInternalTxs {
		ep.processInternalTransactions(ctx, txHash, tx.BlockNumber)
	}

	return nil
}

// processInternalTransactions processes internal transactions
func (ep *EventProcessor) processInternalTransactions(ctx context.Context, txHash string, blockNumber int64) {
	internalTxs, err := ep.rpcClient.GetInternalTransactions(ctx, txHash)
	if err != nil {
		utils.LavaFormatDebug("Failed to get internal transactions", utils.LogAttr("error", err))
		return
	}

	for _, itx := range internalTxs {
		itx.BlockNumber = blockNumber

		event := &StreamEvent{
			ID:              uuid.New().String(),
			Type:            EventTypeInternalTx,
			ChainID:         ep.config.ChainID,
			BlockNumber:     blockNumber,
			TransactionHash: txHash,
			Data:            map[string]interface{}{"internal_transaction": itx},
			EmittedAt:       time.Now(),
		}

		ep.emitEvent(event)
	}
}

// processLogs processes event logs for a block
func (ep *EventProcessor) processLogs(ctx context.Context, blockNum int64) error {
	blockHex := fmt.Sprintf("0x%x", blockNum)

	filter := LogFilter{
		FromBlock: blockHex,
		ToBlock:   blockHex,
	}

	evmLogs, err := ep.rpcClient.GetLogs(ctx, filter)
	if err != nil {
		return err
	}

	for _, evmLog := range evmLogs {
		ep.processLog(&evmLog)
	}

	return nil
}

// processLog processes a single log
func (ep *EventProcessor) processLog(evmLog *EVMLog) {
	log := ep.convertLog(evmLog)

	// Emit raw log event
	event := &StreamEvent{
		ID:              uuid.New().String(),
		Type:            EventTypeLog,
		ChainID:         ep.config.ChainID,
		BlockNumber:     log.BlockNumber,
		BlockHash:       log.BlockHash,
		TransactionHash: log.TransactionHash,
		Data:            map[string]interface{}{"log": log},
		EmittedAt:       time.Now(),
	}

	ep.emitEvent(event)

	// Decode event if enabled
	if ep.config.DecodeEvents && ep.abiDecoder != nil {
		if decoded, err := ep.abiDecoder.DecodeLog(log); err == nil && decoded != nil {
			ep.emitDecodedEvent(decoded, log)
		}
	}
}

// emitDecodedEvent emits a decoded event
func (ep *EventProcessor) emitDecodedEvent(decoded *DecodedEvent, log *Log) {
	eventType := EventTypeDecodedEvent

	// Map to specific event types
	switch decoded.EventType {
	case "transfer":
		eventType = EventTypeTokenTransfer
	case "approval":
		eventType = EventTypeTokenApproval
	}

	event := &StreamEvent{
		ID:              uuid.New().String(),
		Type:            eventType,
		ChainID:         ep.config.ChainID,
		BlockNumber:     log.BlockNumber,
		BlockHash:       log.BlockHash,
		TransactionHash: log.TransactionHash,
		Data: map[string]interface{}{
			"decoded_event":    decoded,
			"contract_address": log.Address,
			"event_type":       decoded.EventType,
			"event_name":       decoded.EventName,
			"parameters":       decoded.Parameters,
		},
		EmittedAt: time.Now(),
	}

	ep.emitEvent(event)
}

// emitEvent emits an event to all delivery channels
func (ep *EventProcessor) emitEvent(event *StreamEvent) {
	// Update metrics
	ep.metrics.EventsEmitted++

	// 1. Send to WebSocket clients (real-time subscriptions)
	if ep.websocketServer != nil {
		ep.websocketServer.SendEvent(event)
	}

	// 2. Send to webhook subscribers (HTTP callbacks)
	if ep.webhookSender != nil {
		matches := ep.subscriptionMgr.MatchEvent(event)
		for _, sub := range matches {
			if sub.Webhook != nil {
				ep.webhookSender.SendEvent(event, sub)
			}
		}
	}

	// 3. Send to message queue (Kafka/RabbitMQ/Redis)
	if ep.messageQueue != nil {
		ep.messageQueue.SendEvent(event)
	}
}

// reportMetrics periodically reports metrics
func (ep *EventProcessor) reportMetrics(ctx context.Context) {
	defer ep.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ep.stopChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			ep.metrics.ActiveSubscriptions = int64(ep.subscriptionMgr.GetActiveCount())
			if ep.websocketServer != nil {
				ep.metrics.WebSocketConnections = int64(ep.websocketServer.GetConnectionCount())
			}

			utils.LavaFormatInfo("Streaming metrics",
				utils.LogAttr("currentBlock", ep.currentBlock.Load()),
				utils.LogAttr("blocksProcessed", ep.metrics.BlocksProcessed),
				utils.LogAttr("eventsEmitted", ep.metrics.EventsEmitted),
				utils.LogAttr("activeSubscriptions", ep.metrics.ActiveSubscriptions),
			)
		}
	}
}

// Helper methods

func (ep *EventProcessor) convertTransaction(receipt *EVMTransactionReceipt, blockHash string) *Transaction {
	blockNumber, _ := parseHexToInt64(receipt.BlockNumber)
	txIndex, _ := parseHexToInt64(receipt.TransactionIndex)
	gasUsed, _ := parseHexToInt64(receipt.GasUsed)

	return &Transaction{
		ChainID:          ep.config.ChainID,
		TxHash:           receipt.TransactionHash,
		BlockNumber:      blockNumber,
		BlockHash:        blockHash,
		TransactionIndex: int(txIndex),
		FromAddress:      receipt.From,
		ToAddress:        receipt.To,
		Status:           parseStatus(receipt.Status),
		GasUsed:          gasUsed,
		ContractAddress:  receipt.ContractAddress,
	}
}

func (ep *EventProcessor) convertLog(evmLog *EVMLog) *Log {
	blockNumber, _ := parseHexToInt64(evmLog.BlockNumber)
	txIndex, _ := parseHexToInt64(evmLog.TransactionIndex)
	logIndex, _ := parseHexToInt64(evmLog.LogIndex)

	log := &Log{
		ChainID:          ep.config.ChainID,
		BlockNumber:      blockNumber,
		BlockHash:        evmLog.BlockHash,
		TransactionHash:  evmLog.TransactionHash,
		TransactionIndex: int(txIndex),
		LogIndex:         int(logIndex),
		Address:          evmLog.Address,
		Data:             evmLog.Data,
		Removed:          evmLog.Removed,
	}

	if len(evmLog.Topics) > 0 {
		log.Topic0 = &evmLog.Topics[0]
	}
	if len(evmLog.Topics) > 1 {
		log.Topic1 = &evmLog.Topics[1]
	}
	if len(evmLog.Topics) > 2 {
		log.Topic2 = &evmLog.Topics[2]
	}
	if len(evmLog.Topics) > 3 {
		log.Topic3 = &evmLog.Topics[3]
	}

	return log
}

func parseHexToInt64(hexStr string) (int64, error) {
	if hexStr == "" || hexStr == "0x" {
		return 0, nil
	}
	var num int64
	_, err := fmt.Sscanf(hexStr, "0x%x", &num)
	return num, err
}

func parseStatus(statusHex string) int {
	if statusHex == "0x1" {
		return 1
	}
	return 0
}

