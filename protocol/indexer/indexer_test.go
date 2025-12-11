package indexer

import (
	"context"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	if config.RPCEndpoint == "" {
		t.Error("RPCEndpoint should not be empty")
	}

	if config.ChainID == "" {
		t.Error("ChainID should not be empty")
	}

	if config.DatabaseType == "" {
		t.Error("DatabaseType should not be empty")
	}

	if config.BatchSize <= 0 {
		t.Error("BatchSize should be positive")
	}

	if config.ConcurrentWorkers <= 0 {
		t.Error("ConcurrentWorkers should be positive")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *IndexerConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "missing rpc endpoint",
			config: &IndexerConfig{
				ChainID:      "ETH1",
				DatabaseType: "sqlite",
				DatabaseURL:  "test.db",
			},
			wantErr: true,
		},
		{
			name: "missing chain id",
			config: &IndexerConfig{
				RPCEndpoint:  "http://localhost:3333",
				DatabaseType: "sqlite",
				DatabaseURL:  "test.db",
			},
			wantErr: true,
		},
		{
			name: "missing database type",
			config: &IndexerConfig{
				RPCEndpoint: "http://localhost:3333",
				ChainID:     "ETH1",
				DatabaseURL: "test.db",
			},
			wantErr: true,
		},
		{
			name: "missing database url",
			config: &IndexerConfig{
				RPCEndpoint:  "http://localhost:3333",
				ChainID:      "ETH1",
				DatabaseType: "sqlite",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMetrics(t *testing.T) {
	metrics := NewIndexerMetrics()

	// Test initial values
	if metrics.GetBlocksIndexed() != 0 {
		t.Error("Initial blocks indexed should be 0")
	}

	if metrics.GetLogsIndexed() != 0 {
		t.Error("Initial logs indexed should be 0")
	}

	if metrics.GetTransactionsIndexed() != 0 {
		t.Error("Initial transactions indexed should be 0")
	}

	// Test incrementing
	metrics.IncBlocksIndexed()
	if metrics.GetBlocksIndexed() != 1 {
		t.Error("Blocks indexed should be 1")
	}

	metrics.IncLogsIndexed()
	if metrics.GetLogsIndexed() != 1 {
		t.Error("Logs indexed should be 1")
	}

	metrics.IncTransactionsIndexed()
	if metrics.GetTransactionsIndexed() != 1 {
		t.Error("Transactions indexed should be 1")
	}

	// Test setting last block
	metrics.SetLastIndexedBlock(12345)
	if metrics.GetLastIndexedBlock() != 12345 {
		t.Error("Last indexed block should be 12345")
	}

	// Test observing duration
	testDuration := 100 * time.Millisecond
	metrics.ObserveBlockIndexDuration(testDuration)
	if metrics.GetLastIndexDuration() != testDuration {
		t.Error("Last index duration should match")
	}

	// Test getting stats
	stats := metrics.GetStats()
	if stats == nil {
		t.Fatal("Stats should not be nil")
	}

	if stats["blocks_indexed"].(int64) != 1 {
		t.Error("Stats blocks_indexed should be 1")
	}
}

func TestParseHexToInt64(t *testing.T) {
	tests := []struct {
		name    string
		hexStr  string
		want    int64
		wantErr bool
	}{
		{"zero", "0x0", 0, false},
		{"empty", "", 0, false},
		{"just 0x", "0x", 0, false},
		{"small number", "0x10", 16, false},
		{"large number", "0x1234567", 19088743, false},
		{"block number", "0xf4240", 1000000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseHexToInt64(tt.hexStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseHexToInt64() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseHexToInt64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseStatus(t *testing.T) {
	tests := []struct {
		name      string
		statusHex string
		want      int
	}{
		{"success", "0x1", 1},
		{"failure", "0x0", 0},
		{"empty", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseStatus(tt.statusHex); got != tt.want {
				t.Errorf("parseStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSQLiteStorage(t *testing.T) {
	// Create a temporary database
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()

	// Test saving and retrieving a block
	block := &Block{
		ChainID:          "ETH1",
		BlockNumber:      12345,
		BlockHash:        "0xabc123",
		ParentHash:       "0xdef456",
		Timestamp:        time.Now().Unix(),
		Miner:            "0x123456",
		GasUsed:          21000,
		GasLimit:         30000,
		Difficulty:       "1000",
		TotalDifficulty:  "2000",
		Size:             500,
		TransactionCount: 1,
		ExtraData:        "0x",
		IndexedAt:        time.Now(),
	}

	err = storage.SaveBlock(ctx, block)
	if err != nil {
		t.Fatalf("Failed to save block: %v", err)
	}

	// Retrieve the block
	retrieved, err := storage.GetBlock(ctx, "ETH1", 12345)
	if err != nil {
		t.Fatalf("Failed to get block: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Retrieved block is nil")
	}

	if retrieved.BlockHash != block.BlockHash {
		t.Errorf("Block hash mismatch: got %s, want %s", retrieved.BlockHash, block.BlockHash)
	}

	// Test getting latest block
	latest, err := storage.GetLatestBlock(ctx, "ETH1")
	if err != nil {
		t.Fatalf("Failed to get latest block: %v", err)
	}

	if latest == nil {
		t.Fatal("Latest block is nil")
	}

	if latest.BlockNumber != 12345 {
		t.Errorf("Latest block number: got %d, want %d", latest.BlockNumber, 12345)
	}
}

func TestContractWatch(t *testing.T) {
	watch := &ContractWatch{
		ChainID:     "ETH1",
		Address:     "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
		Name:        "USDC",
		StartBlock:  6082465,
		EventFilter: []string{"0xddf252ad..."},
		Active:      true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if watch.ChainID != "ETH1" {
		t.Error("ChainID mismatch")
	}

	if watch.Name != "USDC" {
		t.Error("Name mismatch")
	}

	if !watch.Active {
		t.Error("Watch should be active")
	}
}
