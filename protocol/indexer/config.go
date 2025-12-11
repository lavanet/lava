package indexer

import (
	"fmt"
	"time"
)

// IndexerConfig holds configuration for the indexing service
type IndexerConfig struct {
	// RPC Configuration
	RPCEndpoint string `yaml:"rpc_endpoint" json:"rpc_endpoint"` // Smart router or direct RPC endpoint
	ChainID     string `yaml:"chain_id" json:"chain_id"`

	// Database Configuration
	DatabaseType string `yaml:"database_type" json:"database_type"` // "postgres", "sqlite", "badger"
	DatabaseURL  string `yaml:"database_url" json:"database_url"`   // Connection string

	// Indexing Configuration
	StartBlock             int64         `yaml:"start_block" json:"start_block"`                           // Block to start indexing from
	BatchSize              int           `yaml:"batch_size" json:"batch_size"`                             // Number of blocks to fetch per batch
	ConcurrentWorkers      int           `yaml:"concurrent_workers" json:"concurrent_workers"`             // Number of parallel workers
	PollingInterval        time.Duration `yaml:"polling_interval" json:"polling_interval"`                 // How often to check for new blocks
	ConfirmationBlocks     int           `yaml:"confirmation_blocks" json:"confirmation_blocks"`           // Wait for N confirmations before indexing
	ReorgDepth             int           `yaml:"reorg_depth" json:"reorg_depth"`                           // How many blocks to check for reorgs
	IndexTransactions      bool          `yaml:"index_transactions" json:"index_transactions"`             // Whether to index full transaction data
	IndexInternalTxs       bool          `yaml:"index_internal_txs" json:"index_internal_txs"`             // Whether to index internal transactions
	IndexLogs              bool          `yaml:"index_logs" json:"index_logs"`                             // Whether to index event logs
	IndexFullBlocks        bool          `yaml:"index_full_blocks" json:"index_full_blocks"`               // Whether to get full block with transactions
	DecodeEvents           bool          `yaml:"decode_events" json:"decode_events"`                       // Whether to decode events using ABIs
	TrackCommonEvents      bool          `yaml:"track_common_events" json:"track_common_events"`           // Track common ERC20/ERC721 events
	TrackSpecificAddresses []string      `yaml:"track_specific_addresses" json:"track_specific_addresses"` // Track all activity for these addresses

	// API Server Configuration
	EnableAPI     bool   `yaml:"enable_api" json:"enable_api"`           // Enable query API server
	APIListenAddr string `yaml:"api_listen_addr" json:"api_listen_addr"` // API server listen address
	EnableMetrics bool   `yaml:"enable_metrics" json:"enable_metrics"`   // Enable Prometheus metrics
	MetricsAddr   string `yaml:"metrics_addr" json:"metrics_addr"`       // Metrics server address

	// Contract Watch Configuration
	WatchedContracts []ContractWatchConfig `yaml:"watched_contracts,omitempty" json:"watched_contracts,omitempty"`

	// Performance Configuration
	CacheEnabled bool          `yaml:"cache_enabled" json:"cache_enabled"` // Enable in-memory caching
	CacheSize    int           `yaml:"cache_size" json:"cache_size"`       // Cache size in MB
	CacheTTL     time.Duration `yaml:"cache_ttl" json:"cache_ttl"`         // Cache entry TTL
	MaxRetries   int           `yaml:"max_retries" json:"max_retries"`     // Max retries for failed RPC calls
	RetryDelay   time.Duration `yaml:"retry_delay" json:"retry_delay"`     // Delay between retries
}

// ContractWatchConfig holds configuration for watching a specific contract
type ContractWatchConfig struct {
	Name             string   `yaml:"name" json:"name"`
	Address          string   `yaml:"address" json:"address"`
	StartBlock       int64    `yaml:"start_block" json:"start_block"`
	EventFilter      []string `yaml:"event_filter,omitempty" json:"event_filter,omitempty"`             // Event signatures to filter
	ABI              string   `yaml:"abi,omitempty" json:"abi,omitempty"`                               // Contract ABI (JSON string) for decoding
	TrackAllActivity bool     `yaml:"track_all_activity,omitempty" json:"track_all_activity,omitempty"` // Track all txs to/from this address
}

// Validate validates the indexer configuration
func (c *IndexerConfig) Validate() error {
	if c.RPCEndpoint == "" {
		return fmt.Errorf("rpc_endpoint is required")
	}
	if c.ChainID == "" {
		return fmt.Errorf("chain_id is required")
	}
	if c.DatabaseType == "" {
		return fmt.Errorf("database_type is required")
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("database_url is required")
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 10 // Default
	}
	if c.ConcurrentWorkers <= 0 {
		c.ConcurrentWorkers = 4 // Default
	}
	if c.PollingInterval <= 0 {
		c.PollingInterval = 12 * time.Second // Default to Ethereum block time
	}
	if c.ConfirmationBlocks < 0 {
		c.ConfirmationBlocks = 0
	}
	if c.ReorgDepth <= 0 {
		c.ReorgDepth = 10 // Default
	}
	if c.MaxRetries <= 0 {
		c.MaxRetries = 3
	}
	if c.RetryDelay <= 0 {
		c.RetryDelay = 1 * time.Second
	}
	return nil
}

// DefaultConfig returns a default indexer configuration
func DefaultConfig() *IndexerConfig {
	return &IndexerConfig{
		RPCEndpoint:            "http://localhost:3333",
		ChainID:                "ETH1",
		DatabaseType:           "sqlite",
		DatabaseURL:            "indexer.db",
		StartBlock:             0,
		BatchSize:              10,
		ConcurrentWorkers:      4,
		PollingInterval:        12 * time.Second,
		ConfirmationBlocks:     0,
		ReorgDepth:             10,
		IndexTransactions:      true,
		IndexInternalTxs:       false, // Disabled by default (requires debug_traceTransaction)
		IndexLogs:              true,
		IndexFullBlocks:        true,
		DecodeEvents:           true,
		TrackCommonEvents:      true,
		TrackSpecificAddresses: []string{},
		EnableAPI:              true,
		APIListenAddr:          ":8080",
		EnableMetrics:          true,
		MetricsAddr:            ":9090",
		WatchedContracts:       []ContractWatchConfig{},
		CacheEnabled:           true,
		CacheSize:              100,
		CacheTTL:               5 * time.Minute,
		MaxRetries:             3,
		RetryDelay:             1 * time.Second,
	}
}
