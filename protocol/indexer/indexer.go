package indexer

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/lavanet/lava/v5/utils"
)

// Indexer is the main indexing service
type Indexer struct {
	config          *IndexerConfig
	rpcClient       *RPCClient
	storage         Storage
	metrics         *IndexerMetrics
	abiDecoder      *ABIDecoder
	blockIndexer    *BlockIndexer
	contractWatcher *ContractWatcher
	apiServer       *APIServer
}

// NewIndexer creates a new indexer instance
func NewIndexer(config *IndexerConfig) (*Indexer, error) {
	// Validate config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Initialize RPC client
	rpcClient := NewRPCClient(config.RPCEndpoint, config.ChainID, config.MaxRetries, config.RetryDelay)

	// Initialize storage
	storage, err := NewStorage(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Initialize metrics
	metrics := NewIndexerMetrics()

	// Initialize ABI decoder
	abiDecoder := NewABIDecoder()

	// Register ABIs for watched contracts
	for _, contract := range config.WatchedContracts {
		if contract.ABI != "" {
			if err := abiDecoder.RegisterABI(contract.Address, contract.ABI); err != nil {
				return nil, fmt.Errorf("failed to register ABI for %s: %w", contract.Name, err)
			}
		}
	}

	// Initialize block indexer
	blockIndexer := NewBlockIndexer(config, rpcClient, storage, metrics, abiDecoder)

	// Initialize contract watcher
	contractWatcher := NewContractWatcher(config, rpcClient, storage, metrics)

	// Initialize API server
	apiServer := NewAPIServer(config, storage, metrics)

	return &Indexer{
		config:          config,
		rpcClient:       rpcClient,
		storage:         storage,
		metrics:         metrics,
		abiDecoder:      abiDecoder,
		blockIndexer:    blockIndexer,
		contractWatcher: contractWatcher,
		apiServer:       apiServer,
	}, nil
}

// Start starts all indexer components
func (idx *Indexer) Start(ctx context.Context) error {
	utils.LavaFormatInfo("Starting Lava EVM Indexer",
		utils.LogAttr("chainID", idx.config.ChainID),
		utils.LogAttr("rpcEndpoint", idx.config.RPCEndpoint),
		utils.LogAttr("databaseType", idx.config.DatabaseType),
	)

	// Test RPC connection
	blockNum, err := idx.rpcClient.GetBlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to RPC endpoint: %w", err)
	}

	utils.LavaFormatInfo("Connected to RPC endpoint",
		utils.LogAttr("latestBlock", blockNum),
	)

	// Start block indexer
	if err := idx.blockIndexer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start block indexer: %w", err)
	}

	// Start contract watcher
	if err := idx.contractWatcher.Start(ctx); err != nil {
		return fmt.Errorf("failed to start contract watcher: %w", err)
	}

	// Start API server
	if err := idx.apiServer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start API server: %w", err)
	}

	utils.LavaFormatInfo("Indexer started successfully")

	// Wait for shutdown signal
	idx.waitForShutdown(ctx)

	return nil
}

// Stop stops all indexer components
func (idx *Indexer) Stop(ctx context.Context) error {
	utils.LavaFormatInfo("Stopping indexer...")

	// Stop components in reverse order
	if err := idx.apiServer.Stop(ctx); err != nil {
		utils.LavaFormatError("Failed to stop API server", err)
	}

	idx.contractWatcher.Stop()
	idx.blockIndexer.Stop()

	// Close storage
	if err := idx.storage.Close(); err != nil {
		utils.LavaFormatError("Failed to close storage", err)
	}

	utils.LavaFormatInfo("Indexer stopped")
	return nil
}

// waitForShutdown waits for interrupt signal
func (idx *Indexer) waitForShutdown(ctx context.Context) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		utils.LavaFormatInfo("Received shutdown signal", utils.LogAttr("signal", sig.String()))
	case <-ctx.Done():
		utils.LavaFormatInfo("Context cancelled")
	}
}

// GetMetrics returns current indexer metrics
func (idx *Indexer) GetMetrics() *IndexerMetrics {
	return idx.metrics
}

// GetCurrentBlock returns the current block being indexed
func (idx *Indexer) GetCurrentBlock() int64 {
	return idx.blockIndexer.GetCurrentBlock()
}

// AddContractWatch adds a new contract to watch
func (idx *Indexer) AddContractWatch(ctx context.Context, watch *ContractWatch) error {
	return idx.contractWatcher.AddWatch(ctx, watch)
}

// GetContractWatches returns all active contract watches
func (idx *Indexer) GetContractWatches() []*ContractWatch {
	return idx.contractWatcher.GetWatches()
}

// RegisterContractABI registers an ABI for a contract address
func (idx *Indexer) RegisterContractABI(address string, abiJSON string) error {
	return idx.abiDecoder.RegisterABI(address, abiJSON)
}

// DecodeLog decodes a log using registered ABIs
func (idx *Indexer) DecodeLog(log *Log) (*DecodedEvent, error) {
	return idx.abiDecoder.DecodeLog(log)
}
