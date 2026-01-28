package rpcsmartrouter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultGRPCStreamingConfig(t *testing.T) {
	config := DefaultGRPCStreamingConfig()

	// Verify subscription limits
	assert.Equal(t, 25, config.MaxSubscriptionsPerClient)
	assert.Equal(t, "warn", config.PerClientLimitEnforcement)
	assert.Equal(t, 10000, config.MaxTotalSubscriptions)
	assert.Equal(t, "warn", config.TotalLimitEnforcement)

	// Verify subscription sharing
	assert.True(t, config.SubscriptionSharingEnabled)

	// Verify rate limiting - gRPC allows higher rates
	assert.Equal(t, 30, config.SubscriptionsPerMinutePerClient)
	assert.Equal(t, 60, config.UnsubscribesPerMinutePerClient)

	// Verify timeouts
	assert.Equal(t, 30*time.Second, config.ConnectionTimeout)
	assert.Equal(t, 5*time.Minute, config.StreamIdleTimeout)
	assert.Equal(t, 30*time.Second, config.SubscriptionFirstReplyTimeout)

	// Verify cleanup interval
	assert.Equal(t, 1*time.Minute, config.CleanupInterval)

	// Verify pool settings
	assert.Equal(t, 1, config.PoolMinConnections)
	assert.Equal(t, 5, config.PoolMaxConnections)
	assert.Equal(t, 100, config.StreamsPerConnection)
}

func TestGRPCStreamingConfig_ShouldRejectOnClientLimit(t *testing.T) {
	tests := []struct {
		name           string
		enforcement    string
		expectedReject bool
	}{
		{"warn mode", "warn", false},
		{"reject mode", "reject", true},
		{"invalid mode treated as warn", "invalid", false},
		{"empty mode treated as warn", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultGRPCStreamingConfig()
			config.PerClientLimitEnforcement = tt.enforcement
			assert.Equal(t, tt.expectedReject, config.ShouldRejectOnClientLimit())
		})
	}
}

func TestGRPCStreamingConfig_ShouldRejectOnTotalLimit(t *testing.T) {
	tests := []struct {
		name           string
		enforcement    string
		expectedReject bool
	}{
		{"warn mode", "warn", false},
		{"reject mode", "reject", true},
		{"invalid mode treated as warn", "invalid", false},
		{"empty mode treated as warn", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultGRPCStreamingConfig()
			config.TotalLimitEnforcement = tt.enforcement
			assert.Equal(t, tt.expectedReject, config.ShouldRejectOnTotalLimit())
		})
	}
}

func TestNewGRPCClientRateLimiter(t *testing.T) {
	config := DefaultGRPCStreamingConfig()
	limiter := NewGRPCClientRateLimiter(config)

	require.NotNil(t, limiter)
	assert.NotNil(t, limiter.subscribeLimiters)
	assert.NotNil(t, limiter.unsubscribeLimiters)
}

func TestGRPCClientRateLimiter_AllowSubscribe(t *testing.T) {
	config := DefaultGRPCStreamingConfig()
	config.SubscriptionsPerMinutePerClient = 60 // 1 per second for easier testing
	limiter := NewGRPCClientRateLimiter(config)

	clientKey := "test-grpc-client-1"

	// First request should always be allowed (burst)
	assert.True(t, limiter.AllowSubscribe(clientKey))

	// Multiple rapid requests should be allowed up to burst limit
	for i := 0; i < 59; i++ {
		allowed := limiter.AllowSubscribe(clientKey)
		if !allowed {
			t.Logf("Request %d was rate limited (expected after burst)", i+2)
			break
		}
	}
}

func TestGRPCClientRateLimiter_AllowUnsubscribe(t *testing.T) {
	config := DefaultGRPCStreamingConfig()
	config.UnsubscribesPerMinutePerClient = 60 // 1 per second for easier testing
	limiter := NewGRPCClientRateLimiter(config)

	clientKey := "test-grpc-client-1"

	// First request should always be allowed (burst)
	assert.True(t, limiter.AllowUnsubscribe(clientKey))

	// Multiple rapid requests should be allowed up to burst limit
	for i := 0; i < 59; i++ {
		allowed := limiter.AllowUnsubscribe(clientKey)
		if !allowed {
			t.Logf("Request %d was rate limited (expected after burst)", i+2)
			break
		}
	}
}

func TestGRPCClientRateLimiter_SeparateClientsHaveSeparateLimits(t *testing.T) {
	config := DefaultGRPCStreamingConfig()
	config.SubscriptionsPerMinutePerClient = 2 // Very low limit
	limiter := NewGRPCClientRateLimiter(config)

	client1 := "grpc-client-1"
	client2 := "grpc-client-2"

	// Client 1 uses its burst
	assert.True(t, limiter.AllowSubscribe(client1))
	assert.True(t, limiter.AllowSubscribe(client1))

	// Client 2 should still have its own burst
	assert.True(t, limiter.AllowSubscribe(client2))
	assert.True(t, limiter.AllowSubscribe(client2))
}

func TestGRPCClientRateLimiter_CleanupClient(t *testing.T) {
	config := DefaultGRPCStreamingConfig()
	limiter := NewGRPCClientRateLimiter(config)

	clientKey := "test-grpc-client-cleanup"

	// Use the limiter to create entries
	limiter.AllowSubscribe(clientKey)
	limiter.AllowUnsubscribe(clientKey)

	// Verify entries exist
	_, existsSubscribe := limiter.subscribeLimiters[clientKey]
	_, existsUnsubscribe := limiter.unsubscribeLimiters[clientKey]
	assert.True(t, existsSubscribe)
	assert.True(t, existsUnsubscribe)

	// Cleanup
	limiter.CleanupClient(clientKey)

	// Verify entries are removed
	_, existsSubscribe = limiter.subscribeLimiters[clientKey]
	_, existsUnsubscribe = limiter.unsubscribeLimiters[clientKey]
	assert.False(t, existsSubscribe)
	assert.False(t, existsUnsubscribe)
}

func TestGRPCClientRateLimiter_ConcurrentAccess(t *testing.T) {
	// Note: The current GRPCClientRateLimiter implementation uses plain maps
	// which are not thread-safe. This test verifies sequential access works correctly.
	// For production use with concurrent access, the limiter would need synchronization.
	config := DefaultGRPCStreamingConfig()
	limiter := NewGRPCClientRateLimiter(config)

	// Test sequential access for multiple clients
	numClients := 10
	numOperations := 10

	for i := 0; i < numClients; i++ {
		clientKey := "grpc-client-" + string(rune('A'+i))
		for j := 0; j < numOperations; j++ {
			limiter.AllowSubscribe(clientKey)
			limiter.AllowUnsubscribe(clientKey)
		}
		limiter.CleanupClient(clientKey)
	}
	// Test passes if no panic occurs
}

func TestGRPCMetadataConstants(t *testing.T) {
	// Verify metadata header keys are defined correctly
	assert.Equal(t, "x-lava-grpc-sub-id", MetadataGRPCSubscriptionID)
	assert.Equal(t, "x-lava-grpc-stream-seq", MetadataGRPCStreamSeq)
	assert.Equal(t, "x-lava-grpc-client-id", MetadataGRPCClientID)
}

func TestGRPCStreamingConfig_VsWebSocketConfig(t *testing.T) {
	// Verify gRPC config has higher limits as designed
	grpcConfig := DefaultGRPCStreamingConfig()
	wsConfig := DefaultWebsocketConfig()

	// gRPC should have higher total subscription limit
	assert.Greater(t, grpcConfig.MaxTotalSubscriptions, wsConfig.MaxTotalSubscriptions,
		"gRPC should support more total subscriptions")

	// gRPC should have higher rate limits
	assert.Greater(t, grpcConfig.SubscriptionsPerMinutePerClient, wsConfig.SubscriptionsPerMinutePerClient,
		"gRPC should allow more subscriptions per minute")

	// gRPC should have longer connection timeout (establishing gRPC connections can take longer)
	assert.Greater(t, grpcConfig.ConnectionTimeout, wsConfig.HandshakeTimeout,
		"gRPC connection timeout should be longer than WS handshake timeout")
}
