package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// InternalTransaction represents an internal transaction (contract to contract call)
type InternalTransaction struct {
	ChainID      string    `json:"chain_id" db:"chain_id"`
	TxHash       string    `json:"tx_hash" db:"tx_hash"`
	BlockNumber  int64     `json:"block_number" db:"block_number"`
	FromAddress  string    `json:"from_address" db:"from_address"`
	ToAddress    string    `json:"to_address" db:"to_address"`
	Value        string    `json:"value" db:"value"`
	Gas          int64     `json:"gas" db:"gas"`
	GasUsed      int64     `json:"gas_used" db:"gas_used"`
	Input        string    `json:"input" db:"input"`
	Output       string    `json:"output" db:"output"`
	TraceType    string    `json:"trace_type" db:"trace_type"`       // call, create, suicide, reward
	CallType     string    `json:"call_type" db:"call_type"`         // call, delegatecall, staticcall, callcode
	TraceAddress string    `json:"trace_address" db:"trace_address"` // e.g., "[0,1,2]"
	Error        string    `json:"error" db:"error"`
	IndexedAt    time.Time `json:"indexed_at" db:"indexed_at"`
}

// TraceResult represents the result from debug_traceTransaction
type TraceResult struct {
	Type    string        `json:"type"`
	From    string        `json:"from"`
	To      string        `json:"to"`
	Value   string        `json:"value"`
	Gas     string        `json:"gas"`
	GasUsed string        `json:"gasUsed"`
	Input   string        `json:"input"`
	Output  string        `json:"output"`
	Error   string        `json:"error,omitempty"`
	Calls   []TraceResult `json:"calls,omitempty"`
}

// GetInternalTransactions fetches internal transactions for a given transaction hash
func (c *RPCClient) GetInternalTransactions(ctx context.Context, txHash string) ([]InternalTransaction, error) {
	// Use debug_traceTransaction to get internal calls
	params := []interface{}{
		txHash,
		map[string]interface{}{
			"tracer": "callTracer",
		},
	}

	result, err := c.call(ctx, "debug_traceTransaction", params)
	if err != nil {
		// If debug_traceTransaction is not available, try trace_transaction (Parity/OpenEthereum)
		return c.getInternalTransactionsTrace(ctx, txHash)
	}

	var trace TraceResult
	if err := json.Unmarshal(result, &trace); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trace: %w", err)
	}

	// Flatten the trace tree into a list of internal transactions
	internalTxs := make([]InternalTransaction, 0)
	c.flattenTrace(&trace, txHash, []int{}, &internalTxs)

	return internalTxs, nil
}

// getInternalTransactionsTrace uses trace_transaction (Parity/OpenEthereum)
func (c *RPCClient) getInternalTransactionsTrace(ctx context.Context, txHash string) ([]InternalTransaction, error) {
	result, err := c.call(ctx, "trace_transaction", []interface{}{txHash})
	if err != nil {
		return nil, fmt.Errorf("trace_transaction not available: %w", err)
	}

	var traces []struct {
		Type   string `json:"type"`
		Action struct {
			From     string `json:"from"`
			To       string `json:"to"`
			Value    string `json:"value"`
			Gas      string `json:"gas"`
			Input    string `json:"input"`
			CallType string `json:"callType"`
		} `json:"action"`
		Result struct {
			GasUsed string `json:"gasUsed"`
			Output  string `json:"output"`
		} `json:"result"`
		Error        string `json:"error,omitempty"`
		TraceAddress []int  `json:"traceAddress"`
		BlockNumber  int64  `json:"blockNumber"`
	}

	if err := json.Unmarshal(result, &traces); err != nil {
		return nil, fmt.Errorf("failed to unmarshal traces: %w", err)
	}

	internalTxs := make([]InternalTransaction, 0, len(traces))
	for _, trace := range traces {
		if trace.Type != "call" && trace.Type != "create" {
			continue
		}

		gas, _ := parseHexToInt64(trace.Action.Gas)
		gasUsed, _ := parseHexToInt64(trace.Result.GasUsed)

		traceAddr := "[]"
		if len(trace.TraceAddress) > 0 {
			traceAddr = fmt.Sprintf("%v", trace.TraceAddress)
		}

		internalTx := InternalTransaction{
			ChainID:      c.chainID,
			TxHash:       txHash,
			BlockNumber:  trace.BlockNumber,
			FromAddress:  trace.Action.From,
			ToAddress:    trace.Action.To,
			Value:        trace.Action.Value,
			Gas:          gas,
			GasUsed:      gasUsed,
			Input:        trace.Action.Input,
			Output:       trace.Result.Output,
			TraceType:    trace.Type,
			CallType:     trace.Action.CallType,
			TraceAddress: traceAddr,
			Error:        trace.Error,
			IndexedAt:    time.Now(),
		}

		internalTxs = append(internalTxs, internalTx)
	}

	return internalTxs, nil
}

// flattenTrace recursively flattens a trace tree
func (c *RPCClient) flattenTrace(trace *TraceResult, txHash string, traceAddress []int, result *[]InternalTransaction) {
	// Skip the top-level transaction (it's already indexed as a regular transaction)
	if len(traceAddress) > 0 {
		gas, _ := parseHexToInt64(trace.Gas)
		gasUsed, _ := parseHexToInt64(trace.GasUsed)

		traceAddr := fmt.Sprintf("%v", traceAddress)

		internalTx := InternalTransaction{
			ChainID:      c.chainID,
			TxHash:       txHash,
			FromAddress:  trace.From,
			ToAddress:    trace.To,
			Value:        trace.Value,
			Gas:          gas,
			GasUsed:      gasUsed,
			Input:        trace.Input,
			Output:       trace.Output,
			TraceType:    trace.Type,
			CallType:     trace.Type, // In callTracer, type is the call type
			TraceAddress: traceAddr,
			Error:        trace.Error,
			IndexedAt:    time.Now(),
		}

		*result = append(*result, internalTx)
	}

	// Process nested calls
	for i, call := range trace.Calls {
		nestedAddress := append(traceAddress, i)
		c.flattenTrace(&call, txHash, nestedAddress, result)
	}
}

// Storage interface additions for internal transactions
func (s *SQLiteStorage) SaveInternalTransaction(ctx context.Context, tx *InternalTransaction) error {
	query := `
		INSERT OR REPLACE INTO internal_transactions (
			chain_id, tx_hash, block_number, from_address, to_address, value,
			gas, gas_used, input, output, trace_type, call_type, trace_address,
			error, indexed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	executor := s.getExecutor()
	_, err := executor.ExecContext(ctx, query,
		tx.ChainID, tx.TxHash, tx.BlockNumber, tx.FromAddress, tx.ToAddress, tx.Value,
		tx.Gas, tx.GasUsed, tx.Input, tx.Output, tx.TraceType, tx.CallType, tx.TraceAddress,
		tx.Error, tx.IndexedAt,
	)
	return err
}

func (s *SQLiteStorage) SaveInternalTransactions(ctx context.Context, txs []InternalTransaction) error {
	if len(txs) == 0 {
		return nil
	}

	executor := s.getExecutor()

	for _, tx := range txs {
		if err := s.SaveInternalTransaction(ctx, &tx); err != nil {
			return err
		}
	}

	return nil
}

// GetInternalTransactions retrieves internal transactions for a transaction hash
func (s *SQLiteStorage) GetInternalTransactions(ctx context.Context, chainID string, txHash string) ([]InternalTransaction, error) {
	query := `
		SELECT chain_id, tx_hash, block_number, from_address, to_address, value,
			   gas, gas_used, input, output, trace_type, call_type, trace_address,
			   error, indexed_at
		FROM internal_transactions WHERE chain_id = ? AND tx_hash = ?
		ORDER BY trace_address
	`

	rows, err := s.db.QueryContext(ctx, query, chainID, txHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []InternalTransaction
	for rows.Next() {
		var tx InternalTransaction
		err := rows.Scan(
			&tx.ChainID, &tx.TxHash, &tx.BlockNumber, &tx.FromAddress, &tx.ToAddress, &tx.Value,
			&tx.Gas, &tx.GasUsed, &tx.Input, &tx.Output, &tx.TraceType, &tx.CallType, &tx.TraceAddress,
			&tx.Error, &tx.IndexedAt,
		)
		if err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}

	return txs, rows.Err()
}

// Add internal_transactions table to schema
func (s *SQLiteStorage) addInternalTransactionsTable() error {
	schema := `
	CREATE TABLE IF NOT EXISTS internal_transactions (
		chain_id TEXT NOT NULL,
		tx_hash TEXT NOT NULL,
		block_number INTEGER NOT NULL,
		from_address TEXT NOT NULL,
		to_address TEXT NOT NULL,
		value TEXT NOT NULL,
		gas INTEGER NOT NULL,
		gas_used INTEGER NOT NULL,
		input TEXT NOT NULL,
		output TEXT NOT NULL,
		trace_type TEXT NOT NULL,
		call_type TEXT NOT NULL,
		trace_address TEXT NOT NULL,
		error TEXT NOT NULL,
		indexed_at TIMESTAMP NOT NULL,
		PRIMARY KEY (chain_id, tx_hash, trace_address)
	);

	CREATE INDEX IF NOT EXISTS idx_internal_tx_hash ON internal_transactions(tx_hash);
	CREATE INDEX IF NOT EXISTS idx_internal_block ON internal_transactions(block_number);
	CREATE INDEX IF NOT EXISTS idx_internal_from ON internal_transactions(from_address);
	CREATE INDEX IF NOT EXISTS idx_internal_to ON internal_transactions(to_address);
	`

	_, err := s.db.Exec(schema)
	return err
}
