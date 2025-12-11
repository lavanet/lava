package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/lavanet/lava/v5/protocol/indexer"
	"github.com/lavanet/lava/v5/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func CreateIndexerCommand() *cobra.Command {
	var configFile string

	cmd := &cobra.Command{
		Use:   "indexer [config-file]",
		Short: "Run the EVM blockchain indexer service",
		Long: `The indexer service continuously indexes EVM blockchain data including blocks, transactions, and smart contract events.
It can be configured to work with the Lava smart router or any EVM JSON-RPC endpoint.

Features:
- Indexes blocks and transactions
- Monitors smart contract events
- Provides REST API for querying indexed data
- Supports SQLite, PostgreSQL (coming soon)
- Real-time metrics and monitoring`,
		Example: `# Run with config file
lavap indexer indexer.yml

# Run with default config
lavap indexer

# Generate example config
lavap indexer --example-config > indexer.yml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle example config generation
			if exampleConfig, _ := cmd.Flags().GetBool("example-config"); exampleConfig {
				return printExampleConfig()
			}

			// Determine config file
			if len(args) > 0 {
				configFile = args[0]
			} else if configFile == "" {
				configFile = "indexer.yml"
			}

			// Load configuration
			config, err := loadIndexerConfig(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Create and start indexer
			idx, err := indexer.NewIndexer(config)
			if err != nil {
				return fmt.Errorf("failed to create indexer: %w", err)
			}

			ctx := context.Background()
			return idx.Start(ctx)
		},
	}

	cmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to configuration file")
	cmd.Flags().Bool("example-config", false, "Print example configuration and exit")

	return cmd
}

// loadIndexerConfig loads the indexer configuration from a file
func loadIndexerConfig(configFile string) (*indexer.IndexerConfig, error) {
	// Check if file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		utils.LavaFormatWarning("Config file not found, using defaults", err,
			utils.LogAttr("configFile", configFile),
		)
		return indexer.DefaultConfig(), nil
	}

	// Read config file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config indexer.IndexerConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	utils.LavaFormatInfo("Loaded configuration from file", utils.LogAttr("file", configFile))
	return &config, nil
}

// printExampleConfig prints an example configuration
func printExampleConfig() error {
	config := &indexer.IndexerConfig{
		RPCEndpoint:  "http://localhost:3333",
		ChainID:      "ETH1",
		DatabaseType: "sqlite",
		DatabaseURL:  "indexer.db",

		StartBlock:         0,
		BatchSize:          10,
		ConcurrentWorkers:  4,
		PollingInterval:    12 * time.Second,
		ConfirmationBlocks: 0,
		ReorgDepth:         10,
		IndexTransactions:  true,
		IndexLogs:          true,
		IndexFullBlocks:    true,

		EnableAPI:     true,
		APIListenAddr: ":8080",
		EnableMetrics: true,
		MetricsAddr:   ":9090",

		WatchedContracts: []indexer.ContractWatchConfig{
			{
				Name:       "USDC",
				Address:    "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
				StartBlock: 6082465,
				EventFilter: []string{
					"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef", // Transfer event
				},
			},
		},

		CacheEnabled: true,
		CacheSize:    100,
		CacheTTL:     5 * time.Minute,
		MaxRetries:   3,
		RetryDelay:   1 * time.Second,
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	fmt.Println("# Lava EVM Indexer Configuration")
	fmt.Println("# Save this to a file (e.g., indexer.yml) and edit as needed")
	fmt.Println()
	fmt.Print(string(data))

	return nil
}
