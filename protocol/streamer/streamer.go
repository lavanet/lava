package streamer

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/lavanet/lava/v5/utils"
)

// Streamer is the main streaming service
type Streamer struct {
	config          *StreamerConfig
	rpcClient       *RPCClient
	abiDecoder      *ABIDecoder
	metrics         *StreamerMetrics
	subscriptionMgr *SubscriptionManager
	websocketServer *WebSocketServer
	webhookSender   *WebhookSender
	eventProcessor  *EventProcessor
}

// NewStreamer creates a new streamer instance
func NewStreamer(config *StreamerConfig) (*Streamer, error) {
	// Validate config
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Initialize RPC client
	rpcClient := NewRPCClient(config.RPCEndpoint, config.ChainID, config.MaxRetries, config.RetryDelay)

	// Initialize ABI decoder
	abiDecoder := NewABIDecoder()

	// Register ABIs for watched contracts
	for _, contract := range config.WatchedContracts {
		if contract.ABI != "" {
			if err := abiDecoder.RegisterABI(contract.Address, contract.ABI); err != nil {
				return nil, fmt.Errorf("failed to register ABI for %s: %w", contract.Name, err)
			}
		}
	}

	// Initialize metrics
	metrics := NewStreamerMetrics()

	// Initialize subscription manager
	subscriptionMgr := NewSubscriptionManager(metrics)

	// Initialize WebSocket server
	websocketServer := NewWebSocketServer(config, subscriptionMgr, metrics)

	// Initialize webhook sender
	webhookSender := NewWebhookSender(config, subscriptionMgr, metrics)

	// Initialize event processor
	eventProcessor := NewEventProcessor(
		config,
		rpcClient,
		abiDecoder,
		subscriptionMgr,
		websocketServer,
		webhookSender,
		metrics,
	)

	return &Streamer{
		config:          config,
		rpcClient:       rpcClient,
		abiDecoder:      abiDecoder,
		metrics:         metrics,
		subscriptionMgr: subscriptionMgr,
		websocketServer: websocketServer,
		webhookSender:   webhookSender,
		eventProcessor:  eventProcessor,
	}, nil
}

// Start starts all streamer components
func (s *Streamer) Start(ctx context.Context) error {
	utils.LavaFormatInfo("Starting Lava Event Streamer",
		utils.LogAttr("chainID", s.config.ChainID),
		utils.LogAttr("rpcEndpoint", s.config.RPCEndpoint),
	)

	// Test RPC connection
	blockNum, err := s.rpcClient.GetBlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to RPC endpoint: %w", err)
	}

	utils.LavaFormatInfo("Connected to RPC endpoint",
		utils.LogAttr("latestBlock", blockNum),
	)

	// Start subscription cleanup
	go s.subscriptionMgr.CleanupInactive(ctx)

	// Start WebSocket server
	if err := s.websocketServer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start WebSocket server: %w", err)
	}

	// Start webhook sender
	s.webhookSender.Start(ctx)

	// Start event processor
	if err := s.eventProcessor.Start(ctx); err != nil {
		return fmt.Errorf("failed to start event processor: %w", err)
	}

	utils.LavaFormatInfo("Streamer started successfully")

	// Wait for shutdown signal
	s.waitForShutdown(ctx)

	return nil
}

// Stop stops all streamer components
func (s *Streamer) Stop(ctx context.Context) error {
	utils.LavaFormatInfo("Stopping streamer...")

	// Stop components in reverse order
	s.eventProcessor.Stop()
	s.webhookSender.Stop()

	if err := s.websocketServer.Stop(ctx); err != nil {
		utils.LavaFormatError("Failed to stop WebSocket server", err)
	}

	utils.LavaFormatInfo("Streamer stopped")
	return nil
}

// waitForShutdown waits for interrupt signal
func (s *Streamer) waitForShutdown(ctx context.Context) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		utils.LavaFormatInfo("Received shutdown signal", utils.LogAttr("signal", sig.String()))
	case <-ctx.Done():
		utils.LavaFormatInfo("Context cancelled")
	}
}

// GetMetrics returns current metrics
func (s *Streamer) GetMetrics() *StreamerMetrics {
	return s.metrics
}

// GetSubscriptionManager returns the subscription manager
func (s *Streamer) GetSubscriptionManager() *SubscriptionManager {
	return s.subscriptionMgr
}

// RegisterContractABI registers an ABI for a contract address
func (s *Streamer) RegisterContractABI(address string, abiJSON string) error {
	return s.abiDecoder.RegisterABI(address, abiJSON)
}
