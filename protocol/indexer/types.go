package indexer

import (
	"time"
)

// Block represents an indexed EVM block
type Block struct {
	ChainID          string    `json:"chain_id" db:"chain_id"`
	BlockNumber      int64     `json:"block_number" db:"block_number"`
	BlockHash        string    `json:"block_hash" db:"block_hash"`
	ParentHash       string    `json:"parent_hash" db:"parent_hash"`
	Timestamp        int64     `json:"timestamp" db:"timestamp"`
	Miner            string    `json:"miner" db:"miner"`
	GasUsed          int64     `json:"gas_used" db:"gas_used"`
	GasLimit         int64     `json:"gas_limit" db:"gas_limit"`
	BaseFeePerGas    *string   `json:"base_fee_per_gas,omitempty" db:"base_fee_per_gas"` // EIP-1559
	Difficulty       string    `json:"difficulty" db:"difficulty"`
	TotalDifficulty  string    `json:"total_difficulty" db:"total_difficulty"`
	Size             int64     `json:"size" db:"size"`
	TransactionCount int       `json:"transaction_count" db:"transaction_count"`
	ExtraData        string    `json:"extra_data" db:"extra_data"`
	IndexedAt        time.Time `json:"indexed_at" db:"indexed_at"`
}

// Transaction represents an indexed EVM transaction
type Transaction struct {
	ChainID          string    `json:"chain_id" db:"chain_id"`
	TxHash           string    `json:"tx_hash" db:"tx_hash"`
	BlockNumber      int64     `json:"block_number" db:"block_number"`
	BlockHash        string    `json:"block_hash" db:"block_hash"`
	TransactionIndex int       `json:"transaction_index" db:"transaction_index"`
	FromAddress      string    `json:"from_address" db:"from_address"`
	ToAddress        *string   `json:"to_address,omitempty" db:"to_address"` // nil for contract creation
	Value            string    `json:"value" db:"value"`
	Gas              int64     `json:"gas" db:"gas"`
	GasPrice         string    `json:"gas_price" db:"gas_price"`
	MaxFeePerGas     *string   `json:"max_fee_per_gas,omitempty" db:"max_fee_per_gas"`   // EIP-1559
	MaxPriorityFee   *string   `json:"max_priority_fee,omitempty" db:"max_priority_fee"` // EIP-1559
	Input            string    `json:"input" db:"input"`
	Nonce            int64     `json:"nonce" db:"nonce"`
	Status           int       `json:"status" db:"status"` // 1 = success, 0 = failure
	GasUsed          int64     `json:"gas_used" db:"gas_used"`
	ContractAddress  *string   `json:"contract_address,omitempty" db:"contract_address"` // for contract creation
	IndexedAt        time.Time `json:"indexed_at" db:"indexed_at"`
}

// Log represents an indexed smart contract event log
type Log struct {
	ChainID          string    `json:"chain_id" db:"chain_id"`
	BlockNumber      int64     `json:"block_number" db:"block_number"`
	BlockHash        string    `json:"block_hash" db:"block_hash"`
	TransactionHash  string    `json:"transaction_hash" db:"transaction_hash"`
	TransactionIndex int       `json:"transaction_index" db:"transaction_index"`
	LogIndex         int       `json:"log_index" db:"log_index"`
	Address          string    `json:"address" db:"address"` // Contract address
	Topic0           *string   `json:"topic0,omitempty" db:"topic0"`
	Topic1           *string   `json:"topic1,omitempty" db:"topic1"`
	Topic2           *string   `json:"topic2,omitempty" db:"topic2"`
	Topic3           *string   `json:"topic3,omitempty" db:"topic3"`
	Data             string    `json:"data" db:"data"`
	Removed          bool      `json:"removed" db:"removed"`
	IndexedAt        time.Time `json:"indexed_at" db:"indexed_at"`
}

// ContractWatch represents a smart contract to monitor
type ContractWatch struct {
	ID          int64     `json:"id" db:"id"`
	ChainID     string    `json:"chain_id" db:"chain_id"`
	Address     string    `json:"address" db:"address"`
	Name        string    `json:"name" db:"name"`
	StartBlock  int64     `json:"start_block" db:"start_block"`
	EventFilter []string  `json:"event_filter,omitempty" db:"-"` // Topic0 filters (event signatures)
	Active      bool      `json:"active" db:"active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// IndexerState tracks the indexing progress
type IndexerState struct {
	ChainID         string    `json:"chain_id" db:"chain_id"`
	LastBlockNumber int64     `json:"last_block_number" db:"last_block_number"`
	LastBlockHash   string    `json:"last_block_hash" db:"last_block_hash"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// BlockQuery represents query parameters for blocks
type BlockQuery struct {
	ChainID     string
	FromBlock   *int64
	ToBlock     *int64
	Miner       *string
	Limit       int
	Offset      int
	OrderByDesc bool
}

// TransactionQuery represents query parameters for transactions
type TransactionQuery struct {
	ChainID      string
	FromBlock    *int64
	ToBlock      *int64
	FromAddress  *string
	ToAddress    *string
	ContractAddr *string
	Limit        int
	Offset       int
	OrderByDesc  bool
}

// LogQuery represents query parameters for logs
type LogQuery struct {
	ChainID         string
	ContractAddress *string
	FromBlock       *int64
	ToBlock         *int64
	Topics          []string // Topic0, Topic1, Topic2, Topic3 filters
	Limit           int
	Offset          int
	OrderByDesc     bool
}
