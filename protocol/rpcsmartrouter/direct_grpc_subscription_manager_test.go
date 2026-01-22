package rpcsmartrouter

import (
	"context"
	"testing"
	"time"

	"github.com/lavanet/lava/v5/protocol/common"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDirectGRPCSubscriptionManager(t *testing.T) {
	chainID := "COSMOSHUB"
	apiInterface := "grpc"
	endpoints := []*common.NodeUrl{
		{Url: "grpc://localhost:9090"},
		{Url: "grpcs://example.com:443"},
	}

	manager := NewDirectGRPCSubscriptionManager(
		nil, // metricsManager
		chainID,
		apiInterface,
		endpoints,
		nil, // optimizer
		nil, // config (use defaults)
	)

	require.NotNil(t, manager)
	assert.Equal(t, chainID, manager.chainID)
	assert.Equal(t, apiInterface, manager.apiInterface)
	assert.Equal(t, 2, len(manager.grpcEndpoints))
	assert.NotNil(t, manager.activeSubscriptions)
	assert.NotNil(t, manager.pendingSubscriptions)
	assert.NotNil(t, manager.upstreamPools)
	assert.NotNil(t, manager.idMapper)
	assert.NotNil(t, manager.config)
	assert.NotNil(t, manager.rateLimiter)
	assert.NotNil(t, manager.stickySessions)
	assert.NotNil(t, manager.clientSubscriptions)
}

func TestNewDirectGRPCSubscriptionManager_WithConfig(t *testing.T) {
	config := &GRPCStreamingConfig{
		MaxSubscriptionsPerClient: 50,
		PerClientLimitEnforcement: "reject",
		MaxTotalSubscriptions:     20000,
		CleanupInterval:           2 * time.Minute,
	}

	manager := NewDirectGRPCSubscriptionManager(
		nil,
		"ETH",
		"grpc",
		[]*common.NodeUrl{{Url: "grpc://localhost:9090"}},
		nil,
		config,
	)

	require.NotNil(t, manager)
	assert.Equal(t, config, manager.config)
	assert.Equal(t, 50, manager.config.MaxSubscriptionsPerClient)
	assert.Equal(t, "reject", manager.config.PerClientLimitEnforcement)
}

func TestNewDirectGRPCSubscriptionManager_EmptyEndpoints(t *testing.T) {
	manager := NewDirectGRPCSubscriptionManager(
		nil,
		"ETH",
		"grpc",
		[]*common.NodeUrl{}, // Empty endpoints
		nil,
		nil,
	)

	require.NotNil(t, manager)
	assert.Empty(t, manager.grpcEndpoints)
	assert.Empty(t, manager.endpointsByURL)
}

func TestDirectGRPCSubscriptionManager_EndpointLookup(t *testing.T) {
	endpoints := []*common.NodeUrl{
		{Url: "grpc://localhost:9090"},
		{Url: "grpcs://node1.example.com:443"},
		{Url: "grpcs://node2.example.com:443"},
	}

	manager := NewDirectGRPCSubscriptionManager(
		nil,
		"COSMOSHUB",
		"grpc",
		endpoints,
		nil,
		nil,
	)

	// Verify endpoint lookup map is populated
	assert.Equal(t, 3, len(manager.endpointsByURL))
	assert.NotNil(t, manager.endpointsByURL["grpc://localhost:9090"])
	assert.NotNil(t, manager.endpointsByURL["grpcs://node1.example.com:443"])
	assert.NotNil(t, manager.endpointsByURL["grpcs://node2.example.com:443"])
}

func TestDirectGRPCSubscriptionManager_StartStop(t *testing.T) {
	manager := NewDirectGRPCSubscriptionManager(
		nil,
		"ETH",
		"grpc",
		[]*common.NodeUrl{{Url: "grpc://localhost:9090"}},
		nil,
		nil,
	)

	ctx := context.Background()

	// Start should not panic
	manager.Start(ctx)

	// Give cleanup goroutine time to start
	time.Sleep(10 * time.Millisecond)

	// Stop should not panic and should clean up
	manager.Stop()

	// Verify state is cleaned up
	manager.lock.RLock()
	defer manager.lock.RUnlock()
	assert.Empty(t, manager.activeSubscriptions)
	assert.Empty(t, manager.upstreamPools)
}

func TestDirectGRPCSubscriptionManager_Stop_Idempotent(t *testing.T) {
	manager := NewDirectGRPCSubscriptionManager(
		nil,
		"ETH",
		"grpc",
		[]*common.NodeUrl{{Url: "grpc://localhost:9090"}},
		nil,
		nil,
	)

	manager.Start(context.Background())

	// Multiple stops should not panic
	manager.Stop()
	manager.Stop()
	manager.Stop()
}

func TestDirectGRPCSubscriptionManager_TotalSubscriptions(t *testing.T) {
	manager := NewDirectGRPCSubscriptionManager(
		nil,
		"ETH",
		"grpc",
		[]*common.NodeUrl{{Url: "grpc://localhost:9090"}},
		nil,
		nil,
	)

	// Initially should be 0
	assert.Equal(t, int64(0), manager.totalSubscriptions.Load())

	// Simulate subscription count changes
	manager.totalSubscriptions.Add(5)
	assert.Equal(t, int64(5), manager.totalSubscriptions.Load())

	manager.totalSubscriptions.Add(-2)
	assert.Equal(t, int64(3), manager.totalSubscriptions.Load())
}

func TestDirectGRPCSubscriptionManager_RateLimiter(t *testing.T) {
	config := &GRPCStreamingConfig{
		SubscriptionsPerMinutePerClient: 2, // Very low for testing
		UnsubscribesPerMinutePerClient:  2,
	}

	manager := NewDirectGRPCSubscriptionManager(
		nil,
		"ETH",
		"grpc",
		[]*common.NodeUrl{{Url: "grpc://localhost:9090"}},
		nil,
		config,
	)

	clientKey := "test-client"

	// Should allow first few requests (burst)
	assert.True(t, manager.rateLimiter.AllowSubscribe(clientKey))
	assert.True(t, manager.rateLimiter.AllowSubscribe(clientKey))

	// Third should be rate limited
	assert.False(t, manager.rateLimiter.AllowSubscribe(clientKey))
}

func TestDirectGRPCSubscriptionManager_IDMapper(t *testing.T) {
	manager := NewDirectGRPCSubscriptionManager(
		nil,
		"ETH",
		"grpc",
		[]*common.NodeUrl{{Url: "grpc://localhost:9090"}},
		nil,
		nil,
	)

	// ID mapper should be initialized
	require.NotNil(t, manager.idMapper)

	// Test basic ID generation
	clientKey := "client-1"
	routerID := manager.idMapper.GenerateRouterID(clientKey)
	assert.NotEmpty(t, routerID)
	assert.Contains(t, routerID, "rs_") // Router ID format prefix

	// Generate another ID for same client - should be different
	routerID2 := manager.idMapper.GenerateRouterID(clientKey)
	assert.NotEqual(t, routerID, routerID2)

	// Test mapping functionality
	upstreamID := "upstream-sub-123"
	manager.idMapper.RegisterMapping(routerID, upstreamID)

	// Retrieve mapping
	retrievedUpstreamID, found := manager.idMapper.GetUpstreamID(routerID)
	assert.True(t, found)
	assert.Equal(t, upstreamID, retrievedUpstreamID)

	// Get router IDs for upstream
	routerIDs := manager.idMapper.GetRouterIDs(upstreamID)
	assert.Contains(t, routerIDs, routerID)

	// Non-existent mapping should return false
	_, found = manager.idMapper.GetUpstreamID("non-existent-router-id")
	assert.False(t, found)
}

func TestDirectGRPCSubscriptionManager_StickySessions(t *testing.T) {
	manager := NewDirectGRPCSubscriptionManager(
		nil,
		"ETH",
		"grpc",
		[]*common.NodeUrl{
			{Url: "grpc://node1:9090"},
			{Url: "grpc://node2:9090"},
		},
		nil,
		nil,
	)

	// Initially no sticky sessions
	assert.Empty(t, manager.stickySessions)

	// Simulate setting a sticky session
	manager.lock.Lock()
	manager.stickySessions["client-1"] = "grpc://node1:9090"
	manager.stickySessions["client-2"] = "grpc://node2:9090"
	manager.lock.Unlock()

	manager.lock.RLock()
	assert.Equal(t, "grpc://node1:9090", manager.stickySessions["client-1"])
	assert.Equal(t, "grpc://node2:9090", manager.stickySessions["client-2"])
	manager.lock.RUnlock()
}

func TestDirectGRPCSubscriptionManager_ClientSubscriptionTracking(t *testing.T) {
	manager := NewDirectGRPCSubscriptionManager(
		nil,
		"ETH",
		"grpc",
		[]*common.NodeUrl{{Url: "grpc://localhost:9090"}},
		nil,
		nil,
	)

	clientKey := "client-1"

	// Initially no subscriptions for client
	manager.lock.RLock()
	_, exists := manager.clientSubscriptions[clientKey]
	manager.lock.RUnlock()
	assert.False(t, exists)

	// Simulate adding subscriptions for a client
	manager.lock.Lock()
	manager.clientSubscriptions[clientKey] = make(map[string]struct{})
	manager.clientSubscriptions[clientKey]["hash1"] = struct{}{}
	manager.clientSubscriptions[clientKey]["hash2"] = struct{}{}
	manager.lock.Unlock()

	// Verify
	manager.lock.RLock()
	subs := manager.clientSubscriptions[clientKey]
	manager.lock.RUnlock()
	assert.Equal(t, 2, len(subs))

	// Clean up client subscriptions
	manager.lock.Lock()
	delete(manager.clientSubscriptions, clientKey)
	manager.lock.Unlock()

	manager.lock.RLock()
	_, exists = manager.clientSubscriptions[clientKey]
	manager.lock.RUnlock()
	assert.False(t, exists)
}

func TestDirectGRPCSubscriptionManager_ContextCancellation(t *testing.T) {
	manager := NewDirectGRPCSubscriptionManager(
		nil,
		"ETH",
		"grpc",
		[]*common.NodeUrl{{Url: "grpc://localhost:9090"}},
		nil,
		nil,
	)

	// Start with a cancelable context
	ctx, cancel := context.WithCancel(context.Background())
	manager.Start(ctx)

	// Give cleanup goroutine time to start
	time.Sleep(10 * time.Millisecond)

	// Cancel the context
	cancel()

	// Give time for cleanup goroutine to exit
	time.Sleep(50 * time.Millisecond)

	// Manager should still be usable (has its own internal context)
	// but the external context cancellation should trigger cleanup loop exit
}

func TestGRPCActiveSubscription_Initialization(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sub := &grpcActiveSubscription{
		routerSubscriptionID: "router-123",
		hashedParams:         "hash-abc",
		methodPath:           "/cosmos.bank.v1beta1.Query/Balance",
		requestParams:        []byte(`{"address":"cosmos1..."}`),
		clientRouterIDs:      make(map[string]string),
		connectedClients:     make(map[string]*common.SafeChannelSender[*pairingtypes.RelayReply]),
		ctx:                  ctx,
		cancel:               cancel,
		closeSubChan:         make(chan struct{}),
	}

	assert.Equal(t, "router-123", sub.routerSubscriptionID)
	assert.Equal(t, "hash-abc", sub.hashedParams)
	assert.Equal(t, "/cosmos.bank.v1beta1.Query/Balance", sub.methodPath)
	assert.NotNil(t, sub.clientRouterIDs)
	assert.NotNil(t, sub.connectedClients)
	assert.False(t, sub.restoring.Load())
	assert.Equal(t, uint64(0), sub.messageSeq.Load())
}

func TestGRPCActiveSubscription_MessageSequence(t *testing.T) {
	sub := &grpcActiveSubscription{}

	// Initial sequence should be 0
	assert.Equal(t, uint64(0), sub.messageSeq.Load())

	// Increment sequence
	seq1 := sub.messageSeq.Add(1)
	assert.Equal(t, uint64(1), seq1)

	seq2 := sub.messageSeq.Add(1)
	assert.Equal(t, uint64(2), seq2)
}

func TestGRPCActiveSubscription_RestorationFlag(t *testing.T) {
	sub := &grpcActiveSubscription{}

	// Initially not restoring
	assert.False(t, sub.restoring.Load())

	// Mark as restoring
	sub.restoring.Store(true)
	assert.True(t, sub.restoring.Load())

	// Clear restoration flag
	sub.restoring.Store(false)
	assert.False(t, sub.restoring.Load())
}

func TestDirectGRPCSubscriptionManager_DefaultConfig(t *testing.T) {
	manager := NewDirectGRPCSubscriptionManager(
		nil,
		"ETH",
		"grpc",
		[]*common.NodeUrl{{Url: "grpc://localhost:9090"}},
		nil,
		nil, // nil config should use defaults
	)

	// Verify default config is applied
	assert.Equal(t, 25, manager.config.MaxSubscriptionsPerClient)
	assert.Equal(t, 10000, manager.config.MaxTotalSubscriptions)
	assert.Equal(t, "warn", manager.config.PerClientLimitEnforcement)
	assert.True(t, manager.config.SubscriptionSharingEnabled)
}
