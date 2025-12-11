package indexer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lavanet/lava/v5/utils"
)

// ContractWatcher monitors specific smart contracts for events
type ContractWatcher struct {
	config    *IndexerConfig
	rpcClient *RPCClient
	storage   Storage
	metrics   *IndexerMetrics

	watches  []*ContractWatch
	mu       sync.RWMutex
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewContractWatcher creates a new contract watcher
func NewContractWatcher(config *IndexerConfig, rpcClient *RPCClient, storage Storage, metrics *IndexerMetrics) *ContractWatcher {
	return &ContractWatcher{
		config:    config,
		rpcClient: rpcClient,
		storage:   storage,
		metrics:   metrics,
		watches:   make([]*ContractWatch, 0),
		stopChan:  make(chan struct{}),
	}
}

// Start begins watching contracts
func (cw *ContractWatcher) Start(ctx context.Context) error {
	// Load configured contract watches
	if err := cw.loadWatches(ctx); err != nil {
		return fmt.Errorf("failed to load watches: %w", err)
	}

	if len(cw.watches) == 0 {
		utils.LavaFormatInfo("No contracts to watch, contract watcher disabled")
		return nil
	}

	utils.LavaFormatInfo("Contract watcher starting",
		utils.LogAttr("watchCount", len(cw.watches)),
	)

	// Start watcher goroutine
	cw.wg.Add(1)
	go cw.watcher(ctx)

	return nil
}

// Stop stops the contract watcher
func (cw *ContractWatcher) Stop() {
	close(cw.stopChan)
	cw.wg.Wait()
	utils.LavaFormatInfo("Contract watcher stopped")
}

// loadWatches loads contract watches from config and database
func (cw *ContractWatcher) loadWatches(ctx context.Context) error {
	// Load from database
	dbWatches, err := cw.storage.GetContractWatches(ctx, cw.config.ChainID, true)
	if err != nil {
		return err
	}

	cw.mu.Lock()
	defer cw.mu.Unlock()

	// Merge with configured watches
	for _, configWatch := range cw.config.WatchedContracts {
		// Check if already in database
		found := false
		for _, dbWatch := range dbWatches {
			if dbWatch.Address == configWatch.Address {
				found = true
				cw.watches = append(cw.watches, dbWatch)
				break
			}
		}

		if !found {
			// Create new watch
			watch := &ContractWatch{
				ChainID:     cw.config.ChainID,
				Address:     configWatch.Address,
				Name:        configWatch.Name,
				StartBlock:  configWatch.StartBlock,
				EventFilter: configWatch.EventFilter,
				Active:      true,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}

			if err := cw.storage.SaveContractWatch(ctx, watch); err != nil {
				utils.LavaFormatWarning("Failed to save contract watch", err,
					utils.LogAttr("address", watch.Address),
				)
				continue
			}

			cw.watches = append(cw.watches, watch)
			utils.LavaFormatInfo("Added contract watch",
				utils.LogAttr("name", watch.Name),
				utils.LogAttr("address", watch.Address),
			)
		}
	}

	return nil
}

// watcher continuously monitors contracts for new events
func (cw *ContractWatcher) watcher(ctx context.Context) {
	defer cw.wg.Done()

	ticker := time.NewTicker(cw.config.PollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cw.stopChan:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := cw.checkContracts(ctx); err != nil {
				utils.LavaFormatError("Failed to check contracts", err)
				cw.metrics.IncIndexErrors()
			}
		}
	}
}

// checkContracts checks all watched contracts for new events
func (cw *ContractWatcher) checkContracts(ctx context.Context) error {
	cw.mu.RLock()
	watches := make([]*ContractWatch, len(cw.watches))
	copy(watches, cw.watches)
	cw.mu.RUnlock()

	// Get latest block
	latestBlock, err := cw.rpcClient.GetBlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest block: %w", err)
	}

	for _, watch := range watches {
		if err := cw.checkContract(ctx, watch, latestBlock); err != nil {
			utils.LavaFormatError("Failed to check contract", err,
				utils.LogAttr("contract", watch.Name),
				utils.LogAttr("address", watch.Address),
			)
		}
	}

	return nil
}

// checkContract checks a single contract for new events
func (cw *ContractWatcher) checkContract(ctx context.Context, watch *ContractWatch, latestBlock int64) error {
	// Determine from block
	fromBlock := watch.StartBlock

	// Get last indexed log for this contract
	lastLog, err := cw.getLastIndexedLog(ctx, watch.Address)
	if err != nil {
		return err
	}

	if lastLog != nil && lastLog.BlockNumber > fromBlock {
		fromBlock = lastLog.BlockNumber + 1
	}

	// Don't index too far ahead (use confirmation blocks)
	toBlock := latestBlock - int64(cw.config.ConfirmationBlocks)
	if fromBlock > toBlock {
		// Already caught up
		return nil
	}

	// Limit batch size
	if toBlock-fromBlock > int64(cw.config.BatchSize) {
		toBlock = fromBlock + int64(cw.config.BatchSize) - 1
	}

	utils.LavaFormatDebug("Checking contract for events",
		utils.LogAttr("contract", watch.Name),
		utils.LogAttr("from", fromBlock),
		utils.LogAttr("to", toBlock),
	)

	// Build filter
	filter := LogFilter{
		FromBlock: fmt.Sprintf("0x%x", fromBlock),
		ToBlock:   fmt.Sprintf("0x%x", toBlock),
		Address:   []string{watch.Address},
	}

	// Add topic filters if specified
	if len(watch.EventFilter) > 0 {
		topics := make([][]string, 1)
		topics[0] = watch.EventFilter
		filter.Topics = topics
	}

	// Fetch logs
	evmLogs, err := cw.rpcClient.GetLogs(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to fetch logs: %w", err)
	}

	if len(evmLogs) == 0 {
		return nil
	}

	utils.LavaFormatInfo("Found contract events",
		utils.LogAttr("contract", watch.Name),
		utils.LogAttr("count", len(evmLogs)),
		utils.LogAttr("fromBlock", fromBlock),
		utils.LogAttr("toBlock", toBlock),
	)

	// Convert and save logs
	logs := make([]*Log, 0, len(evmLogs))
	for _, evmLog := range evmLogs {
		log := cw.convertLog(&evmLog)
		logs = append(logs, log)
	}

	if err := cw.storage.SaveLogs(ctx, logs); err != nil {
		return fmt.Errorf("failed to save logs: %w", err)
	}

	cw.metrics.IncLogsIndexed()

	return nil
}

// getLastIndexedLog gets the last indexed log for a contract
func (cw *ContractWatcher) getLastIndexedLog(ctx context.Context, contractAddr string) (*Log, error) {
	query := &LogQuery{
		ChainID:         cw.config.ChainID,
		ContractAddress: &contractAddr,
		Limit:           1,
		OrderByDesc:     true,
	}

	logs, err := cw.storage.GetLogs(ctx, query)
	if err != nil {
		return nil, err
	}

	if len(logs) > 0 {
		return logs[0], nil
	}

	return nil, nil
}

// convertLog converts an EVM log to our internal Log type
func (cw *ContractWatcher) convertLog(evmLog *EVMLog) *Log {
	blockNumber, _ := parseHexToInt64(evmLog.BlockNumber)
	txIndex, _ := parseHexToInt64(evmLog.TransactionIndex)
	logIndex, _ := parseHexToInt64(evmLog.LogIndex)

	log := &Log{
		ChainID:          cw.config.ChainID,
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

	return log
}

// AddWatch adds a new contract to watch
func (cw *ContractWatcher) AddWatch(ctx context.Context, watch *ContractWatch) error {
	if err := cw.storage.SaveContractWatch(ctx, watch); err != nil {
		return err
	}

	cw.mu.Lock()
	cw.watches = append(cw.watches, watch)
	cw.mu.Unlock()

	utils.LavaFormatInfo("Added contract watch",
		utils.LogAttr("name", watch.Name),
		utils.LogAttr("address", watch.Address),
	)

	return nil
}

// GetWatches returns all active contract watches
func (cw *ContractWatcher) GetWatches() []*ContractWatch {
	cw.mu.RLock()
	defer cw.mu.RUnlock()

	watches := make([]*ContractWatch, len(cw.watches))
	copy(watches, cw.watches)
	return watches
}
