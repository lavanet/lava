package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/lavanet/lava/v5/protocol/streamer"
	"github.com/lavanet/lava/v5/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func CreateStreamerCommand() *cobra.Command {
	var configFile string

	cmd := &cobra.Command{
		Use:   "streamer [config-file]",
		Short: "Run the EVM blockchain streaming service",
		Long: `The streamer service provides real-time blockchain event streaming via WebSockets and webhooks.
Unlike traditional indexers, the streamer is stateless and focuses on real-time event delivery.

Features:
- Real-time block and transaction streaming
- WebSocket subscriptions
- Webhook notifications
- Event decoding (ERC20, ERC721, custom ABIs)
- Internal transaction tracking
- Stateless architecture (no database)
- Optional in-memory cache for recent data`,
		Example: `# Run with config file
lavap streamer streamer.yml

# Run with default config
lavap streamer

# Generate example config
lavap streamer --example-config > streamer.yml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Handle example config generation
			if exampleConfig, _ := cmd.Flags().GetBool("example-config"); exampleConfig {
				return printExampleStreamerConfig()
			}

			// Determine config file
			if len(args) > 0 {
				configFile = args[0]
			} else if configFile == "" {
				configFile = "streamer.yml"
			}

			// Load configuration
			config, err := loadStreamerConfig(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Create and start streamer
			s, err := streamer.NewStreamer(config)
			if err != nil {
				return fmt.Errorf("failed to create streamer: %w", err)
			}

			ctx := context.Background()
			return s.Start(ctx)
		},
	}

	cmd.Flags().StringVarP(&configFile, "config", "c", "", "Path to configuration file")
	cmd.Flags().Bool("example-config", false, "Print example configuration and exit")

	return cmd
}

// loadStreamerConfig loads the streamer configuration from a file
func loadStreamerConfig(configFile string) (*streamer.StreamerConfig, error) {
	// Check if file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		utils.LavaFormatWarning("Config file not found, using defaults", err,
			utils.LogAttr("configFile", configFile),
		)
		return streamer.DefaultConfig(), nil
	}

	// Read config file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config streamer.StreamerConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	utils.LavaFormatInfo("Loaded configuration from file", utils.LogAttr("file", configFile))
	return &config, nil
}

// printExampleStreamerConfig prints an example configuration
func printExampleStreamerConfig() error {
	config := &streamer.StreamerConfig{
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
		EnableMessageQueue:    false,
		MessageQueueType:      "kafka",
		MessageQueueAddr:      "localhost:9092",
		MessageQueueTopic:     "lava-events",
		EnableCache:           true,
		CacheType:             "memory",
		CacheMaxBlocks:        1000,
		CacheTTL:              10 * time.Minute,
		EnableAPI:             true,
		APIListenAddr:         ":8081",
		EnableMetrics:         true,
		MetricsAddr:           ":9090",
		WatchedContracts: []streamer.ContractWatchConfig{
			{
				Name:       "USDC",
				Address:    "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
				StartBlock: 6082465,
				EventFilter: []string{
					"0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
				},
				StreamAllTxs: true,
			},
		},
		MaxRetries: 3,
		RetryDelay: 1 * time.Second,
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	fmt.Println("# Lava Event Streamer Configuration")
	fmt.Println("# Save this to a file (e.g., streamer.yml) and edit as needed")
	fmt.Println()
	fmt.Print(string(data))

	return nil
}
