package streamer

import (
	"fmt"
	"time"
)

// StreamerConfig holds configuration for the streaming service
type StreamerConfig struct {
	// RPC Configuration
	RPCEndpoint string `yaml:"rpc_endpoint" json:"rpc_endpoint"` // Smart router or direct RPC endpoint
	ChainID     string `yaml:"chain_id" json:"chain_id"`

	// Streaming Configuration
	StartBlock           int64         `yaml:"start_block" json:"start_block"`                       // Block to start streaming from (0 = latest)
	PollingInterval      time.Duration `yaml:"polling_interval" json:"polling_interval"`             // How often to check for new blocks
	ConfirmationBlocks   int           `yaml:"confirmation_blocks" json:"confirmation_blocks"`       // Wait for N confirmations
	StreamTransactions   bool          `yaml:"stream_transactions" json:"stream_transactions"`       // Stream transactions
	StreamInternalTxs    bool          `yaml:"stream_internal_txs" json:"stream_internal_txs"`       // Stream internal transactions
	StreamLogs           bool          `yaml:"stream_logs" json:"stream_logs"`                       // Stream event logs
	DecodeEvents         bool          `yaml:"decode_events" json:"decode_events"`                   // Decode events with ABIs
	TrackCommonEvents    bool          `yaml:"track_common_events" json:"track_common_events"`       // Track ERC20/ERC721
	MaxEventsBufferSize  int           `yaml:"max_events_buffer_size" json:"max_events_buffer_size"` // Event buffer size
	MaxConcurrentStreams int           `yaml:"max_concurrent_streams" json:"max_concurrent_streams"` // Max concurrent streams

	// WebSocket Server Configuration
	EnableWebSocket       bool          `yaml:"enable_websocket" json:"enable_websocket"`
	WebSocketAddr         string        `yaml:"websocket_addr" json:"websocket_addr"`
	WebSocketPath         string        `yaml:"websocket_path" json:"websocket_path"`
	MaxWebSocketClients   int           `yaml:"max_websocket_clients" json:"max_websocket_clients"`
	WebSocketPingInterval time.Duration `yaml:"websocket_ping_interval" json:"websocket_ping_interval"`

	// Webhook Configuration
	EnableWebhooks    bool          `yaml:"enable_webhooks" json:"enable_webhooks"`
	WebhookWorkers    int           `yaml:"webhook_workers" json:"webhook_workers"`
	WebhookQueueSize  int           `yaml:"webhook_queue_size" json:"webhook_queue_size"`
	WebhookTimeout    time.Duration `yaml:"webhook_timeout" json:"webhook_timeout"`
	WebhookMaxRetries int           `yaml:"webhook_max_retries" json:"webhook_max_retries"`
	WebhookRetryDelay time.Duration `yaml:"webhook_retry_delay" json:"webhook_retry_delay"`

	// API Server Configuration
	EnableAPI     bool   `yaml:"enable_api" json:"enable_api"`
	APIListenAddr string `yaml:"api_listen_addr" json:"api_listen_addr"`
	EnableMetrics bool   `yaml:"enable_metrics" json:"enable_metrics"`
	MetricsAddr   string `yaml:"metrics_addr" json:"metrics_addr"`

	// Contract Watch Configuration
	WatchedContracts []ContractWatchConfig `yaml:"watched_contracts,omitempty" json:"watched_contracts,omitempty"`

	// Performance Configuration
	MaxRetries int           `yaml:"max_retries" json:"max_retries"`
	RetryDelay time.Duration `yaml:"retry_delay" json:"retry_delay"`
}

// ContractWatchConfig holds configuration for watching a specific contract
type ContractWatchConfig struct {
	Name         string   `yaml:"name" json:"name"`
	Address      string   `yaml:"address" json:"address"`
	StartBlock   int64    `yaml:"start_block" json:"start_block"`
	EventFilter  []string `yaml:"event_filter,omitempty" json:"event_filter,omitempty"`
	ABI          string   `yaml:"abi,omitempty" json:"abi,omitempty"`
	StreamAllTxs bool     `yaml:"stream_all_txs,omitempty" json:"stream_all_txs,omitempty"`
}

// Validate validates the streamer configuration
func (c *StreamerConfig) Validate() error {
	if c.RPCEndpoint == "" {
		return fmt.Errorf("rpc_endpoint is required")
	}
	if c.ChainID == "" {
		return fmt.Errorf("chain_id is required")
	}

	// Set defaults
	if c.PollingInterval <= 0 {
		c.PollingInterval = 6 * time.Second
	}
	if c.MaxEventsBufferSize <= 0 {
		c.MaxEventsBufferSize = 10000
	}
	if c.MaxConcurrentStreams <= 0 {
		c.MaxConcurrentStreams = 100
	}
	if c.WebSocketPath == "" {
		c.WebSocketPath = "/stream"
	}
	if c.MaxWebSocketClients <= 0 {
		c.MaxWebSocketClients = 1000
	}
	if c.WebSocketPingInterval <= 0 {
		c.WebSocketPingInterval = 30 * time.Second
	}
	if c.WebhookWorkers <= 0 {
		c.WebhookWorkers = 10
	}
	if c.WebhookQueueSize <= 0 {
		c.WebhookQueueSize = 10000
	}
	if c.WebhookTimeout <= 0 {
		c.WebhookTimeout = 30 * time.Second
	}
	if c.WebhookMaxRetries <= 0 {
		c.WebhookMaxRetries = 3
	}
	if c.WebhookRetryDelay <= 0 {
		c.WebhookRetryDelay = 1 * time.Second
	}
	if c.MaxRetries <= 0 {
		c.MaxRetries = 3
	}
	if c.RetryDelay <= 0 {
		c.RetryDelay = 1 * time.Second
	}

	return nil
}

// DefaultConfig returns a default streamer configuration
func DefaultConfig() *StreamerConfig {
	return &StreamerConfig{
		RPCEndpoint:           "http://localhost:3333",
		ChainID:               "ETH1",
		StartBlock:            0,
		PollingInterval:       6 * time.Second,
		ConfirmationBlocks:    0,
		StreamTransactions:    true,
		StreamInternalTxs:     false,
		StreamLogs:            true,
		DecodeEvents:          true,
		TrackCommonEvents:     true,
		MaxEventsBufferSize:   10000,
		MaxConcurrentStreams:  100,
		EnableWebSocket:       true,
		WebSocketAddr:         ":8080",
		WebSocketPath:         "/stream",
		MaxWebSocketClients:   1000,
		WebSocketPingInterval: 30 * time.Second,
		EnableWebhooks:        true,
		WebhookWorkers:        10,
		WebhookQueueSize:      10000,
		WebhookTimeout:        30 * time.Second,
		WebhookMaxRetries:     3,
		WebhookRetryDelay:     1 * time.Second,
		EnableAPI:             true,
		APIListenAddr:         ":8081",
		EnableMetrics:         true,
		MetricsAddr:           ":9090",
		WatchedContracts:      []ContractWatchConfig{},
		MaxRetries:            3,
		RetryDelay:            1 * time.Second,
	}
}
