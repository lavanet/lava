package indexer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lavanet/lava/v5/utils"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// Storage defines the interface for persistent storage
type Storage interface {
	// Block operations
	SaveBlock(ctx context.Context, block *Block) error
	GetBlock(ctx context.Context, chainID string, blockNumber int64) (*Block, error)
	GetBlocks(ctx context.Context, query *BlockQuery) ([]*Block, error)
	GetLatestBlock(ctx context.Context, chainID string) (*Block, error)

	// Transaction operations
	SaveTransaction(ctx context.Context, tx *Transaction) error
	SaveTransactions(ctx context.Context, txs []*Transaction) error
	GetTransaction(ctx context.Context, chainID string, txHash string) (*Transaction, error)
	GetTransactions(ctx context.Context, query *TransactionQuery) ([]*Transaction, error)

	// Log operations
	SaveLog(ctx context.Context, log *Log) error
	SaveLogs(ctx context.Context, logs []*Log) error
	GetLogs(ctx context.Context, query *LogQuery) ([]*Log, error)

	// Contract watch operations
	SaveContractWatch(ctx context.Context, watch *ContractWatch) error
	GetContractWatches(ctx context.Context, chainID string, active bool) ([]*ContractWatch, error)
	UpdateContractWatch(ctx context.Context, watch *ContractWatch) error
	DeleteContractWatch(ctx context.Context, id int64) error

	// State operations
	SaveIndexerState(ctx context.Context, state *IndexerState) error
	GetIndexerState(ctx context.Context, chainID string) (*IndexerState, error)

	// Utility operations
	BeginTx(ctx context.Context) (Storage, error)
	Commit() error
	Rollback() error
	Close() error
}

// SQLiteStorage implements Storage using SQLite
type SQLiteStorage struct {
	db *sql.DB
	tx *sql.Tx
}

// NewSQLiteStorage creates a new SQLite storage
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrent access
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	storage := &SQLiteStorage{db: db}

	// Initialize schema
	if err := storage.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return storage, nil
}

// initSchema creates database tables if they don't exist
func (s *SQLiteStorage) initSchema() error {
	// Create main tables
	if err := s.createMainTables(); err != nil {
		return err
	}

	// Create internal transactions table
	if err := s.addInternalTransactionsTable(); err != nil {
		return err
	}

	// Create decoded events table
	if err := s.createDecodedEventsTable(); err != nil {
		return err
	}

	return nil
}

func (s *SQLiteStorage) createMainTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS blocks (
		chain_id TEXT NOT NULL,
		block_number INTEGER NOT NULL,
		block_hash TEXT NOT NULL,
		parent_hash TEXT NOT NULL,
		timestamp INTEGER NOT NULL,
		miner TEXT NOT NULL,
		gas_used INTEGER NOT NULL,
		gas_limit INTEGER NOT NULL,
		base_fee_per_gas TEXT,
		difficulty TEXT NOT NULL,
		total_difficulty TEXT NOT NULL,
		size INTEGER NOT NULL,
		transaction_count INTEGER NOT NULL,
		extra_data TEXT NOT NULL,
		indexed_at TIMESTAMP NOT NULL,
		PRIMARY KEY (chain_id, block_number)
	);

	CREATE INDEX IF NOT EXISTS idx_blocks_hash ON blocks(block_hash);
	CREATE INDEX IF NOT EXISTS idx_blocks_timestamp ON blocks(timestamp);
	CREATE INDEX IF NOT EXISTS idx_blocks_miner ON blocks(miner);

	CREATE TABLE IF NOT EXISTS transactions (
		chain_id TEXT NOT NULL,
		tx_hash TEXT NOT NULL,
		block_number INTEGER NOT NULL,
		block_hash TEXT NOT NULL,
		transaction_index INTEGER NOT NULL,
		from_address TEXT NOT NULL,
		to_address TEXT,
		value TEXT NOT NULL,
		gas INTEGER NOT NULL,
		gas_price TEXT NOT NULL,
		max_fee_per_gas TEXT,
		max_priority_fee TEXT,
		input TEXT NOT NULL,
		nonce INTEGER NOT NULL,
		status INTEGER NOT NULL,
		gas_used INTEGER NOT NULL,
		contract_address TEXT,
		indexed_at TIMESTAMP NOT NULL,
		PRIMARY KEY (chain_id, tx_hash)
	);

	CREATE INDEX IF NOT EXISTS idx_transactions_block ON transactions(block_number);
	CREATE INDEX IF NOT EXISTS idx_transactions_from ON transactions(from_address);
	CREATE INDEX IF NOT EXISTS idx_transactions_to ON transactions(to_address);
	CREATE INDEX IF NOT EXISTS idx_transactions_contract ON transactions(contract_address);

	CREATE TABLE IF NOT EXISTS logs (
		chain_id TEXT NOT NULL,
		block_number INTEGER NOT NULL,
		block_hash TEXT NOT NULL,
		transaction_hash TEXT NOT NULL,
		transaction_index INTEGER NOT NULL,
		log_index INTEGER NOT NULL,
		address TEXT NOT NULL,
		topic0 TEXT,
		topic1 TEXT,
		topic2 TEXT,
		topic3 TEXT,
		data TEXT NOT NULL,
		removed INTEGER NOT NULL,
		indexed_at TIMESTAMP NOT NULL,
		PRIMARY KEY (chain_id, transaction_hash, log_index)
	);

	CREATE INDEX IF NOT EXISTS idx_logs_block ON logs(block_number);
	CREATE INDEX IF NOT EXISTS idx_logs_address ON logs(address);
	CREATE INDEX IF NOT EXISTS idx_logs_topic0 ON logs(topic0);
	CREATE INDEX IF NOT EXISTS idx_logs_topic1 ON logs(topic1);

	CREATE TABLE IF NOT EXISTS contract_watches (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chain_id TEXT NOT NULL,
		address TEXT NOT NULL,
		name TEXT NOT NULL,
		start_block INTEGER NOT NULL,
		active INTEGER NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		UNIQUE(chain_id, address)
	);

	CREATE INDEX IF NOT EXISTS idx_contract_watches_chain ON contract_watches(chain_id, active);

	CREATE TABLE IF NOT EXISTS indexer_state (
		chain_id TEXT PRIMARY KEY,
		last_block_number INTEGER NOT NULL,
		last_block_hash TEXT NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);
	`

	_, err := s.db.Exec(schema)
	return err
}

// createDecodedEventsTable creates table for decoded events
func (s *SQLiteStorage) createDecodedEventsTable() error {
	schema := `
	CREATE TABLE IF NOT EXISTS decoded_events (
		chain_id TEXT NOT NULL,
		transaction_hash TEXT NOT NULL,
		log_index INTEGER NOT NULL,
		block_number INTEGER NOT NULL,
		contract_address TEXT NOT NULL,
		event_type TEXT NOT NULL,
		event_name TEXT NOT NULL,
		event_signature TEXT NOT NULL,
		parameters TEXT NOT NULL,
		indexed_at TIMESTAMP NOT NULL,
		PRIMARY KEY (chain_id, transaction_hash, log_index)
	);

	CREATE INDEX IF NOT EXISTS idx_decoded_events_contract ON decoded_events(contract_address);
	CREATE INDEX IF NOT EXISTS idx_decoded_events_type ON decoded_events(event_type);
	CREATE INDEX IF NOT EXISTS idx_decoded_events_block ON decoded_events(block_number);
	`

	_, err := s.db.Exec(schema)
	return err
}

// SaveDecodedEvent saves a decoded event
func (s *SQLiteStorage) SaveDecodedEvent(ctx context.Context, decoded *DecodedEvent) error {
	if decoded.RawLog == nil {
		return fmt.Errorf("raw log is required")
	}

	paramsJSON, err := json.Marshal(decoded.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	query := `
		INSERT OR REPLACE INTO decoded_events (
			chain_id, transaction_hash, log_index, block_number, contract_address,
			event_type, event_name, event_signature, parameters, indexed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	executor := s.getExecutor()
	_, err = executor.ExecContext(ctx, query,
		decoded.RawLog.ChainID,
		decoded.RawLog.TransactionHash,
		decoded.RawLog.LogIndex,
		decoded.RawLog.BlockNumber,
		decoded.RawLog.Address,
		string(decoded.EventType),
		decoded.EventName,
		decoded.Signature,
		string(paramsJSON),
		time.Now(),
	)
	return err
}

// GetDecodedEvents retrieves decoded events
func (s *SQLiteStorage) GetDecodedEvents(ctx context.Context, chainID string, contractAddr *string, eventType *string, fromBlock, toBlock *int64, limit int) ([]*DecodedEvent, error) {
	sql := `SELECT chain_id, transaction_hash, log_index, block_number, contract_address,
				   event_type, event_name, event_signature, parameters, indexed_at
			FROM decoded_events WHERE chain_id = ?`
	args := []interface{}{chainID}

	if contractAddr != nil {
		sql += " AND contract_address = ?"
		args = append(args, *contractAddr)
	}
	if eventType != nil {
		sql += " AND event_type = ?"
		args = append(args, *eventType)
	}
	if fromBlock != nil {
		sql += " AND block_number >= ?"
		args = append(args, *fromBlock)
	}
	if toBlock != nil {
		sql += " AND block_number <= ?"
		args = append(args, *toBlock)
	}

	sql += " ORDER BY block_number DESC, log_index DESC"

	if limit > 0 {
		sql += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*DecodedEvent
	for rows.Next() {
		var event DecodedEvent
		var paramsJSON string
		var chainID, txHash string
		var logIndex int
		var blockNumber int64
		var contractAddr string
		var indexedAt time.Time

		err := rows.Scan(
			&chainID, &txHash, &logIndex, &blockNumber, &contractAddr,
			&event.EventType, &event.EventName, &event.Signature, &paramsJSON, &indexedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(paramsJSON), &event.Parameters); err != nil {
			return nil, err
		}

		// Create minimal raw log for context
		event.RawLog = &Log{
			ChainID:         chainID,
			TransactionHash: txHash,
			LogIndex:        logIndex,
			BlockNumber:     blockNumber,
			Address:         contractAddr,
			IndexedAt:       indexedAt,
		}

		events = append(events, &event)
	}

	return events, rows.Err()
}

// SaveBlock saves a block to the database
func (s *SQLiteStorage) SaveBlock(ctx context.Context, block *Block) error {
	query := `
		INSERT OR REPLACE INTO blocks (
			chain_id, block_number, block_hash, parent_hash, timestamp,
			miner, gas_used, gas_limit, base_fee_per_gas, difficulty,
			total_difficulty, size, transaction_count, extra_data, indexed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	executor := s.getExecutor()
	_, err := executor.ExecContext(ctx, query,
		block.ChainID, block.BlockNumber, block.BlockHash, block.ParentHash, block.Timestamp,
		block.Miner, block.GasUsed, block.GasLimit, block.BaseFeePerGas, block.Difficulty,
		block.TotalDifficulty, block.Size, block.TransactionCount, block.ExtraData, block.IndexedAt,
	)
	return err
}

// GetBlock retrieves a block by chain ID and block number
func (s *SQLiteStorage) GetBlock(ctx context.Context, chainID string, blockNumber int64) (*Block, error) {
	query := `
		SELECT chain_id, block_number, block_hash, parent_hash, timestamp,
			   miner, gas_used, gas_limit, base_fee_per_gas, difficulty,
			   total_difficulty, size, transaction_count, extra_data, indexed_at
		FROM blocks WHERE chain_id = ? AND block_number = ?
	`

	block := &Block{}
	err := s.db.QueryRowContext(ctx, query, chainID, blockNumber).Scan(
		&block.ChainID, &block.BlockNumber, &block.BlockHash, &block.ParentHash, &block.Timestamp,
		&block.Miner, &block.GasUsed, &block.GasLimit, &block.BaseFeePerGas, &block.Difficulty,
		&block.TotalDifficulty, &block.Size, &block.TransactionCount, &block.ExtraData, &block.IndexedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return block, nil
}

// GetBlocks retrieves blocks matching the query
func (s *SQLiteStorage) GetBlocks(ctx context.Context, query *BlockQuery) ([]*Block, error) {
	sql := `SELECT chain_id, block_number, block_hash, parent_hash, timestamp,
				   miner, gas_used, gas_limit, base_fee_per_gas, difficulty,
				   total_difficulty, size, transaction_count, extra_data, indexed_at
			FROM blocks WHERE chain_id = ?`
	args := []interface{}{query.ChainID}

	if query.FromBlock != nil {
		sql += " AND block_number >= ?"
		args = append(args, *query.FromBlock)
	}
	if query.ToBlock != nil {
		sql += " AND block_number <= ?"
		args = append(args, *query.ToBlock)
	}
	if query.Miner != nil {
		sql += " AND miner = ?"
		args = append(args, *query.Miner)
	}

	if query.OrderByDesc {
		sql += " ORDER BY block_number DESC"
	} else {
		sql += " ORDER BY block_number ASC"
	}

	if query.Limit > 0 {
		sql += " LIMIT ?"
		args = append(args, query.Limit)
	}
	if query.Offset > 0 {
		sql += " OFFSET ?"
		args = append(args, query.Offset)
	}

	rows, err := s.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blocks []*Block
	for rows.Next() {
		block := &Block{}
		err := rows.Scan(
			&block.ChainID, &block.BlockNumber, &block.BlockHash, &block.ParentHash, &block.Timestamp,
			&block.Miner, &block.GasUsed, &block.GasLimit, &block.BaseFeePerGas, &block.Difficulty,
			&block.TotalDifficulty, &block.Size, &block.TransactionCount, &block.ExtraData, &block.IndexedAt,
		)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}

	return blocks, rows.Err()
}

// GetLatestBlock retrieves the latest indexed block
func (s *SQLiteStorage) GetLatestBlock(ctx context.Context, chainID string) (*Block, error) {
	query := `
		SELECT chain_id, block_number, block_hash, parent_hash, timestamp,
			   miner, gas_used, gas_limit, base_fee_per_gas, difficulty,
			   total_difficulty, size, transaction_count, extra_data, indexed_at
		FROM blocks WHERE chain_id = ? ORDER BY block_number DESC LIMIT 1
	`

	block := &Block{}
	err := s.db.QueryRowContext(ctx, query, chainID).Scan(
		&block.ChainID, &block.BlockNumber, &block.BlockHash, &block.ParentHash, &block.Timestamp,
		&block.Miner, &block.GasUsed, &block.GasLimit, &block.BaseFeePerGas, &block.Difficulty,
		&block.TotalDifficulty, &block.Size, &block.TransactionCount, &block.ExtraData, &block.IndexedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return block, nil
}

// SaveTransaction saves a single transaction
func (s *SQLiteStorage) SaveTransaction(ctx context.Context, tx *Transaction) error {
	query := `
		INSERT OR REPLACE INTO transactions (
			chain_id, tx_hash, block_number, block_hash, transaction_index,
			from_address, to_address, value, gas, gas_price, max_fee_per_gas,
			max_priority_fee, input, nonce, status, gas_used, contract_address, indexed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	executor := s.getExecutor()
	_, err := executor.ExecContext(ctx, query,
		tx.ChainID, tx.TxHash, tx.BlockNumber, tx.BlockHash, tx.TransactionIndex,
		tx.FromAddress, tx.ToAddress, tx.Value, tx.Gas, tx.GasPrice, tx.MaxFeePerGas,
		tx.MaxPriorityFee, tx.Input, tx.Nonce, tx.Status, tx.GasUsed, tx.ContractAddress, tx.IndexedAt,
	)
	return err
}

// SaveTransactions saves multiple transactions in a batch
func (s *SQLiteStorage) SaveTransactions(ctx context.Context, txs []*Transaction) error {
	if len(txs) == 0 {
		return nil
	}

	executor := s.getExecutor()

	// Build bulk insert
	placeholders := make([]string, len(txs))
	args := make([]interface{}, 0, len(txs)*18)

	for i, tx := range txs {
		placeholders[i] = "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
		args = append(args,
			tx.ChainID, tx.TxHash, tx.BlockNumber, tx.BlockHash, tx.TransactionIndex,
			tx.FromAddress, tx.ToAddress, tx.Value, tx.Gas, tx.GasPrice, tx.MaxFeePerGas,
			tx.MaxPriorityFee, tx.Input, tx.Nonce, tx.Status, tx.GasUsed, tx.ContractAddress, tx.IndexedAt,
		)
	}

	query := fmt.Sprintf(`
		INSERT OR REPLACE INTO transactions (
			chain_id, tx_hash, block_number, block_hash, transaction_index,
			from_address, to_address, value, gas, gas_price, max_fee_per_gas,
			max_priority_fee, input, nonce, status, gas_used, contract_address, indexed_at
		) VALUES %s
	`, strings.Join(placeholders, ","))

	_, err := executor.ExecContext(ctx, query, args...)
	return err
}

// GetTransaction retrieves a transaction by hash
func (s *SQLiteStorage) GetTransaction(ctx context.Context, chainID string, txHash string) (*Transaction, error) {
	query := `
		SELECT chain_id, tx_hash, block_number, block_hash, transaction_index,
			   from_address, to_address, value, gas, gas_price, max_fee_per_gas,
			   max_priority_fee, input, nonce, status, gas_used, contract_address, indexed_at
		FROM transactions WHERE chain_id = ? AND tx_hash = ?
	`

	tx := &Transaction{}
	err := s.db.QueryRowContext(ctx, query, chainID, txHash).Scan(
		&tx.ChainID, &tx.TxHash, &tx.BlockNumber, &tx.BlockHash, &tx.TransactionIndex,
		&tx.FromAddress, &tx.ToAddress, &tx.Value, &tx.Gas, &tx.GasPrice, &tx.MaxFeePerGas,
		&tx.MaxPriorityFee, &tx.Input, &tx.Nonce, &tx.Status, &tx.GasUsed, &tx.ContractAddress, &tx.IndexedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return tx, nil
}

// GetTransactions retrieves transactions matching the query
func (s *SQLiteStorage) GetTransactions(ctx context.Context, query *TransactionQuery) ([]*Transaction, error) {
	sql := `SELECT chain_id, tx_hash, block_number, block_hash, transaction_index,
				   from_address, to_address, value, gas, gas_price, max_fee_per_gas,
				   max_priority_fee, input, nonce, status, gas_used, contract_address, indexed_at
			FROM transactions WHERE chain_id = ?`
	args := []interface{}{query.ChainID}

	if query.FromBlock != nil {
		sql += " AND block_number >= ?"
		args = append(args, *query.FromBlock)
	}
	if query.ToBlock != nil {
		sql += " AND block_number <= ?"
		args = append(args, *query.ToBlock)
	}
	if query.FromAddress != nil {
		sql += " AND from_address = ?"
		args = append(args, *query.FromAddress)
	}
	if query.ToAddress != nil {
		sql += " AND to_address = ?"
		args = append(args, *query.ToAddress)
	}
	if query.ContractAddr != nil {
		sql += " AND contract_address = ?"
		args = append(args, *query.ContractAddr)
	}

	if query.OrderByDesc {
		sql += " ORDER BY block_number DESC, transaction_index DESC"
	} else {
		sql += " ORDER BY block_number ASC, transaction_index ASC"
	}

	if query.Limit > 0 {
		sql += " LIMIT ?"
		args = append(args, query.Limit)
	}
	if query.Offset > 0 {
		sql += " OFFSET ?"
		args = append(args, query.Offset)
	}

	rows, err := s.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []*Transaction
	for rows.Next() {
		tx := &Transaction{}
		err := rows.Scan(
			&tx.ChainID, &tx.TxHash, &tx.BlockNumber, &tx.BlockHash, &tx.TransactionIndex,
			&tx.FromAddress, &tx.ToAddress, &tx.Value, &tx.Gas, &tx.GasPrice, &tx.MaxFeePerGas,
			&tx.MaxPriorityFee, &tx.Input, &tx.Nonce, &tx.Status, &tx.GasUsed, &tx.ContractAddress, &tx.IndexedAt,
		)
		if err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}

	return txs, rows.Err()
}

// SaveLog saves a single log
func (s *SQLiteStorage) SaveLog(ctx context.Context, log *Log) error {
	query := `
		INSERT OR REPLACE INTO logs (
			chain_id, block_number, block_hash, transaction_hash, transaction_index,
			log_index, address, topic0, topic1, topic2, topic3, data, removed, indexed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	executor := s.getExecutor()
	_, err := executor.ExecContext(ctx, query,
		log.ChainID, log.BlockNumber, log.BlockHash, log.TransactionHash, log.TransactionIndex,
		log.LogIndex, log.Address, log.Topic0, log.Topic1, log.Topic2, log.Topic3, log.Data, log.Removed, log.IndexedAt,
	)
	return err
}

// SaveLogs saves multiple logs in a batch
func (s *SQLiteStorage) SaveLogs(ctx context.Context, logs []*Log) error {
	if len(logs) == 0 {
		return nil
	}

	executor := s.getExecutor()

	placeholders := make([]string, len(logs))
	args := make([]interface{}, 0, len(logs)*14)

	for i, log := range logs {
		placeholders[i] = "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
		args = append(args,
			log.ChainID, log.BlockNumber, log.BlockHash, log.TransactionHash, log.TransactionIndex,
			log.LogIndex, log.Address, log.Topic0, log.Topic1, log.Topic2, log.Topic3, log.Data, log.Removed, log.IndexedAt,
		)
	}

	query := fmt.Sprintf(`
		INSERT OR REPLACE INTO logs (
			chain_id, block_number, block_hash, transaction_hash, transaction_index,
			log_index, address, topic0, topic1, topic2, topic3, data, removed, indexed_at
		) VALUES %s
	`, strings.Join(placeholders, ","))

	_, err := executor.ExecContext(ctx, query, args...)
	return err
}

// GetLogs retrieves logs matching the query
func (s *SQLiteStorage) GetLogs(ctx context.Context, query *LogQuery) ([]*Log, error) {
	sql := `SELECT chain_id, block_number, block_hash, transaction_hash, transaction_index,
				   log_index, address, topic0, topic1, topic2, topic3, data, removed, indexed_at
			FROM logs WHERE chain_id = ?`
	args := []interface{}{query.ChainID}

	if query.ContractAddress != nil {
		sql += " AND address = ?"
		args = append(args, *query.ContractAddress)
	}
	if query.FromBlock != nil {
		sql += " AND block_number >= ?"
		args = append(args, *query.FromBlock)
	}
	if query.ToBlock != nil {
		sql += " AND block_number <= ?"
		args = append(args, *query.ToBlock)
	}

	// Add topic filters
	for i, topic := range query.Topics {
		if topic != "" {
			sql += fmt.Sprintf(" AND topic%d = ?", i)
			args = append(args, topic)
		}
	}

	if query.OrderByDesc {
		sql += " ORDER BY block_number DESC, log_index DESC"
	} else {
		sql += " ORDER BY block_number ASC, log_index ASC"
	}

	if query.Limit > 0 {
		sql += " LIMIT ?"
		args = append(args, query.Limit)
	}
	if query.Offset > 0 {
		sql += " OFFSET ?"
		args = append(args, query.Offset)
	}

	rows, err := s.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*Log
	for rows.Next() {
		log := &Log{}
		err := rows.Scan(
			&log.ChainID, &log.BlockNumber, &log.BlockHash, &log.TransactionHash, &log.TransactionIndex,
			&log.LogIndex, &log.Address, &log.Topic0, &log.Topic1, &log.Topic2, &log.Topic3, &log.Data, &log.Removed, &log.IndexedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// SaveContractWatch saves a contract watch configuration
func (s *SQLiteStorage) SaveContractWatch(ctx context.Context, watch *ContractWatch) error {
	query := `
		INSERT INTO contract_watches (chain_id, address, name, start_block, active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	executor := s.getExecutor()
	result, err := executor.ExecContext(ctx, query,
		watch.ChainID, watch.Address, watch.Name, watch.StartBlock, watch.Active, watch.CreatedAt, watch.UpdatedAt,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	watch.ID = id
	return nil
}

// GetContractWatches retrieves contract watches
func (s *SQLiteStorage) GetContractWatches(ctx context.Context, chainID string, active bool) ([]*ContractWatch, error) {
	query := `
		SELECT id, chain_id, address, name, start_block, active, created_at, updated_at
		FROM contract_watches WHERE chain_id = ? AND active = ?
	`

	rows, err := s.db.QueryContext(ctx, query, chainID, active)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var watches []*ContractWatch
	for rows.Next() {
		watch := &ContractWatch{}
		err := rows.Scan(
			&watch.ID, &watch.ChainID, &watch.Address, &watch.Name, &watch.StartBlock,
			&watch.Active, &watch.CreatedAt, &watch.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		watches = append(watches, watch)
	}

	return watches, rows.Err()
}

// UpdateContractWatch updates a contract watch
func (s *SQLiteStorage) UpdateContractWatch(ctx context.Context, watch *ContractWatch) error {
	query := `
		UPDATE contract_watches
		SET name = ?, start_block = ?, active = ?, updated_at = ?
		WHERE id = ?
	`

	executor := s.getExecutor()
	_, err := executor.ExecContext(ctx, query,
		watch.Name, watch.StartBlock, watch.Active, time.Now(), watch.ID,
	)
	return err
}

// DeleteContractWatch deletes a contract watch
func (s *SQLiteStorage) DeleteContractWatch(ctx context.Context, id int64) error {
	query := "DELETE FROM contract_watches WHERE id = ?"
	executor := s.getExecutor()
	_, err := executor.ExecContext(ctx, query, id)
	return err
}

// SaveIndexerState saves the indexer state
func (s *SQLiteStorage) SaveIndexerState(ctx context.Context, state *IndexerState) error {
	query := `
		INSERT OR REPLACE INTO indexer_state (chain_id, last_block_number, last_block_hash, updated_at)
		VALUES (?, ?, ?, ?)
	`

	executor := s.getExecutor()
	_, err := executor.ExecContext(ctx, query,
		state.ChainID, state.LastBlockNumber, state.LastBlockHash, state.UpdatedAt,
	)
	return err
}

// GetIndexerState retrieves the indexer state
func (s *SQLiteStorage) GetIndexerState(ctx context.Context, chainID string) (*IndexerState, error) {
	query := `
		SELECT chain_id, last_block_number, last_block_hash, updated_at
		FROM indexer_state WHERE chain_id = ?
	`

	state := &IndexerState{}
	err := s.db.QueryRowContext(ctx, query, chainID).Scan(
		&state.ChainID, &state.LastBlockNumber, &state.LastBlockHash, &state.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return state, nil
}

// BeginTx starts a new transaction
func (s *SQLiteStorage) BeginTx(ctx context.Context) (Storage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &SQLiteStorage{
		db: s.db,
		tx: tx,
	}, nil
}

// Commit commits the transaction
func (s *SQLiteStorage) Commit() error {
	if s.tx == nil {
		return fmt.Errorf("no transaction to commit")
	}
	return s.tx.Commit()
}

// Rollback rolls back the transaction
func (s *SQLiteStorage) Rollback() error {
	if s.tx == nil {
		return nil
	}
	return s.tx.Rollback()
}

// Close closes the database connection
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

// getExecutor returns the appropriate executor (transaction or database)
func (s *SQLiteStorage) getExecutor() interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
} {
	if s.tx != nil {
		return s.tx
	}
	return s.db
}

// NewStorage creates a new storage instance based on configuration
func NewStorage(config *IndexerConfig) (Storage, error) {
	switch config.DatabaseType {
	case "sqlite":
		storage, err := NewSQLiteStorage(config.DatabaseURL)
		if err != nil {
			return nil, err
		}
		utils.LavaFormatInfo("SQLite storage initialized", utils.LogAttr("path", config.DatabaseURL))
		return storage, nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", config.DatabaseType)
	}
}
