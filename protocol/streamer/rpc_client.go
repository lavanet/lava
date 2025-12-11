package streamer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lavanet/lava/v5/utils"
)

// RPCClient handles JSON-RPC communication with EVM nodes
type RPCClient struct {
	endpoint   string
	httpClient *http.Client
	chainID    string
	maxRetries int
	retryDelay time.Duration
}

// NewRPCClient creates a new RPC client
func NewRPCClient(endpoint string, chainID string, maxRetries int, retryDelay time.Duration) *RPCClient {
	return &RPCClient{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		chainID:    chainID,
		maxRetries: maxRetries,
		retryDelay: retryDelay,
	}
}

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
	ID      int             `json:"id"`
}

// JSONRPCError represents a JSON-RPC error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// call makes a JSON-RPC call with retries
func (c *RPCClient) call(ctx context.Context, method string, params []interface{}) (json.RawMessage, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			utils.LavaFormatWarning("Retrying RPC call", nil,
				utils.LogAttr("method", method),
				utils.LogAttr("attempt", attempt),
				utils.LogAttr("maxRetries", c.maxRetries),
			)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.retryDelay * time.Duration(attempt)):
			}
		}

		req := &JSONRPCRequest{
			JSONRPC: "2.0",
			Method:  method,
			Params:  params,
			ID:      1,
		}

		reqBody, err := json.Marshal(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to marshal request: %w", err)
			continue
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(reqBody))
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("failed to send request: %w", err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
			continue
		}

		var rpcResp JSONRPCResponse
		if err := json.Unmarshal(body, &rpcResp); err != nil {
			lastErr = fmt.Errorf("failed to unmarshal response: %w", err)
			continue
		}

		if rpcResp.Error != nil {
			lastErr = fmt.Errorf("RPC error: code=%d, message=%s", rpcResp.Error.Code, rpcResp.Error.Message)
			continue
		}

		return rpcResp.Result, nil
	}

	return nil, fmt.Errorf("RPC call failed after %d attempts: %w", c.maxRetries+1, lastErr)
}

// GetBlockNumber returns the latest block number
func (c *RPCClient) GetBlockNumber(ctx context.Context) (int64, error) {
	result, err := c.call(ctx, "eth_blockNumber", []interface{}{})
	if err != nil {
		return 0, err
	}

	var blockHex string
	if err := json.Unmarshal(result, &blockHex); err != nil {
		return 0, fmt.Errorf("failed to unmarshal block number: %w", err)
	}

	var blockNum int64
	_, err = fmt.Sscanf(blockHex, "0x%x", &blockNum)
	if err != nil {
		return 0, fmt.Errorf("failed to parse block number: %w", err)
	}

	return blockNum, nil
}

// EVMBlock represents an Ethereum block from JSON-RPC
type EVMBlock struct {
	Number           string        `json:"number"`
	Hash             string        `json:"hash"`
	ParentHash       string        `json:"parentHash"`
	Nonce            string        `json:"nonce"`
	Sha3Uncles       string        `json:"sha3Uncles"`
	LogsBloom        string        `json:"logsBloom"`
	TransactionsRoot string        `json:"transactionsRoot"`
	StateRoot        string        `json:"stateRoot"`
	ReceiptsRoot     string        `json:"receiptsRoot"`
	Miner            string        `json:"miner"`
	Difficulty       string        `json:"difficulty"`
	TotalDifficulty  string        `json:"totalDifficulty"`
	ExtraData        string        `json:"extraData"`
	Size             string        `json:"size"`
	GasLimit         string        `json:"gasLimit"`
	GasUsed          string        `json:"gasUsed"`
	Timestamp        string        `json:"timestamp"`
	Transactions     []interface{} `json:"transactions"` // Can be hashes or full tx objects
	Uncles           []string      `json:"uncles"`
	BaseFeePerGas    *string       `json:"baseFeePerGas,omitempty"`
}

// GetBlockByNumber retrieves a block by number
func (c *RPCClient) GetBlockByNumber(ctx context.Context, blockNum int64, fullTx bool) (*EVMBlock, error) {
	blockHex := fmt.Sprintf("0x%x", blockNum)
	result, err := c.call(ctx, "eth_getBlockByNumber", []interface{}{blockHex, fullTx})
	if err != nil {
		return nil, err
	}

	var block EVMBlock
	if err := json.Unmarshal(result, &block); err != nil {
		return nil, fmt.Errorf("failed to unmarshal block: %w", err)
	}

	return &block, nil
}

// EVMTransaction represents an Ethereum transaction from JSON-RPC
type EVMTransaction struct {
	BlockHash            string  `json:"blockHash"`
	BlockNumber          string  `json:"blockNumber"`
	From                 string  `json:"from"`
	Gas                  string  `json:"gas"`
	GasPrice             string  `json:"gasPrice"`
	MaxFeePerGas         *string `json:"maxFeePerGas,omitempty"`
	MaxPriorityFeePerGas *string `json:"maxPriorityFeePerGas,omitempty"`
	Hash                 string  `json:"hash"`
	Input                string  `json:"input"`
	Nonce                string  `json:"nonce"`
	To                   *string `json:"to"` // nil for contract creation
	TransactionIndex     string  `json:"transactionIndex"`
	Value                string  `json:"value"`
	Type                 string  `json:"type"`
	ChainID              *string `json:"chainId,omitempty"`
	V                    string  `json:"v"`
	R                    string  `json:"r"`
	S                    string  `json:"s"`
}

// EVMTransactionReceipt represents an Ethereum transaction receipt
type EVMTransactionReceipt struct {
	TransactionHash   string   `json:"transactionHash"`
	TransactionIndex  string   `json:"transactionIndex"`
	BlockHash         string   `json:"blockHash"`
	BlockNumber       string   `json:"blockNumber"`
	From              string   `json:"from"`
	To                *string  `json:"to"`
	CumulativeGasUsed string   `json:"cumulativeGasUsed"`
	GasUsed           string   `json:"gasUsed"`
	ContractAddress   *string  `json:"contractAddress"`
	Logs              []EVMLog `json:"logs"`
	LogsBloom         string   `json:"logsBloom"`
	Status            string   `json:"status"`
	EffectiveGasPrice string   `json:"effectiveGasPrice"`
}

// EVMLog represents an Ethereum event log
type EVMLog struct {
	Address          string   `json:"address"`
	Topics           []string `json:"topics"`
	Data             string   `json:"data"`
	BlockNumber      string   `json:"blockNumber"`
	TransactionHash  string   `json:"transactionHash"`
	TransactionIndex string   `json:"transactionIndex"`
	BlockHash        string   `json:"blockHash"`
	LogIndex         string   `json:"logIndex"`
	Removed          bool     `json:"removed"`
}

// GetTransactionReceipt retrieves a transaction receipt
func (c *RPCClient) GetTransactionReceipt(ctx context.Context, txHash string) (*EVMTransactionReceipt, error) {
	result, err := c.call(ctx, "eth_getTransactionReceipt", []interface{}{txHash})
	if err != nil {
		return nil, err
	}

	// Check if receipt is null (transaction not found or not mined)
	if bytes.Equal(result, []byte("null")) {
		return nil, nil
	}

	var receipt EVMTransactionReceipt
	if err := json.Unmarshal(result, &receipt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal receipt: %w", err)
	}

	return &receipt, nil
}

// LogFilter represents parameters for eth_getLogs
type LogFilter struct {
	FromBlock string     `json:"fromBlock,omitempty"`
	ToBlock   string     `json:"toBlock,omitempty"`
	Address   []string   `json:"address,omitempty"`
	Topics    [][]string `json:"topics,omitempty"`
}

// GetLogs retrieves logs matching the filter
func (c *RPCClient) GetLogs(ctx context.Context, filter LogFilter) ([]EVMLog, error) {
	result, err := c.call(ctx, "eth_getLogs", []interface{}{filter})
	if err != nil {
		return nil, err
	}

	var logs []EVMLog
	if err := json.Unmarshal(result, &logs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal logs: %w", err)
	}

	return logs, nil
}
