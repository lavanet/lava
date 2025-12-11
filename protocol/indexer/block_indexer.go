package indexer

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/lavanet/lava/v5/utils"
)

// BlockIndexer handles the indexing of blockchain blocks
type BlockIndexer struct {
	config     *IndexerConfig
	rpcClient  *RPCClient
	storage    Storage
	metrics    *IndexerMetrics
	abiDecoder *ABIDecoder

	// State
	currentBlock int64
	mu           sync.RWMutex
	stopChan     chan struct{}
	wg           sync.WaitGroup
}

// NewBlockIndexer creates a new block indexer
func NewBlockIndexer(config *IndexerConfig, rpcClient *RPCClient, storage Storage, metrics *IndexerMetrics, abiDecoder *ABIDecoder) *BlockIndexer {
	return &BlockIndexer{
		config:     config,
		rpcClient:  rpcClient,
		storage:    storage,
		metrics:    metrics,
		abiDecoder: abiDecoder,
		stopChan:   make(chan struct{}),
	}
}

// Start begins the block indexing process
func (bi *BlockIndexer) Start(ctx context.Context) error {
	// Determine starting block
	startBlock, err := bi.determineStartBlock(ctx)
	if err != nil {
		return fmt.Errorf("failed to determine start block: %w", err)
	}

	bi.mu.Lock()
	bi.currentBlock = startBlock
	bi.mu.Unlock()

	utils.LavaFormatInfo("Block indexer starting",
		utils.LogAttr("chainID", bi.config.ChainID),
		utils.LogAttr("startBlock", startBlock),
		utils.LogAttr("batchSize", bi.config.BatchSize),
		utils.LogAttr("concurrentWorkers", bi.config.ConcurrentWorkers),
	)

	// Start worker pool
	for i := 0; i < bi.config.ConcurrentWorkers; i++ {
		bi.wg.Add(1)
		go bi.worker(ctx, i)
	}

	// Start monitoring goroutine
	bi.wg.Add(1)
	go bi.monitor(ctx)

	return nil
}

// Stop stops the block indexer
func (bi *BlockIndexer) Stop() {
	close(bi.stopChan)
	bi.wg.Wait()
	utils.LavaFormatInfo("Block indexer stopped", utils.LogAttr("chainID", bi.config.ChainID))
}

// determineStartBlock determines which block to start indexing from
func (bi *BlockIndexer) determineStartBlock(ctx context.Context) (int64, error) {
	// Check if we have existing state in database
	state, err := bi.storage.GetIndexerState(ctx, bi.config.ChainID)
	if err != nil {
		return 0, fmt.Errorf("failed to get indexer state: %w", err)
	}

	if state != nil && state.LastBlockNumber > 0 {
		// Resume from last indexed block + 1
		utils.LavaFormatInfo("Resuming from previous state",
			utils.LogAttr("lastBlock", state.LastBlockNumber),
		)
		return state.LastBlockNumber + 1, nil
	}

	// No previous state, use configured start block
	if bi.config.StartBlock > 0 {
		return bi.config.StartBlock, nil
	}

	// Default to latest block
	latestBlock, err := bi.rpcClient.GetBlockNumber(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest block: %w", err)
	}

	utils.LavaFormatInfo("Starting from latest block", utils.LogAttr("block", latestBlock))
	return latestBlock, nil
}

// worker processes blocks concurrently
func (bi *BlockIndexer) worker(ctx context.Context, workerID int) {
	defer bi.wg.Done()

	utils.LavaFormatDebug("Worker started", utils.LogAttr("workerID", workerID))

	ticker := time.NewTicker(bi.config.PollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-bi.stopChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := bi.processNextBatch(ctx); err != nil {
				utils.LavaFormatError("Failed to process batch", err,
					utils.LogAttr("workerID", workerID),
				)
				bi.metrics.IncIndexErrors()
			}
		}
	}
}

// monitor logs indexing progress periodically
func (bi *BlockIndexer) monitor(ctx context.Context) {
	defer bi.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-bi.stopChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			bi.logProgress(ctx)
		}
	}
}

// processNextBatch processes a batch of blocks
func (bi *BlockIndexer) processNextBatch(ctx context.Context) error {
	// Get latest block from chain
	latestChainBlock, err := bi.rpcClient.GetBlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest block: %w", err)
	}

	bi.mu.Lock()
	currentBlock := bi.currentBlock
	bi.mu.Unlock()

	// Account for confirmation blocks
	targetBlock := latestChainBlock - int64(bi.config.ConfirmationBlocks)
	if currentBlock > targetBlock {
		// We're caught up, nothing to do
		return nil
	}

	// Determine batch end
	batchEnd := currentBlock + int64(bi.config.BatchSize) - 1
	if batchEnd > targetBlock {
		batchEnd = targetBlock
	}

	utils.LavaFormatDebug("Processing batch",
		utils.LogAttr("from", currentBlock),
		utils.LogAttr("to", batchEnd),
		utils.LogAttr("latest", latestChainBlock),
	)

	// Process blocks in batch
	for blockNum := currentBlock; blockNum <= batchEnd; blockNum++ {
		if err := bi.indexBlock(ctx, blockNum); err != nil {
			return fmt.Errorf("failed to index block %d: %w", blockNum, err)
		}

		bi.mu.Lock()
		bi.currentBlock = blockNum + 1
		bi.mu.Unlock()

		bi.metrics.SetLastIndexedBlock(blockNum)
	}

	// Update state
	lastBlock, err := bi.storage.GetBlock(ctx, bi.config.ChainID, batchEnd)
	if err != nil {
		return fmt.Errorf("failed to get last block: %w", err)
	}

	if lastBlock != nil {
		state := &IndexerState{
			ChainID:         bi.config.ChainID,
			LastBlockNumber: batchEnd,
			LastBlockHash:   lastBlock.BlockHash,
			UpdatedAt:       time.Now(),
		}
		if err := bi.storage.SaveIndexerState(ctx, state); err != nil {
			return fmt.Errorf("failed to save indexer state: %w", err)
		}
	}

	return nil
}

// indexBlock indexes a single block and its transactions/logs
func (bi *BlockIndexer) indexBlock(ctx context.Context, blockNum int64) error {
	startTime := time.Now()

	// Fetch block from RPC
	evmBlock, err := bi.rpcClient.GetBlockByNumber(ctx, blockNum, bi.config.IndexFullBlocks)
	if err != nil {
		return fmt.Errorf("failed to fetch block: %w", err)
	}

	if evmBlock == nil {
		return fmt.Errorf("block %d not found", blockNum)
	}

	// Convert and save block
	block, err := bi.convertBlock(evmBlock)
	if err != nil {
		return fmt.Errorf("failed to convert block: %w", err)
	}

	// Use transaction for atomic save
	txStorage, err := bi.storage.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer txStorage.Rollback()

	if err := txStorage.SaveBlock(ctx, block); err != nil {
		return fmt.Errorf("failed to save block: %w", err)
	}

	// Index transactions if enabled and available
	if bi.config.IndexTransactions && len(evmBlock.Transactions) > 0 {
		if err := bi.indexTransactions(ctx, txStorage, evmBlock); err != nil {
			return fmt.Errorf("failed to index transactions: %w", err)
		}
	}

	// Index logs if enabled
	if bi.config.IndexLogs {
		if err := bi.indexBlockLogs(ctx, txStorage, blockNum); err != nil {
			return fmt.Errorf("failed to index logs: %w", err)
		}
	}

	// Index internal transactions if enabled
	if bi.config.IndexInternalTxs && len(evmBlock.Transactions) > 0 {
		if err := bi.indexInternalTransactions(ctx, txStorage, evmBlock); err != nil {
			utils.LavaFormatWarning("Failed to index internal transactions", err,
				utils.LogAttr("block", blockNum),
			)
			// Don't fail the whole block if internal tx indexing fails
		}
	}

	if err := txStorage.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	duration := time.Since(startTime)
	bi.metrics.ObserveBlockIndexDuration(duration)
	bi.metrics.IncBlocksIndexed()

	utils.LavaFormatDebug("Block indexed",
		utils.LogAttr("block", blockNum),
		utils.LogAttr("hash", block.BlockHash[:10]+"..."),
		utils.LogAttr("txCount", block.TransactionCount),
		utils.LogAttr("duration", duration),
	)

	return nil
}

// convertBlock converts an EVM block to our internal Block type
func (bi *BlockIndexer) convertBlock(evmBlock *EVMBlock) (*Block, error) {
	blockNumber, err := parseHexToInt64(evmBlock.Number)
	if err != nil {
		return nil, fmt.Errorf("failed to parse block number: %w", err)
	}

	timestamp, err := parseHexToInt64(evmBlock.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	gasUsed, err := parseHexToInt64(evmBlock.GasUsed)
	if err != nil {
		return nil, fmt.Errorf("failed to parse gas used: %w", err)
	}

	gasLimit, err := parseHexToInt64(evmBlock.GasLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to parse gas limit: %w", err)
	}

	size, err := parseHexToInt64(evmBlock.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to parse size: %w", err)
	}

	return &Block{
		ChainID:          bi.config.ChainID,
		BlockNumber:      blockNumber,
		BlockHash:        evmBlock.Hash,
		ParentHash:       evmBlock.ParentHash,
		Timestamp:        timestamp,
		Miner:            evmBlock.Miner,
		GasUsed:          gasUsed,
		GasLimit:         gasLimit,
		BaseFeePerGas:    evmBlock.BaseFeePerGas,
		Difficulty:       evmBlock.Difficulty,
		TotalDifficulty:  evmBlock.TotalDifficulty,
		Size:             size,
		TransactionCount: len(evmBlock.Transactions),
		ExtraData:        evmBlock.ExtraData,
		IndexedAt:        time.Now(),
	}, nil
}

// indexTransactions indexes all transactions in a block
func (bi *BlockIndexer) indexTransactions(ctx context.Context, storage Storage, evmBlock *EVMBlock) error {
	if len(evmBlock.Transactions) == 0 {
		return nil
	}

	// Check if transactions are full objects or just hashes
	if _, ok := evmBlock.Transactions[0].(string); ok {
		// Just hashes, need to fetch full transaction data
		return bi.indexTransactionsByHash(ctx, storage, evmBlock)
	}

	// Full transaction objects
	return bi.indexFullTransactions(ctx, storage, evmBlock)
}

// indexFullTransactions indexes transactions when block contains full tx objects
func (bi *BlockIndexer) indexFullTransactions(ctx context.Context, storage Storage, evmBlock *EVMBlock) error {
	transactions := make([]*Transaction, 0, len(evmBlock.Transactions))

	for _, txInterface := range evmBlock.Transactions {
		txData, ok := txInterface.(map[string]interface{})
		if !ok {
			continue
		}

		tx, err := bi.convertTransactionFromMap(txData, evmBlock.Hash)
		if err != nil {
			utils.LavaFormatWarning("Failed to convert transaction", err)
			continue
		}

		// Fetch receipt for status and gas used
		receipt, err := bi.rpcClient.GetTransactionReceipt(ctx, tx.TxHash)
		if err != nil {
			utils.LavaFormatWarning("Failed to get transaction receipt", err,
				utils.LogAttr("txHash", tx.TxHash),
			)
			continue
		}

		if receipt != nil {
			tx.Status = parseStatus(receipt.Status)
			gasUsed, _ := parseHexToInt64(receipt.GasUsed)
			tx.GasUsed = gasUsed

			if receipt.ContractAddress != nil {
				tx.ContractAddress = receipt.ContractAddress
			}
		}

		transactions = append(transactions, tx)
	}

	if len(transactions) > 0 {
		return storage.SaveTransactions(ctx, transactions)
	}

	return nil
}

// indexTransactionsByHash indexes transactions when block contains only tx hashes
func (bi *BlockIndexer) indexTransactionsByHash(ctx context.Context, storage Storage, evmBlock *EVMBlock) error {
	transactions := make([]*Transaction, 0, len(evmBlock.Transactions))

	for _, txInterface := range evmBlock.Transactions {
		txHash, ok := txInterface.(string)
		if !ok {
			continue
		}

		receipt, err := bi.rpcClient.GetTransactionReceipt(ctx, txHash)
		if err != nil {
			utils.LavaFormatWarning("Failed to get transaction receipt", err,
				utils.LogAttr("txHash", txHash),
			)
			continue
		}

		if receipt == nil {
			continue
		}

		tx, err := bi.convertTransactionFromReceipt(receipt)
		if err != nil {
			utils.LavaFormatWarning("Failed to convert transaction from receipt", err)
			continue
		}

		transactions = append(transactions, tx)
	}

	if len(transactions) > 0 {
		return storage.SaveTransactions(ctx, transactions)
	}

	return nil
}

// indexBlockLogs indexes all logs for a block
func (bi *BlockIndexer) indexBlockLogs(ctx context.Context, storage Storage, blockNum int64) error {
	blockHex := fmt.Sprintf("0x%x", blockNum)

	// Fetch logs for this block
	filter := LogFilter{
		FromBlock: blockHex,
		ToBlock:   blockHex,
	}

	evmLogs, err := bi.rpcClient.GetLogs(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to fetch logs: %w", err)
	}

	if len(evmLogs) == 0 {
		return nil
	}

	logs := make([]*Log, 0, len(evmLogs))
	for _, evmLog := range evmLogs {
		log, err := bi.convertLog(&evmLog)
		if err != nil {
			utils.LavaFormatWarning("Failed to convert log", err)
			continue
		}
		logs = append(logs, log)

		// Decode event if enabled
		if bi.config.DecodeEvents && bi.abiDecoder != nil {
			decoded, err := bi.abiDecoder.DecodeLog(log)
			if err != nil {
				utils.LavaFormatDebug("Failed to decode log", utils.LogAttr("error", err))
			} else if decoded != nil && decoded.EventType != EventTypeUnknown {
				// Save decoded event
				if sqlStorage, ok := storage.(*SQLiteStorage); ok {
					if err := sqlStorage.SaveDecodedEvent(ctx, decoded); err != nil {
						utils.LavaFormatWarning("Failed to save decoded event", err)
					}
				}
			}
		}
	}

	if len(logs) > 0 {
		return storage.SaveLogs(ctx, logs)
	}

	return nil
}

// indexInternalTransactions indexes internal transactions for a block
func (bi *BlockIndexer) indexInternalTransactions(ctx context.Context, storage Storage, evmBlock *EVMBlock) error {
	allInternalTxs := make([]InternalTransaction, 0)

	// Get transaction hashes
	var txHashes []string
	for _, txInterface := range evmBlock.Transactions {
		switch tx := txInterface.(type) {
		case string:
			txHashes = append(txHashes, tx)
		case map[string]interface{}:
			if hash, ok := tx["hash"].(string); ok {
				txHashes = append(txHashes, hash)
			}
		}
	}

	// Fetch internal transactions for each transaction
	for _, txHash := range txHashes {
		internalTxs, err := bi.rpcClient.GetInternalTransactions(ctx, txHash)
		if err != nil {
			utils.LavaFormatDebug("Failed to get internal transactions",
				utils.LogAttr("txHash", txHash),
				utils.LogAttr("error", err),
			)
			continue
		}

		// Set block number for each internal tx
		blockNum, _ := parseHexToInt64(evmBlock.Number)
		for i := range internalTxs {
			internalTxs[i].BlockNumber = blockNum
		}

		allInternalTxs = append(allInternalTxs, internalTxs...)
	}

	if len(allInternalTxs) > 0 {
		if sqlStorage, ok := storage.(*SQLiteStorage); ok {
			return sqlStorage.SaveInternalTransactions(ctx, allInternalTxs)
		}
	}

	return nil
}

// convertTransactionFromMap converts transaction from map[string]interface{}
func (bi *BlockIndexer) convertTransactionFromMap(txData map[string]interface{}, blockHash string) (*Transaction, error) {
	getString := func(key string) string {
		if v, ok := txData[key].(string); ok {
			return v
		}
		return ""
	}

	blockNumber, _ := parseHexToInt64(getString("blockNumber"))
	txIndex, _ := parseHexToInt64(getString("transactionIndex"))
	gas, _ := parseHexToInt64(getString("gas"))
	nonce, _ := parseHexToInt64(getString("nonce"))

	tx := &Transaction{
		ChainID:          bi.config.ChainID,
		TxHash:           getString("hash"),
		BlockNumber:      blockNumber,
		BlockHash:        blockHash,
		TransactionIndex: int(txIndex),
		FromAddress:      getString("from"),
		Value:            getString("value"),
		Gas:              gas,
		GasPrice:         getString("gasPrice"),
		Input:            getString("input"),
		Nonce:            nonce,
		IndexedAt:        time.Now(),
	}

	// Handle optional fields
	if to := getString("to"); to != "" {
		tx.ToAddress = &to
	}
	if maxFee := getString("maxFeePerGas"); maxFee != "" {
		tx.MaxFeePerGas = &maxFee
	}
	if maxPriority := getString("maxPriorityFeePerGas"); maxPriority != "" {
		tx.MaxPriorityFee = &maxPriority
	}

	return tx, nil
}

// convertTransactionFromReceipt converts transaction from receipt
func (bi *BlockIndexer) convertTransactionFromReceipt(receipt *EVMTransactionReceipt) (*Transaction, error) {
	blockNumber, _ := parseHexToInt64(receipt.BlockNumber)
	txIndex, _ := parseHexToInt64(receipt.TransactionIndex)
	gasUsed, _ := parseHexToInt64(receipt.GasUsed)

	tx := &Transaction{
		ChainID:          bi.config.ChainID,
		TxHash:           receipt.TransactionHash,
		BlockNumber:      blockNumber,
		BlockHash:        receipt.BlockHash,
		TransactionIndex: int(txIndex),
		FromAddress:      receipt.From,
		Status:           parseStatus(receipt.Status),
		GasUsed:          gasUsed,
		IndexedAt:        time.Now(),
	}

	if receipt.To != nil {
		tx.ToAddress = receipt.To
	}
	if receipt.ContractAddress != nil {
		tx.ContractAddress = receipt.ContractAddress
	}

	return tx, nil
}

// convertLog converts an EVM log to our internal Log type
func (bi *BlockIndexer) convertLog(evmLog *EVMLog) (*Log, error) {
	blockNumber, _ := parseHexToInt64(evmLog.BlockNumber)
	txIndex, _ := parseHexToInt64(evmLog.TransactionIndex)
	logIndex, _ := parseHexToInt64(evmLog.LogIndex)

	log := &Log{
		ChainID:          bi.config.ChainID,
		BlockNumber:      blockNumber,
		BlockHash:        evmLog.BlockHash,
		TransactionHash:  evmLog.TransactionHash,
		TransactionIndex: int(txIndex),
		LogIndex:         int(logIndex),
		Address:          evmLog.Address,
		Data:             evmLog.Data,
		Removed:          evmLog.Removed,
		IndexedAt:        time.Now(),
	}

	// Set topics
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

	return log, nil
}

// logProgress logs the current indexing progress
func (bi *BlockIndexer) logProgress(ctx context.Context) {
	bi.mu.RLock()
	currentBlock := bi.currentBlock
	bi.mu.RUnlock()

	latestBlock, err := bi.rpcClient.GetBlockNumber(ctx)
	if err != nil {
		utils.LavaFormatWarning("Failed to get latest block number", err)
		return
	}

	blocksBehind := latestBlock - currentBlock
	progress := float64(currentBlock) / float64(latestBlock) * 100

	utils.LavaFormatInfo("Indexing progress",
		utils.LogAttr("current", currentBlock),
		utils.LogAttr("latest", latestBlock),
		utils.LogAttr("behind", blocksBehind),
		utils.LogAttr("progress", fmt.Sprintf("%.2f%%", progress)),
	)
}

// GetCurrentBlock returns the current block being indexed
func (bi *BlockIndexer) GetCurrentBlock() int64 {
	bi.mu.RLock()
	defer bi.mu.RUnlock()
	return bi.currentBlock
}

// Helper functions

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
		return 1 // Success
	}
	return 0 // Failure
}

// Helper to parse hex string to int
func hexToInt(hexStr string) int {
	num, _ := strconv.ParseInt(hexStr, 0, 64)
	return int(num)
}
