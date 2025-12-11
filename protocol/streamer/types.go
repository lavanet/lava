package streamer

import (
	"time"
)

// StreamEvent represents a real-time event emitted by the streamer
type StreamEvent struct {
	ID              string                 `json:"id"`               // Unique event ID
	Type            StreamEventType        `json:"type"`             // Event type
	ChainID         string                 `json:"chain_id"`         // Chain identifier
	BlockNumber     int64                  `json:"block_number"`     // Block number
	BlockHash       string                 `json:"block_hash"`       // Block hash
	TransactionHash string                 `json:"transaction_hash"` // Transaction hash
	Timestamp       int64                  `json:"timestamp"`        // Block timestamp
	Data            map[string]interface{} `json:"data"`             // Event-specific data
	EmittedAt       time.Time              `json:"emitted_at"`       // When streamer emitted this
}

// StreamEventType represents the type of stream event
type StreamEventType string

const (
	EventTypeNewBlock           StreamEventType = "new_block"
	EventTypeTransaction        StreamEventType = "transaction"
	EventTypeInternalTx         StreamEventType = "internal_transaction"
	EventTypeLog                StreamEventType = "log"
	EventTypeDecodedEvent       StreamEventType = "decoded_event"
	EventTypeTokenTransfer      StreamEventType = "token_transfer"
	EventTypeTokenApproval      StreamEventType = "token_approval"
	EventTypeNFTTransfer        StreamEventType = "nft_transfer"
	EventTypeNFTApproval        StreamEventType = "nft_approval"
	EventTypeContractDeployment StreamEventType = "contract_deployment"
)

// Subscription represents a client subscription to events
type Subscription struct {
	ID              string              `json:"id"`
	ClientID        string              `json:"client_id"`
	Filters         *EventFilter        `json:"filters"`
	WebSocket       interface{}         `json:"-"` // WebSocket connection
	Webhook         *WebhookConfig      `json:"webhook,omitempty"`
	MessageQueue    *MessageQueueConfig `json:"message_queue,omitempty"`
	Active          bool                `json:"active"`
	CreatedAt       time.Time           `json:"created_at"`
	LastEventAt     *time.Time          `json:"last_event_at,omitempty"`
	EventCount      int64               `json:"event_count"`
	MaxEventsPerSec int                 `json:"max_events_per_sec,omitempty"` // Rate limit
}

// EventFilter defines which events a subscription receives
type EventFilter struct {
	EventTypes []StreamEventType `json:"event_types,omitempty"` // Filter by event type
	ChainID    string            `json:"chain_id,omitempty"`    // Filter by chain

	// Block filters
	FromBlock *int64 `json:"from_block,omitempty"` // Start from block
	ToBlock   *int64 `json:"to_block,omitempty"`   // End at block (for historical)

	// Transaction filters
	FromAddress     *string  `json:"from_address,omitempty"`     // Filter by sender
	ToAddress       *string  `json:"to_address,omitempty"`       // Filter by receiver
	ContractAddress *string  `json:"contract_address,omitempty"` // Filter by contract
	Addresses       []string `json:"addresses,omitempty"`        // Match any of these addresses

	// Event filters
	Topics           []string `json:"topics,omitempty"`             // Event topics/signatures
	DecodedEventType *string  `json:"decoded_event_type,omitempty"` // transfer, approval, etc.

	// Value filters
	MinValue *string `json:"min_value,omitempty"` // Minimum transaction value
	MaxValue *string `json:"max_value,omitempty"` // Maximum transaction value

	// Advanced filters
	IncludeInternalTxs bool `json:"include_internal_txs,omitempty"` // Include internal transactions
	DecodeEvents       bool `json:"decode_events,omitempty"`        // Decode events with ABI
}

// WebhookConfig defines webhook delivery settings
type WebhookConfig struct {
	URL            string            `json:"url"`
	Headers        map[string]string `json:"headers,omitempty"`
	Secret         string            `json:"secret,omitempty"`            // For HMAC signing
	RetryAttempts  int               `json:"retry_attempts,omitempty"`    // Number of retries
	RetryDelay     time.Duration     `json:"retry_delay,omitempty"`       // Delay between retries
	TimeoutSeconds int               `json:"timeout_seconds,omitempty"`   // Request timeout
	BatchSize      int               `json:"batch_size,omitempty"`        // Batch events (0 = no batching)
	BatchMaxWaitMs int               `json:"batch_max_wait_ms,omitempty"` // Max wait for batch
}

// MessageQueueConfig defines message queue delivery settings
type MessageQueueConfig struct {
	Type     string `json:"type"`    // kafka, rabbitmq, redis
	Address  string `json:"address"` // Broker address
	Topic    string `json:"topic"`   // Topic/queue name
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// Block represents a simplified block for streaming
type Block struct {
	ChainID          string  `json:"chain_id"`
	BlockNumber      int64   `json:"block_number"`
	BlockHash        string  `json:"block_hash"`
	ParentHash       string  `json:"parent_hash"`
	Timestamp        int64   `json:"timestamp"`
	Miner            string  `json:"miner"`
	GasUsed          int64   `json:"gas_used"`
	GasLimit         int64   `json:"gas_limit"`
	BaseFeePerGas    *string `json:"base_fee_per_gas,omitempty"`
	TransactionCount int     `json:"transaction_count"`
}

// Transaction represents a simplified transaction for streaming
type Transaction struct {
	ChainID          string  `json:"chain_id"`
	TxHash           string  `json:"tx_hash"`
	BlockNumber      int64   `json:"block_number"`
	BlockHash        string  `json:"block_hash"`
	TransactionIndex int     `json:"transaction_index"`
	FromAddress      string  `json:"from_address"`
	ToAddress        *string `json:"to_address,omitempty"`
	Value            string  `json:"value"`
	Gas              int64   `json:"gas"`
	GasPrice         string  `json:"gas_price"`
	MaxFeePerGas     *string `json:"max_fee_per_gas,omitempty"`
	MaxPriorityFee   *string `json:"max_priority_fee,omitempty"`
	Input            string  `json:"input"`
	Nonce            int64   `json:"nonce"`
	Status           int     `json:"status"`
	GasUsed          int64   `json:"gas_used"`
	ContractAddress  *string `json:"contract_address,omitempty"`
}

// Log represents a simplified event log for streaming
type Log struct {
	ChainID          string  `json:"chain_id"`
	BlockNumber      int64   `json:"block_number"`
	BlockHash        string  `json:"block_hash"`
	TransactionHash  string  `json:"transaction_hash"`
	TransactionIndex int     `json:"transaction_index"`
	LogIndex         int     `json:"log_index"`
	Address          string  `json:"address"`
	Topic0           *string `json:"topic0,omitempty"`
	Topic1           *string `json:"topic1,omitempty"`
	Topic2           *string `json:"topic2,omitempty"`
	Topic3           *string `json:"topic3,omitempty"`
	Data             string  `json:"data"`
	Removed          bool    `json:"removed"`
}

// InternalTransaction represents an internal transaction
type InternalTransaction struct {
	ChainID      string `json:"chain_id"`
	TxHash       string `json:"tx_hash"`
	BlockNumber  int64  `json:"block_number"`
	FromAddress  string `json:"from_address"`
	ToAddress    string `json:"to_address"`
	Value        string `json:"value"`
	Gas          int64  `json:"gas"`
	GasUsed      int64  `json:"gas_used"`
	Input        string `json:"input"`
	Output       string `json:"output"`
	TraceType    string `json:"trace_type"`
	CallType     string `json:"call_type"`
	TraceAddress string `json:"trace_address"`
	Error        string `json:"error,omitempty"`
}

// DecodedEvent represents a decoded smart contract event
type DecodedEvent struct {
	EventType  string                 `json:"event_type"`
	EventName  string                 `json:"event_name"`
	Signature  string                 `json:"signature"`
	Parameters map[string]interface{} `json:"parameters"`
	RawLog     *Log                   `json:"raw_log,omitempty"`
}

// StreamerMetrics tracks real-time metrics
type StreamerMetrics struct {
	BlocksProcessed      int64 `json:"blocks_processed"`
	EventsEmitted        int64 `json:"events_emitted"`
	ActiveSubscriptions  int64 `json:"active_subscriptions"`
	WebSocketConnections int64 `json:"websocket_connections"`
	WebhooksDelivered    int64 `json:"webhooks_delivered"`
	WebhooksFailed       int64 `json:"webhooks_failed"`
	LastBlockProcessed   int64 `json:"last_block_processed"`
	EventsPerSecond      int64 `json:"events_per_second"`
}
